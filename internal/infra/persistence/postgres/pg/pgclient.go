package pg

import (
	"context"
	"distributed-crawler/internal/infra/persistence"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
)

type pgClient struct {
	database persistence.DB
}

func New(ctx context.Context, dsn string) (persistence.Client, error) {
	dbc, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db: %w", err)
	}

	return &pgClient{
		database: &pgDb{dbc: dbc},
	}, nil
}

func (c *pgClient) DB() persistence.DB {
	return c.database
}

func (c *pgClient) Close() error {
	if c.database != nil {
		c.database.Close()
	}

	return nil
}
