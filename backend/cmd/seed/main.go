// Command seed loads the development seed SQL into PostgreSQL.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const seedFile = "scripts/seed.sql"

func main() {
	if err := run(context.Background(), os.Getenv("DATABASE_URL")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, databaseURL string) error {
	if strings.TrimSpace(databaseURL) == "" {
		return errors.New("seed failed: DATABASE_URL is required")
	}

	sqlPath := filepath.Join(sourceDir(), "..", "..", seedFile)
	script, err := os.ReadFile(sqlPath)
	if err != nil {
		return fmt.Errorf("seed failed: read %s: %w", seedFile, err)
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("seed failed: open database: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("seed failed: database connection: %w", err)
	}
	if _, err := db.ExecContext(ctx, string(script)); err != nil {
		return fmt.Errorf("seed failed: execute %s: %w", seedFile, err)
	}

	fmt.Println("Seed data loaded successfully.")
	return nil
}

func sourceDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}
