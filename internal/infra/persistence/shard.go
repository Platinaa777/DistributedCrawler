package persistence

import (
	"context"
	"hash/fnv"
)

type shardKey string

const ctxShardKey shardKey = "shard_key"

// WithShardKey injects a shard routing key (typically crawl_job_id) into the context.
func WithShardKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, ctxShardKey, key)
}

// ShardKeyFromContext extracts the shard routing key from the context.
func ShardKeyFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxShardKey).(string)
	return v, ok && v != ""
}

// ShardIndex computes a deterministic shard index for the given key.
func ShardIndex(key string, numShards int) int {
	if numShards <= 1 {
		return 0
	}
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() % uint32(numShards))
}
