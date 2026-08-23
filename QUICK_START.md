# Quick Start Guide

Get the MiniClass development environment running in under 5 minutes.

---

## Prerequisites Checklist

- [ ] Docker & Docker Compose installed
- [ ] Go 1.27+ installed (`go version`)
- [ ] Node.js 20+ installed (`node --version`)
- [ ] Make installed (`make --version`)
- [ ] Git installed

---

## First Time Setup

### 1. Environment Configuration
```bash
# Copy environment template
cp .env.example .env

# .env is already configured with sensible defaults
# No changes needed for local development
```

### 2. Start Database
```bash
# Start PostgreSQL in Docker
docker-compose up -d postgres

# Verify it's running
docker-compose ps

# Expected: miniclass-postgres with status "Up"
```

### 3. Backend Setup
```bash
cd backend

# Install Go dependencies
go mod download

# Install development tools (air, sqlc, goose)
make install-tools

# Create initial migration (you'll implement this in Phase 1)
# make migrate-create NAME=initial_schema

# For now, migrations directory is empty - this is expected
# You'll create the first migration during implementation

cd ..
```

### 4. Frontend Setup
```bash
cd frontend

# Install npm dependencies
npm install

cd ..
```

---

## Running the Stack

Open **three terminal windows**:

### Terminal 1: Database
```bash
# If not already running
docker-compose up postgres

# Leave this running
# Ctrl+C to stop
```

### Terminal 2: Backend
```bash
cd backend

# Run migrations (once you've created them)
# make migrate-up

# Start API server with hot reload
make dev

# Server starts on http://localhost:8080
# Ctrl+C to stop
```

### Terminal 3: Frontend
```bash
cd frontend

# Start Vite dev server
npm run dev

# Dev server starts on http://localhost:5173
# Ctrl+C to stop
```

---

## Verify It's Working

### Check Health Endpoint
```bash
# Once backend is running and health check is implemented:
curl http://localhost:8080/api/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2026-08-22T10:30:00Z",
  "database": "connected",
  "version": "0.1.0"
}
```

### Check Frontend
Open browser to: http://localhost:5173

You should see the health check component displaying backend status.

### Check Database GUI (Optional)
Open browser to: http://localhost:8081

- **System:** PostgreSQL
- **Server:** postgres
- **Username:** miniclass
- **Password:** miniclass_dev_password
- **Database:** miniclass

---

## Development Workflow

### Making Backend Changes

1. Edit files in `backend/internal/` or `backend/cmd/`
2. Air automatically rebuilds and restarts the server
3. Check terminal for build errors
4. Test your changes

### Making Frontend Changes

1. Edit files in `frontend/src/`
2. Vite Hot Module Replacement (HMR) updates instantly
3. Check browser console for errors
4. Changes appear immediately

### Database Changes

```bash
cd backend

# Create new migration
make migrate-create NAME=add_users_table

# Edit the generated file in migrations/
# Add your SQL in the up section

# Apply migration
make migrate-up

# If you need to undo
make migrate-down
```

### Updating SQL Queries

```bash
cd backend

# Edit .sql files in sql/queries/

# Regenerate Go code
make sqlc

# New type-safe functions available in internal/db/
```

---

## Running Tests

### Backend Tests
```bash
cd backend
make test

# With coverage
make test-coverage
```

### Frontend Tests
```bash
cd frontend
npm run test
```

---

## Troubleshooting

### Port Already in Use

**Problem:** `Error: Port 8080 already in use`

**Solution:** Edit `.env` and change ports:
```env
PORT=8081
VITE_PORT=5174
POSTGRES_PORT=5433
VITE_API_URL=http://localhost:8081/api
```

Then restart services.

---

### Database Connection Failed

**Problem:** Backend can't connect to database

**Check:**
```bash
# Is Postgres running?
docker-compose ps

# Check logs
docker-compose logs postgres

# Verify connection manually
docker exec -it miniclass-postgres psql -U miniclass -d miniclass

# Should open psql prompt
# Type \q to quit
```

**Solution:**
```bash
# Restart Postgres
docker-compose restart postgres

# Or full reset
docker-compose down
docker-compose up -d postgres
```

---

### Migration Failed

**Problem:** `make migrate-up` fails

**Solution:**
```bash
# Check current migration status
cd backend
goose -dir migrations postgres "$DATABASE_URL" status

# Reset database (CAUTION: destroys all data)
make reset-db
```

---

### Frontend Can't Reach Backend

**Problem:** API calls fail with network errors

**Check:**
1. Is backend running? Check Terminal 2
2. Is backend on correct port? Check `.env` PORT value
3. Does `VITE_API_URL` in `.env` match backend URL?

**Solution:**
```bash
# Verify backend is running
curl http://localhost:8080/api/health

# Update VITE_API_URL if needed
# Restart frontend dev server
```

---

### Hot Reload Not Working

**Backend (Air):**
```bash
# Check .air.toml exists
ls backend/.air.toml

# Check air is installed
air -v

# Reinstall if needed
go install github.com/air-verse/air@latest
```

**Frontend (Vite):**
```bash
# Restart dev server
# Ctrl+C, then npm run dev
```

---

## Resetting Everything

```bash
# Stop all services
docker-compose down

# Remove database volume (destroys all data)
docker-compose down -v

# Remove node_modules (optional)
rm -rf frontend/node_modules

# Remove backend binaries (optional)
rm -rf backend/tmp backend/bin

# Start fresh
docker-compose up -d postgres
cd backend && make migrate-up && make seed
cd ../frontend && npm install
```

---

## Current Status

🟡 **Scaffolding Complete** - Structure is in place, implementation needed

### What's Done
- ✅ Folder structure created
- ✅ Configuration files in place
- ✅ Docker Compose ready
- ✅ Makefiles and scripts ready
- ✅ Package configs (Go, Node) ready

### What's Next
See `IMPLEMENTATION_PLAN.md` for detailed steps:
1. Implement backend config loading
2. Implement database connection
3. Create first migration
4. Implement health check endpoint
5. Implement frontend health check component
6. Write integration tests
7. Verify end-to-end

---

## Getting Help

- **Architecture:** See `achitecture.md`
- **Structure:** See `STRUCTURE.md`
- **Implementation:** See `IMPLEMENTATION_PLAN.md`
- **Commands:** Run `make help` in backend/

---

## Quick Reference

```bash
# Database
docker-compose up -d postgres    # Start
docker-compose logs postgres     # View logs
docker-compose down              # Stop

# Backend
cd backend
make dev                         # Run with hot reload
make test                        # Run tests
make migrate-up                  # Apply migrations
make reset-db                    # Nuclear reset

# Frontend  
cd frontend
npm run dev                      # Start dev server
npm run build                    # Production build
npm run test                     # Run tests

# URLs
http://localhost:5173            # Frontend
http://localhost:8080            # Backend API
http://localhost:8080/api/health # Health check
http://localhost:8081            # Adminer (DB GUI)
```

---

**Ready to start?** Head to `IMPLEMENTATION_PLAN.md` and begin Phase 1! 🚀
