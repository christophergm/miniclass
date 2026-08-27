// Command migrate applies or rolls back Goose migrations through the
// schema-owning miniclass_migrator role from DATABASE_URL.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const migrationDir = "migrations"

func main() {
	if err := run(context.Background(), os.Args[1:], os.Getenv("DATABASE_URL")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, databaseURL string) error {
	command := "up"
	if len(args) > 1 {
		return errors.New("migration command accepts at most one action: up, down, or status")
	}
	if len(args) == 1 {
		command = strings.ToLower(strings.TrimSpace(args[0]))
	}
	if command != "up" && command != "down" && command != "status" {
		return fmt.Errorf("unsupported migration action %q; use up, down, or status", command)
	}
	if strings.TrimSpace(databaseURL) == "" {
		return errors.New("migration failed: DATABASE_URL is required")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("migration failed: open database: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("migration failed: database connection: %w", err)
	}
	if err := goose.RunWithOptionsContext(ctx, command, db, migrationDir, nil, goose.WithAllowMissing()); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Printf("Migration %s completed successfully.\n", command)
	return nil
}
