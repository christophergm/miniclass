package integration

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/identity"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestIdentityBootstrapAndInvitationRegeneration(t *testing.T) {
	store, ctx := openIdentityStore(t)
	now := time.Now().UTC().Truncate(time.Second)

	first, err := identity.Bootstrap(ctx, store, identity.BootstrapInput{
		OrganizationName: "Synthetic Academy",
		HomeroomLabel:    "advisory",
		OwnerEmail:       "Owner@Example.test",
		ClaimBaseURL:     "https://planner.example/claim?next=%2Fwelcome",
		InvitationTTL:    2 * time.Hour,
		Now:              now,
	})
	require.NoError(t, err)
	require.Equal(t, "owner@example.test", *first.Member.InvitedEmail)
	require.Nil(t, first.Member.UserID)
	require.Equal(t, "owner", first.Member.Role)
	require.Equal(t, 1, first.Token.Generation)
	require.NotEmpty(t, first.TokenValue)

	claimURL, err := url.Parse(first.ClaimURL)
	require.NoError(t, err)
	require.Equal(t, first.TokenValue, claimURL.Query().Get("token"))
	require.Equal(t, "/welcome", claimURL.Query().Get("next"))

	oldToken, err := store.GetAccessTokenByBearer(ctx, first.TokenValue)
	require.NoError(t, err)
	member, err := store.GetInvitationMember(ctx, first.Token.ID)
	require.NoError(t, err)
	require.Equal(t, first.Token.ID, oldToken.ID)
	require.Equal(t, identity.PurposeAdminInvitation, oldToken.Purpose)
	require.Equal(t, first.Member.ID, member.ID)

	replacement, err := identity.RegenerateAdminInvitation(ctx, store, string(first.Token.ID), "https://planner.example/claim", time.Hour, now.Add(time.Minute))
	require.NoError(t, err)
	require.NotEqual(t, first.Token.ID, replacement.Token.ID)
	require.Equal(t, 2, replacement.Token.Generation)
	require.NotEqual(t, first.TokenValue, replacement.TokenValue)

	old, err := store.GetAccessTokenByBearer(ctx, first.TokenValue)
	require.NoError(t, err)
	require.NotNil(t, old.RevokedAt, "old invitation was not revoked")
	consumed, err := store.ConsumeAccessToken(ctx, old.ID)
	require.NoError(t, err)
	require.False(t, consumed, "revoked invitation was consumed")

	fresh, err := store.GetAccessTokenByBearer(ctx, replacement.TokenValue)
	require.NoError(t, err)
	consumed, err = store.ConsumeAccessToken(ctx, fresh.ID)
	require.NoError(t, err)
	require.True(t, consumed, "replacement invitation was not consumed")

	replacementURL, err := url.Parse(replacement.ClaimURL)
	require.NoError(t, err)
	require.Equal(t, replacement.TokenValue, replacementURL.Query().Get("token"))
	member, err = store.GetInvitationMember(ctx, replacement.Token.ID)
	require.NoError(t, err)
	require.Equal(t, first.Member.ID, member.ID)
}

func openIdentityStore(t *testing.T) (*identity.Store, context.Context) {
	t.Helper()
	testDatabaseURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if testDatabaseURL == "" {
		t.Skip("TEST_DATABASE_URL is required for the PostgreSQL integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	adminPool, err := pgxpool.New(ctx, testDatabaseURL)
	require.NoError(t, err)
	if err != nil {
		return nil, ctx
	}
	require.NoError(t, adminPool.Ping(ctx))
	t.Cleanup(adminPool.Close)

	schemaName := fmt.Sprintf("miniclass_identity_%d", time.Now().UnixNano())
	_, err = adminPool.Exec(ctx, "create schema "+schemaName)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := adminPool.Exec(context.Background(), "drop schema if exists "+schemaName+" cascade")
		require.NoError(t, cleanupErr)
	})

	schemaURL, err := withSearchPath(testDatabaseURL, schemaName)
	require.NoError(t, err)
	gooseDB, err := goose.OpenDBWithDriver("postgres", schemaURL)
	require.NoError(t, err)
	if err != nil {
		return nil, ctx
	}
	t.Cleanup(func() { require.NoError(t, gooseDB.Close()) })
	require.NoError(t, goose.Up(gooseDB, migrationsPath(t), goose.WithAllowMissing()))

	database, err := data.NewFromURL(ctx, schemaURL)
	require.NoError(t, err)
	if err != nil {
		return nil, ctx
	}
	t.Cleanup(database.Close)
	return identity.NewStore(database), ctx
}

func migrationsPath(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations")
}
