package pg

import (
	"context"
	"distributed-crawler/internal/infra/persistence"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type key string

const (
	TxKey key = "tx"
)

type pgDb struct {
	dbc *pgxpool.Pool
}

func NewDB(dbc *pgxpool.Pool) persistence.DB {
	return &pgDb{
		dbc: dbc,
	}
}

func (p *pgDb) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return p.dbc.BeginTx(ctx, txOptions)
}

func (p *pgDb) Close() {
	p.dbc.Close()
}

func (p *pgDb) ExecContext(ctx context.Context, q persistence.Query, args ...any) (pgconn.CommandTag, error) {
	logQuery(ctx, q, args...)

	tx, ok := ctx.Value(TxKey).(pgx.Tx)
	if ok {
		return tx.Exec(ctx, q.QueryRaw, args...)
	}

	return p.dbc.Exec(ctx, q.QueryRaw, args...)
}

func (p *pgDb) QueryContext(ctx context.Context, q persistence.Query, args ...any) (pgx.Rows, error) {
	logQuery(ctx, q, args...)

	tx, ok := ctx.Value(TxKey).(pgx.Tx)
	if ok {
		return tx.Query(ctx, q.QueryRaw, args...)
	}

	return p.dbc.Query(ctx, q.QueryRaw, args...)
}

func (p *pgDb) Ping(ctx context.Context) error {
	return p.dbc.Ping(ctx)
}

func (p *pgDb) QueryRowContext(ctx context.Context, q persistence.Query, args ...any) pgx.Row {
	logQuery(ctx, q, args...)

	tx, ok := ctx.Value(TxKey).(pgx.Tx)
	if ok {
		return tx.QueryRow(ctx, q.QueryRaw, args...)
	}

	return p.dbc.QueryRow(ctx, q.QueryRaw, args...)
}

func (p *pgDb) ScanAllContext(ctx context.Context, dest any, q persistence.Query, args ...any) error {
	logQuery(ctx, q, args...)

	rows, err := p.QueryContext(ctx, q, args...)
	if err != nil {
		return err
	}

	return pgxscan.ScanAll(dest, rows)
}

func (p *pgDb) ScanOneContext(ctx context.Context, dest any, q persistence.Query, args ...any) error {
	logQuery(ctx, q, args...)

	row, err := p.QueryContext(ctx, q, args...)
	if err != nil {
		return err
	}

	return pgxscan.ScanOne(dest, row)
}

func logQuery(ctx context.Context, q persistence.Query, args ...any) {
	// log.Println(
	// 	ctx,
	// 	fmt.Sprintf("sql: %s", q.Name),
	// 	fmt.Sprintf("query: %s", q.QueryRaw),
	// )
}
