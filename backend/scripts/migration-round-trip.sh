#!/usr/bin/env sh
set -eu

: "${POSTGRES_ADMIN_DATABASE_URL:?POSTGRES_ADMIN_DATABASE_URL is required}"
: "${MIGRATION_ROUNDTRIP_DATABASE_URL:?MIGRATION_ROUNDTRIP_DATABASE_URL is required}"
: "${MIGRATION_ROUNDTRIP_ADMIN_DATABASE_URL:?MIGRATION_ROUNDTRIP_ADMIN_DATABASE_URL is required}"

database_name=miniclass_migration_roundtrip
cleanup() {
    psql "$POSTGRES_ADMIN_DATABASE_URL" -v ON_ERROR_STOP=1 \
        -c "drop database if exists $database_name with (force)" >/dev/null
}
trap cleanup EXIT

psql "$POSTGRES_ADMIN_DATABASE_URL" -v ON_ERROR_STOP=1 \
    -c "drop database if exists $database_name with (force)" \
    -c "create database $database_name owner miniclass_migrator"

echo "Applying migrations"
DATABASE_URL="$MIGRATION_ROUNDTRIP_DATABASE_URL" go run ./cmd/migrate up
echo "Rolling migrations back"
DATABASE_URL="$MIGRATION_ROUNDTRIP_ADMIN_DATABASE_URL" go run ./cmd/migrate down
echo "Reapplying migrations"
DATABASE_URL="$MIGRATION_ROUNDTRIP_DATABASE_URL" go run ./cmd/migrate up
