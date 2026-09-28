package data

import (
	"context"
	"fmt"
	"strings"

	"github.com/chrismott/miniclass/internal/ids"
)

// GuardianLoginContext is the minimal cross-tenant result permitted before a
// verified guardian selects a context. The database function exposes no member
// or student information and filters to current guardian relationships.
type GuardianLoginContext struct {
	OrganizationID   ids.XID
	SchoolYearID     ids.XID
	AdultID          ids.XID
	OrganizationName string
	SchoolYearLabel  string
}

// FindGuardianLoginContexts invokes the sole narrowly scoped cross-tenant
// lookup used by email-only guardian login. All other domain reads require an
// InTenantRead transaction.
func (d *DB) FindGuardianLoginContexts(ctx context.Context, email string) ([]GuardianLoginContext, error) {
	if d == nil || d.pool == nil {
		return nil, fmt.Errorf("find guardian login contexts: database is nil")
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, fmt.Errorf("find guardian login contexts: email is empty")
	}
	rows, err := d.pool.Query(ctx, "select * from public.find_guardian_login_contexts($1, current_schema())", email)
	if err != nil {
		return nil, fmt.Errorf("find guardian login contexts: %w", err)
	}
	defer rows.Close()
	contexts := make([]GuardianLoginContext, 0)
	for rows.Next() {
		var item GuardianLoginContext
		if err := rows.Scan(&item.OrganizationID, &item.SchoolYearID, &item.AdultID, &item.OrganizationName, &item.SchoolYearLabel); err != nil {
			return nil, fmt.Errorf("find guardian login contexts: %w", err)
		}
		contexts = append(contexts, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("find guardian login contexts: %w", err)
	}
	return contexts, nil
}
