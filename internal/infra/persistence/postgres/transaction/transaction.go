package transaction

import (
	"context"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/pg"
	"fmt"

	"github.com/jackc/pgx/v4"
)

type manager struct {
	db persistence.Transactor
}

func NewTransactorManager(transactor persistence.Transactor) persistence.TxManager {
	return &manager{
		db: transactor,
	}
}

func (m *manager) ReadCommitted(ctx context.Context, exec persistence.Handler) error {
	txOpts := pgx.TxOptions{IsoLevel: pgx.ReadCommitted}
	return m.transaction(ctx, txOpts, exec)
}

func (m *manager) transaction(ctx context.Context, opts pgx.TxOptions, fn persistence.Handler) (err error) {
	// if some transaction is already exist
	tx, ok := ctx.Value(pg.TxKey).(pgx.Tx)
	if ok {
		return fn(ctx)
	}

	tx, err = m.db.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("cant begin transaction: %w", err)
	}

	ctx = makeContextTx(ctx, tx)

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered: %v", r)
		}

		if err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				err = fmt.Errorf("%w: errRollback: %v", err, errRollback)
			}

			return
		}

		if err == nil {
			if err = tx.Commit(ctx); err != nil {
				err = fmt.Errorf("tx commit failed: %w", err)
			}
		}
	}()

	if err = fn(ctx); err != nil {
		err = fmt.Errorf("failed executing code inside transaction: %w", err)
	}

	return err
}

func makeContextTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, pg.TxKey, tx)
}
