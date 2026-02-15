package pg

import (
	"context"
	"distributed-crawler/internal/infra/persistence"
	"fmt"
	"sync"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// shardedClient holds multiple database pools, one per shard.
type shardedClient struct {
	shards []persistence.DB
}

// NewSharded creates a sharded client with a connection pool per DSN.
func NewSharded(ctx context.Context, dsns []string) (persistence.Client, error) {
	if len(dsns) == 0 {
		return nil, fmt.Errorf("at least one shard DSN is required")
	}

	shards := make([]persistence.DB, len(dsns))
	for i, dsn := range dsns {
		pool, err := pgxpool.Connect(ctx, dsn)
		if err != nil {
			// close already opened pools
			for j := 0; j < i; j++ {
				shards[j].Close()
			}
			return nil, fmt.Errorf("failed to connect to shard %d: %w", i, err)
		}
		shards[i] = &pgDb{dbc: pool}
	}

	return &shardedClient{shards: shards}, nil
}

func (c *shardedClient) DB() persistence.DB {
	return &shardedDB{shards: c.shards}
}

func (c *shardedClient) Close() error {
	for _, s := range c.shards {
		s.Close()
	}
	return nil
}

// shardedDB routes every operation to the correct shard based on
// the shard key stored in the context.
type shardedDB struct {
	shards []persistence.DB
}

func (s *shardedDB) resolve(ctx context.Context) persistence.DB {
	key, ok := persistence.ShardKeyFromContext(ctx)
	if !ok {
		return s.shards[0]
	}
	idx := persistence.ShardIndex(key, len(s.shards))
	return s.shards[idx]
}

func (s *shardedDB) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return s.resolve(ctx).BeginTx(ctx, txOptions)
}

func (s *shardedDB) Close() {
	for _, shard := range s.shards {
		shard.Close()
	}
}

func (s *shardedDB) Ping(ctx context.Context) error {
	for i, shard := range s.shards {
		if err := shard.Ping(ctx); err != nil {
			return fmt.Errorf("shard %d ping failed: %w", i, err)
		}
	}
	return nil
}

func (s *shardedDB) ExecContext(ctx context.Context, q persistence.Query, args ...any) (pgconn.CommandTag, error) {
	return s.resolve(ctx).ExecContext(ctx, q, args...)
}

func (s *shardedDB) QueryContext(ctx context.Context, q persistence.Query, args ...any) (pgx.Rows, error) {
	_, ok := persistence.ShardKeyFromContext(ctx)
	if ok || len(s.shards) == 1 {
		return s.resolve(ctx).QueryContext(ctx, q, args...)
	}
	return s.fanoutQuery(ctx, q, args...)
}

func (s *shardedDB) QueryRowContext(ctx context.Context, q persistence.Query, args ...any) pgx.Row {
	return s.resolve(ctx).QueryRowContext(ctx, q, args...)
}

func (s *shardedDB) ScanOneContext(ctx context.Context, dest any, q persistence.Query, args ...any) error {
	return s.resolve(ctx).ScanOneContext(ctx, dest, q, args...)
}

func (s *shardedDB) ScanAllContext(ctx context.Context, dest any, q persistence.Query, args ...any) error {
	_, ok := persistence.ShardKeyFromContext(ctx)
	if ok || len(s.shards) == 1 {
		return s.resolve(ctx).ScanAllContext(ctx, dest, q, args...)
	}
	return s.fanoutScanAll(ctx, dest, q, args...)
}

// fanoutQuery executes a query on all shards and returns a merged row set.
func (s *shardedDB) fanoutQuery(ctx context.Context, q persistence.Query, args ...any) (pgx.Rows, error) {
	type result struct {
		rows pgx.Rows
		err  error
	}

	results := make([]result, len(s.shards))
	var wg sync.WaitGroup

	for i, shard := range s.shards {
		wg.Add(1)
		go func(idx int, db persistence.DB) {
			defer wg.Done()
			rows, err := db.QueryContext(ctx, q, args...)
			results[idx] = result{rows: rows, err: err}
		}(i, shard)
	}
	wg.Wait()

	// Return first non-nil rows found; collect errors.
	var firstRows pgx.Rows
	var allRows []pgx.Rows
	for _, r := range results {
		if r.err != nil {
			// Close any rows we already collected.
			for _, rows := range allRows {
				rows.Close()
			}
			return nil, r.err
		}
		allRows = append(allRows, r.rows)
		if firstRows == nil {
			firstRows = r.rows
		}
	}

	return &mergedRows{sets: allRows}, nil
}

// fanoutScanAll executes ScanAll on every shard and appends results.
func (s *shardedDB) fanoutScanAll(ctx context.Context, dest any, q persistence.Query, args ...any) error {
	type result struct {
		rows pgx.Rows
		err  error
	}

	results := make([]result, len(s.shards))
	var wg sync.WaitGroup

	for i, shard := range s.shards {
		wg.Add(1)
		go func(idx int, db persistence.DB) {
			defer wg.Done()
			rows, err := db.QueryContext(ctx, q, args...)
			results[idx] = result{rows: rows, err: err}
		}(i, shard)
	}
	wg.Wait()

	var allRows []pgx.Rows
	for _, r := range results {
		if r.err != nil {
			for _, rows := range allRows {
				rows.Close()
			}
			return r.err
		}
		allRows = append(allRows, r.rows)
	}

	merged := &mergedRows{sets: allRows}
	defer merged.Close()

	return pgxscan.ScanAll(dest, merged)
}
