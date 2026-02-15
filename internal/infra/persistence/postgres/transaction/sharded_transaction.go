package transaction

import (
	"context"
	"distributed-crawler/internal/infra/persistence"
	"fmt"
)

// shardedManager routes transactions to the correct shard based on
// the shard key in the context.
type shardedManager struct {
	managers []persistence.TxManager
}

// NewShardedTransactorManager creates a TxManager that wraps one TxManager per shard.
func NewShardedTransactorManager(transactors []persistence.Transactor) persistence.TxManager {
	managers := make([]persistence.TxManager, len(transactors))
	for i, t := range transactors {
		managers[i] = NewTransactorManager(t)
	}
	return &shardedManager{managers: managers}
}

func (m *shardedManager) ReadCommitted(ctx context.Context, exec persistence.Handler) error {
	key, ok := persistence.ShardKeyFromContext(ctx)
	if !ok {
		return m.managers[0].ReadCommitted(ctx, exec)
	}

	idx := persistence.ShardIndex(key, len(m.managers))
	if idx < 0 || idx >= len(m.managers) {
		return fmt.Errorf("shard index %d out of range (total shards: %d)", idx, len(m.managers))
	}

	return m.managers[idx].ReadCommitted(ctx, exec)
}
