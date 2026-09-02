// Package registry describes the tenant entities exercised by the Layer 2
// isolation harness. Each entity owns its registration file so future entity
// work does not serialize on one central test fixture.
package registry

import (
	"context"
	"fmt"
	"sync"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	testharness "github.com/chrismott/miniclass/internal/testing"
)

// Entity supplies the smallest set of operations needed to generate Layer 2
// tenant-isolation cases for an entity.
type Entity struct {
	TableName               string
	YearScoped              bool
	Immutable               bool
	Factory                 func(context.Context, *testharness.Harness, ids.XID) (ids.XID, error)
	ReadIDs                 func(context.Context, *data.Tx) ([]ids.XID, error)
	FetchByID               func(context.Context, *data.Tx, ids.XID) (bool, error)
	UpdateByID              func(context.Context, *data.Tx, ids.XID) (bool, error)
	DeleteByID              func(context.Context, *data.Tx, ids.XID) (bool, error)
	InsertWithForeignParent func(context.Context, *testharness.Harness, ids.XID, ids.XID) error
}

var (
	mu      sync.RWMutex
	entries []Entity
)

// Register adds one entity to the process-wide registry. It is intended for
// init functions in per-entity files and rejects incomplete/duplicate entries
// immediately so the harness cannot silently lose coverage.
func Register(entity Entity) {
	if entity.TableName == "" || entity.Factory == nil || entity.ReadIDs == nil || entity.FetchByID == nil || entity.UpdateByID == nil || entity.DeleteByID == nil || entity.InsertWithForeignParent == nil {
		panic(fmt.Sprintf("register isolation entity %q: all operations are required", entity.TableName))
	}
	mu.Lock()
	defer mu.Unlock()
	for _, existing := range entries {
		if existing.TableName == entity.TableName {
			panic(fmt.Sprintf("register isolation entity %q: duplicate table", entity.TableName))
		}
	}
	entries = append(entries, entity)
}

// Entries returns the registered entities in registration order.
func Entries() []Entity {
	mu.RLock()
	defer mu.RUnlock()
	return append([]Entity(nil), entries...)
}

// ForTable returns an entity by table name.
func ForTable(tableName string) (Entity, bool) {
	mu.RLock()
	defer mu.RUnlock()
	for _, entity := range entries {
		if entity.TableName == tableName {
			return entity, true
		}
	}
	return Entity{}, false
}
