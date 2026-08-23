# Docker Compose Modernization

## Summary

Updated project from legacy `docker-compose` to modern `docker compose` syntax and conventions.

## Changes Made

### 1. Command Syntax
**Before:** `docker-compose up -d`  
**After:** `docker compose up -d`

### 2. Compose File
**Before:** `docker-compose.yml`  
**After:** `compose.yaml` (recommended naming convention)

### 3. Compose File Format
**Before:**
```yaml
version: '3.9'

services:
  postgres:
    ...
```

**After:**
```yaml
# Docker Compose configuration for local development
# https://docs.docker.com/compose/

services:
  postgres:
    ...
```

The `version` field is now obsolete and removed.

### 4. Postgres 18 Volume Mount
**Before:**
```yaml
volumes:
  - postgres_data:/var/lib/postgresql/data
```

**After:**
```yaml
volumes:
  - postgres_data:/var/lib/postgresql
```

Postgres 18+ uses a new data directory structure. The recommended mount point is now `/var/lib/postgresql` which allows for easier upgrades using `pg_upgrade --link`.

## Requirements

- **Docker Compose v2+** (ships with Docker Desktop)
- Verify: `docker compose version`
- Should show: v2.0.0 or higher

## Migration for Existing Developers

If you're upgrading from the old setup:

1. **Stop old containers:**
   ```bash
   docker-compose down  # Old command still works
   ```

2. **Pull latest code:**
   ```bash
   git pull
   ```

3. **Restart with new commands:**
   ```bash
   docker compose up -d postgres
   ```

4. **If you have data to preserve:**
   The volume name hasn't changed, so your data is safe. However, if you encounter issues, you may need to recreate:
   ```bash
   docker compose down -v  # WARNING: Destroys data
   docker compose up -d postgres
   cd backend && make migrate-up && make seed
   ```

## Documentation Updates

All documentation has been updated:
- ✅ README.md
- ✅ QUICK_START.md
- ✅ STRUCTURE.md
- ✅ IMPLEMENTATION_PLAN.md
- ✅ TOOLING_AUDIT.md

## Common Commands (New Syntax)

```bash
# Start services
docker compose up -d postgres
docker compose up -d  # All services

# View logs
docker compose logs postgres
docker compose logs -f postgres  # Follow logs

# Stop services
docker compose down

# Stop and remove volumes (destroys data)
docker compose down -v

# View running containers
docker compose ps

# Restart a service
docker compose restart postgres

# Execute commands in container
docker exec -it miniclass-postgres psql -U miniclass

# View service health
docker compose ps  # Shows health status
```

## Troubleshooting

### "command not found: docker-compose"

✅ **This is expected!** Use `docker compose` (with space) instead.

### "version is obsolete"

If you see warnings about the version field, pull the latest `compose.yaml` which has it removed.

### Port 5432 already in use

Another Postgres container is running. Find and stop it:
```bash
docker ps | grep postgres
docker stop <container-name>
```

Or use a different port in `.env`:
```env
POSTGRES_PORT=5433
```

### Data directory errors with Postgres 18

If you see warnings about `/var/lib/postgresql/data`, you're using old volume mounts. Pull the latest `compose.yaml` which uses `/var/lib/postgresql`.

## Why This Change?

1. **Docker Compose v2** is the current standard (since 2021)
2. **Better performance** - Native Go implementation vs Python
3. **Modern syntax** - `docker compose` matches other `docker` commands
4. **Active development** - v1 (`docker-compose`) is deprecated
5. **Postgres 18 compatibility** - New data directory structure

## References

- [Docker Compose v2 announcement](https://docs.docker.com/compose/cli-command/)
- [Compose file reference](https://docs.docker.com/compose/compose-file/)
- [Postgres Docker image changes](https://github.com/docker-library/postgres/issues/37)

---

**Updated:** August 23, 2026  
**Status:** ✅ Complete
