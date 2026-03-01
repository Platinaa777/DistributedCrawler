// migrate runs database migrations using goose.
//
// Usage:
//
//	migrate [flags] <command>
//
// Commands: up, up-by-one, down, reset, status, version, create <name>
//
// Flags:
//
//	--dsn       Postgres DSN (overrides PG_DSN env var)
//	--dir       Migrations directory (default: internal/infra/persistence/postgres/migrations)
//
// Examples:
//
//	migrate up
//	migrate --dsn "postgres://user:pass@localhost:5432/db?sslmode=disable" up
//	migrate down
//	migrate status
//	migrate create add_something
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/pressly/goose/v3"
)

const defaultMigrationsDir = "internal/infra/persistence/postgres/migrations"

func main() {
	dsn := flag.String("dsn", "", "Postgres DSN (falls back to PG_DSN env var)")
	dir := flag.String("dir", defaultMigrationsDir, "Migrations directory")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "\nAvailable commands: up, up-by-one, down, reset, status, version, create <name>")
		os.Exit(1)
	}

	if *dsn == "" {
		*dsn = os.Getenv("PG_DSN")
	}
	if *dsn == "" {
		log.Fatal("DSN is required: use --dsn flag or PG_DSN env var")
	}

	db, err := sql.Open("pgx", *dsn)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}

	goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("failed to set dialect: %v", err)
	}

	command := args[0]
	commandArgs := args[1:]

	if err := goose.RunContext(context.Background(), command, db, *dir, commandArgs...); err != nil {
		log.Fatalf("goose %s: %v", command, err)
	}
}
