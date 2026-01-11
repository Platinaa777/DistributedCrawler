package persistence

import "context"

type TxManager interface {
	ReadCommitted(ctx context.Context, exec Handler) error
}

type Handler func(ctx context.Context) error
