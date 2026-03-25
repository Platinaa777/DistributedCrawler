package routing

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	queuemodels "distributed-crawler/internal/domain/queue/models"
	"distributed-crawler/internal/infra/persistence"

	"github.com/redis/go-redis/v9"
)

// QueueRoutingPolicy selects a queue name for a given job and pipeline stage.
type QueueRoutingPolicy interface {
	// SelectQueue returns the queue/topic name for the given job ID and stage.
	// Falls back to fallback if no assignment exists for the job.
	SelectQueue(ctx context.Context, jobID string, stage queuemodels.Stage) (string, error)
}

// endpointEntry holds a resolved queue name and its weight.
type endpointEntry struct {
	queueName string
	weight    int32
}

// JobQueueLoader loads queue endpoint assignments for a job from the DB.
type JobQueueLoader interface {
	LoadAssignments(ctx context.Context, jobID string, stage queuemodels.Stage) ([]endpointEntry, error)
}

// -- In-memory weighted-random routing policy --

type inMemoryRoutingPolicy struct {
	loader        JobQueueLoader
	fallbackCrawl string
	fallbackParse string

	mu       sync.Mutex
	cache    map[string][]endpointEntry // keyed by "jobID:stage"
	cachedAt map[string]time.Time
	cacheTTL time.Duration
}

// NewInMemoryRoutingPolicy creates a job-aware routing policy using weighted random selection.
// It caches per-job assignments for cacheTTL (default 30s).
func NewInMemoryRoutingPolicy(
	loader JobQueueLoader,
	fallbackCrawl string,
	fallbackParse string,
) QueueRoutingPolicy {
	return &inMemoryRoutingPolicy{
		loader:        loader,
		fallbackCrawl: fallbackCrawl,
		fallbackParse: fallbackParse,
		cache:         make(map[string][]endpointEntry),
		cachedAt:      make(map[string]time.Time),
		cacheTTL:      30 * time.Second,
	}
}

func (p *inMemoryRoutingPolicy) SelectQueue(ctx context.Context, jobID string, stage queuemodels.Stage) (string, error) {
	entries, err := p.getEntries(ctx, jobID, stage)
	if err != nil || len(entries) == 0 {
		return p.fallback(stage), nil //nolint:nilerr
	}

	return weightedRandom(entries), nil
}

func (p *inMemoryRoutingPolicy) getEntries(ctx context.Context, jobID string, stage queuemodels.Stage) ([]endpointEntry, error) {
	key := jobID + ":" + string(stage)

	p.mu.Lock()
	defer p.mu.Unlock()

	if t, ok := p.cachedAt[key]; ok && time.Since(t) < p.cacheTTL {
		return p.cache[key], nil
	}

	entries, err := p.loader.LoadAssignments(ctx, jobID, stage)
	if err != nil {
		return nil, fmt.Errorf("routing: load assignments for job %s stage %s: %w", jobID, stage, err)
	}

	p.cache[key] = entries
	p.cachedAt[key] = time.Now()
	return entries, nil
}

func (p *inMemoryRoutingPolicy) fallback(stage queuemodels.Stage) string {
	switch stage {
	case queuemodels.StageCrawl:
		return p.fallbackCrawl
	case queuemodels.StageParse:
		return p.fallbackParse
	default:
		return ""
	}
}

// weightedRandom picks an entry using weighted random selection.
func weightedRandom(entries []endpointEntry) string {
	var total int32
	for _, e := range entries {
		total += e.weight
	}
	if total <= 0 {
		return entries[0].queueName
	}

	pick := rand.Int31n(total)
	var cumulative int32
	for _, e := range entries {
		cumulative += e.weight
		if pick < cumulative {
			return e.queueName
		}
	}
	return entries[len(entries)-1].queueName
}

// -- Redis weighted round-robin routing policy --

// redisRoutingPolicy uses Redis HINCRBY counters for weighted round-robin.
// Each job has a hash key "qlb:{jobID}" with fields "{stage}:{queueName}" → counter.
// A Lua script atomically finds the queue with the lowest ratio counter/weight.
type redisRoutingPolicy struct {
	inMemory  *inMemoryRoutingPolicy
	rdb       *redis.Client
	keyPrefix string
}

// NewRedisRoutingPolicy creates a job-aware routing policy backed by Redis counters.
// Falls back to weighted random if Redis is unavailable.
func NewRedisRoutingPolicy(
	loader JobQueueLoader,
	rdb *redis.Client,
	fallbackCrawl string,
	fallbackParse string,
) QueueRoutingPolicy {
	mem := &inMemoryRoutingPolicy{
		loader:        loader,
		fallbackCrawl: fallbackCrawl,
		fallbackParse: fallbackParse,
		cache:         make(map[string][]endpointEntry),
		cachedAt:      make(map[string]time.Time),
		cacheTTL:      30 * time.Second,
	}
	return &redisRoutingPolicy{
		inMemory:  mem,
		rdb:       rdb,
		keyPrefix: "qlb:",
	}
}

// luaWeightedRR atomically increments the counter for the chosen queue and returns its name.
// It picks the queue with the minimum (counter / weight) ratio.
// KEYS[1] = hash key for the job
// ARGV = pairs of (queueName, weight) for each candidate
var luaWeightedRR = redis.NewScript(`
local key = KEYS[1]
local best = nil
local bestRatio = nil
local i = 1
while i <= #ARGV do
  local name = ARGV[i]
  local w = tonumber(ARGV[i+1])
  local cnt = tonumber(redis.call('HGET', key, name) or 0)
  if cnt == nil then cnt = 0 end
  local ratio = cnt / w
  if bestRatio == nil or ratio < bestRatio then
    best = name
    bestRatio = ratio
  end
  i = i + 2
end
redis.call('HINCRBY', key, best, 1)
redis.call('EXPIRE', key, 3600)
return best
`)

func (p *redisRoutingPolicy) SelectQueue(ctx context.Context, jobID string, stage queuemodels.Stage) (string, error) {
	entries, err := p.inMemory.getEntries(ctx, jobID, stage)
	if err != nil || len(entries) == 0 {
		return p.inMemory.fallback(stage), nil //nolint:nilerr
	}

	if len(entries) == 1 {
		return entries[0].queueName, nil
	}

	// Build ARGV pairs for Lua script
	argv := make([]interface{}, 0, len(entries)*2)
	for _, e := range entries {
		argv = append(argv, e.queueName, e.weight)
	}

	hashKey := p.keyPrefix + jobID + ":" + strings.ToLower(string(stage))
	result, err := luaWeightedRR.Run(ctx, p.rdb, []string{hashKey}, argv...).Text()
	if err != nil {
		// Redis unavailable — fall back to weighted random
		return weightedRandom(entries), nil
	}

	return result, nil
}

// -- DB-backed JobQueueLoader --

// DBJobQueueLoader loads endpoint assignments via a raw SQL join.
type DBJobQueueLoader struct {
	db persistence.DB
}

// NewDBJobQueueLoader creates a loader backed by the postgres persistence client.
func NewDBJobQueueLoader(db persistence.DB) JobQueueLoader {
	return &DBJobQueueLoader{db: db}
}

const loadAssignmentsSQL = `
SELECT qe.queue_name, cjcqe.weight
FROM crawl_job_config_queue_endpoints cjcqe
JOIN queue_endpoints qe ON qe.id = cjcqe.queue_endpoint_id
JOIN crawl_job_configs cjc ON cjc.id = cjcqe.crawl_job_config_id
JOIN crawl_jobs cj ON cj.job_config_id = cjc.id
WHERE cj.id = $1
  AND qe.stage = $2
`

func (l *DBJobQueueLoader) LoadAssignments(ctx context.Context, jobID string, stage queuemodels.Stage) ([]endpointEntry, error) {
	rows, err := l.db.QueryContext(ctx, persistence.Query{
		Name:     "DBJobQueueLoader.LoadAssignments",
		QueryRaw: loadAssignmentsSQL,
	}, jobID, string(stage))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []endpointEntry
	for rows.Next() {
		var e endpointEntry
		if err := rows.Scan(&e.queueName, &e.weight); err != nil {
			return nil, err
		}
		if e.weight <= 0 {
			e.weight = 1
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
