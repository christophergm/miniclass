package data

import (
	"context"
	"strings"
	"testing"

	"github.com/chrismott/miniclass/internal/config"
)

func TestNewRequiresConfiguration(t *testing.T) {
	_, err := New(context.Background(), nil)
	if err == nil || err.Error() != "create database: configuration is nil" {
		t.Fatalf("New() error = %v", err)
	}
}

func TestNewRejectsInvalidConfiguration(t *testing.T) {
	cfg := &config.Config{Port: "8080"}

	_, err := New(context.Background(), cfg)
	if err == nil || err.Error() != "create database: configuration error: DATABASE_URL is required" {
		t.Fatalf("New() error = %v", err)
	}
}

func TestNewFromURLRejectsMalformedURL(t *testing.T) {
	_, err := NewFromURL(context.Background(), "not-a-postgres-url")
	if err == nil || !strings.Contains(err.Error(), "parse database URL:") {
		t.Fatalf("NewFromURL() error = %v", err)
	}
}

func TestNewFromURLReturnsPingFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewFromURL(ctx, "postgres://miniclass:miniclass@127.0.0.1:5432/miniclass")
	if err == nil || !strings.Contains(err.Error(), "create database: ping database:") {
		t.Fatalf("NewFromURL() error = %v", err)
	}
}

func TestPingDBRejectsNilDatabase(t *testing.T) {
	var database *DB

	err := database.PingDB(context.Background())
	if err == nil || err.Error() != "ping database: connection pool is nil" {
		t.Fatalf("PingDB() error = %v", err)
	}
}

func TestPingDBRejectsNilPool(t *testing.T) {
	err := PingDB(context.Background(), nil)
	if err == nil || err.Error() != "ping database: connection pool is nil" {
		t.Fatalf("PingDB() error = %v", err)
	}
}

func TestCloseIsSafeForNilDatabase(t *testing.T) {
	var database *DB

	database.Close()
}
