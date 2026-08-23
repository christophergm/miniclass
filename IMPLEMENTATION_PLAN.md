# Implementation Plan: Local Development Environment

This document outlines the step-by-step plan to achieve the first milestone: a fully functional local development environment with health check and tests.

## Milestone Goal

✅ All services running locally  
✅ Health check endpoint working end-to-end  
✅ At least one backend integration test passing  

---

## Phase 1: Backend Foundation

### 1.1 Configuration Management
**File:** `backend/internal/config/config.go`

- [ ] Load environment variables from `.env`
- [ ] Define `Config` struct with all settings
- [ ] Provide sensible defaults
- [ ] Validate required configuration

**Acceptance:** Config can be loaded and accessed by main.go

---

### 1.2 Database Connection
**File:** `backend/internal/db/db.go`

- [ ] Create connection pool using pgx
- [ ] Implement health check function (`PingDB()`)
- [ ] Handle connection errors gracefully
- [ ] Support connection string from config

**Acceptance:** Can connect to Postgres and ping successfully

---

### 1.3 First Migration
**File:** `backend/migrations/00001_initial_schema.sql`

- [ ] Create basic schema for health check testing
- [ ] Add a simple `health_checks` table (optional, for testing)
- [ ] Ensure migration is reversible (down migration)

**Commands:**
```bash
make migrate-create NAME=initial_schema
make migrate-up
```

**Acceptance:** Migration runs successfully against local database

---

### 1.4 HTTP Server & Routing
**Files:** 
- `backend/internal/api/server.go`
- `backend/internal/api/routes.go`
- `backend/internal/api/middleware.go`

- [ ] Setup Chi router
- [ ] Add CORS middleware
- [ ] Add request logging middleware
- [ ] Add JSON content-type middleware
- [ ] Define route structure

**Acceptance:** Server starts and responds to requests

---

### 1.5 Health Check Handler
**Files:**
- `backend/internal/api/handlers/health.go`
- `backend/internal/api/handlers/health_test.go` (unit test)

- [ ] Implement `GET /api/health` endpoint
- [ ] Check database connectivity
- [ ] Return JSON with status, timestamp, database status, version
- [ ] Handle database connection failures gracefully

**Response Format:**
```json
{
  "status": "healthy",
  "timestamp": "2026-08-22T10:30:00Z",
  "database": "connected",
  "version": "0.1.0"
}
```

**Acceptance:** Health check endpoint returns 200 with correct JSON

---

### 1.6 Main API Entry Point
**File:** `backend/cmd/api/main.go`

- [ ] Initialize configuration
- [ ] Setup database connection
- [ ] Create HTTP server
- [ ] Handle graceful shutdown
- [ ] Log startup information

**Acceptance:** `go run ./cmd/api` starts server successfully

---

### 1.7 Migration & Seed Commands
**Files:**
- `backend/cmd/migrate/main.go`
- `backend/cmd/seed/main.go`
- `backend/scripts/seed.sql`

**Migrate:**
- [ ] Wrapper around Goose CLI
- [ ] Reads DATABASE_URL from env

**Seed:**
- [ ] Load `scripts/seed.sql`
- [ ] Execute against database
- [ ] Report success/failure

**Acceptance:** `make migrate-up` and `make seed` work correctly

---

### 1.8 Integration Test
**File:** `backend/tests/integration/health_test.go`

- [ ] Setup test database connection
- [ ] Run migrations before tests
- [ ] Test health check endpoint
- [ ] Test database connectivity
- [ ] Clean up after tests

**Test Flow:**
1. Connect to test database
2. Run migrations
3. Start HTTP server (test mode)
4. Call `GET /api/health`
5. Assert response is correct
6. Cleanup

**Acceptance:** `make test` passes with green output

---

## Phase 2: Frontend Foundation

### 2.1 Basic App Structure
**Files:**
- `frontend/src/main.tsx`
- `frontend/src/App.tsx`

- [ ] Setup React with TypeScript
- [ ] Configure router (react-router-dom)
- [ ] Create basic layout

**Acceptance:** `bun run dev` starts dev server

---

### 2.2 API Client
**File:** `frontend/src/lib/api.ts`

- [ ] Create axios/fetch wrapper
- [ ] Read API URL from env (`VITE_API_URL`)
- [ ] Export typed API methods
- [ ] Handle errors consistently

**Acceptance:** Can make requests to backend

---

### 2.3 Health Check Page
**Files:**
- `frontend/src/features/health/HealthCheck.tsx`
- `frontend/src/lib/hooks/useHealth.ts`

- [ ] Create simple health check component
- [ ] Use TanStack Query for data fetching
- [ ] Display connection status
- [ ] Show backend version, timestamp, database status
- [ ] Add auto-refresh every 30s

**Acceptance:** Health check displays and updates from backend

---

### 2.4 Home Page
**File:** `frontend/src/App.tsx`

- [ ] Create home/landing page
- [ ] Include health check component
- [ ] Basic styling

**Acceptance:** Frontend loads and shows health status

---

## Phase 3: Integration & Verification

### 3.1 End-to-End Smoke Test

**Manual steps:**
1. [ ] Copy `.env.example` to `.env`
2. [ ] Start Postgres: `docker compose up -d postgres`
3. [ ] Run migrations: `cd backend && make migrate-up`
4. [ ] Start backend: `make dev`
5. [ ] Start frontend: `cd ../frontend && bun run dev`
6. [ ] Open browser to http://localhost:5173
7. [ ] Verify health check shows "healthy"
8. [ ] Check backend logs show request
9. [ ] Check Adminer at http://localhost:8081

**Acceptance:** Full stack runs successfully

---

### 3.2 Test Suite
- [ ] Backend integration tests pass: `make test`
- [ ] Frontend builds without errors: `bun run build`
- [ ] No linting errors: `bun run lint`

**Acceptance:** All tests and builds pass

---

### 3.3 Documentation Updates
- [ ] Update README.md with any learnings
- [ ] Add troubleshooting section if needed
- [ ] Document any gotchas

---

## Phase 4: Developer Experience Improvements

### 4.1 Database Reset Script
**File:** `backend/scripts/reset.sh` (optional helper)

- [ ] Already in Makefile as `reset-db`
- [ ] Test reset functionality
- [ ] Verify seed data loads correctly

---

### 4.2 Sample Seed Data
**File:** `backend/scripts/seed.sql`

- [ ] Add sample teacher
- [ ] Add sample class
- [ ] Add sample students
- [ ] Add sample assignments

**Purpose:** Make it easy to develop features with realistic data

---

## Success Criteria

### Must Have (Blocking)
- [x] Folder structure created
- [ ] Docker Compose starts Postgres
- [ ] Backend connects to database
- [ ] Health check endpoint responds correctly
- [ ] Frontend displays health check data
- [ ] At least one integration test passes
- [ ] All services run in parallel without errors

### Nice to Have (Non-blocking)
- [ ] Seed data with sample records
- [ ] Reset database helper working
- [ ] Hot reload working for both frontend and backend
- [ ] Adminer accessible for database inspection

---

## Execution Order

The phases should be executed in order, but within each phase, tasks can be parallelized:

1. **Start with Phase 1** (Backend Foundation) - Complete all tasks
2. **Then Phase 2** (Frontend Foundation) - Can be done in parallel with Phase 1 after 1.5 is complete
3. **Then Phase 3** (Integration) - Validates everything works together
4. **Finally Phase 4** (DX improvements) - Polish

---

## Next Steps After Milestone

Once the first milestone is complete:

1. **Data Model Design** - Define full schema (users, classes, students, assignments)
2. **CRUD Operations** - Implement create/read/update/delete for each entity
3. **Authentication** - Integrate Supabase Auth
4. **Assignment Algorithm** - Implement sorting/matching logic
5. **Drag & Drop UI** - Build interactive assignment interface

---

## Notes

- This plan focuses on infrastructure first, features second
- Each phase builds on previous phases
- Tests are written alongside implementation, not after
- Documentation is updated continuously
- Use `.env` for all configuration (no hardcoded values)

---

## Tracking

Mark items complete as you go. Update this document if scope changes or new tasks are discovered.

**Current Status:** Scaffolding complete, ready for implementation

**Started:** 2026-08-22  
**Target Completion:** TBD  
**Actual Completion:** TBD
