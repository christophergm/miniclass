# MiniClass

A class planning and assignment management application for students and teachers.

## Architecture

See [architecture.md](./achitecture.md) for the full architecture overview.

**Stack:**
- Frontend: React + TypeScript + Vite
- Backend: Go + Chi + pgx + sqlc
- Database: PostgreSQL (Supabase in production, Docker locally)
- Auth: Supabase Auth

## Project Structure

```
miniclass/
├── backend/              # Go API server
│   ├── cmd/             # Entry points (api, migrate, seed)
│   ├── internal/        # Application code
│   ├── migrations/      # Database migrations (Goose)
│   ├── sql/             # SQL queries (sqlc)
│   ├── scripts/         # Utility scripts
│   └── tests/           # Integration tests
├── frontend/            # React application
│   ├── src/
│   │   ├── components/  # Shared UI components
│   │   ├── features/    # Feature modules
│   │   ├── lib/         # Utilities and API client
│   │   └── hooks/       # Custom React hooks
│   └── public/          # Static assets
└── docker-compose.yml   # Local development services
```

## Getting Started

### Prerequisites

- Docker & Docker Compose
- Go 1.27+
- Node.js 20+
- Make

### First Time Setup

1. **Clone and setup environment:**
   ```bash
   git clone <repo-url>
   cd miniclass
   cp .env.example .env
   ```

2. **Start database:**
   ```bash
   docker-compose up -d postgres
   ```

3. **Install backend tools:**
   ```bash
   cd backend
   make install-tools
   ```

4. **Run migrations:**
   ```bash
   make migrate-up
   ```

5. **Seed database (optional):**
   ```bash
   make seed
   ```

6. **Install frontend dependencies:**
   ```bash
   cd ../frontend
   npm install
   ```

### Running Locally

**Terminal 1 - Database:**
```bash
docker-compose up postgres
```

**Terminal 2 - Backend:**
```bash
cd backend
make dev
```

**Terminal 3 - Frontend:**
```bash
cd frontend
npm run dev
```

**Access:**
- Frontend: http://localhost:5173
- Backend API: http://localhost:8080
- Adminer (DB GUI): http://localhost:8081

### Frontend shell

The frontend starts at the responsive MiniClass workspace overview. Its initial
navigation is available at:

- `/` — classroom overview
- `/classes` — classes workspace placeholder
- `/assignments` — assignments workspace placeholder
- `/students` — students workspace placeholder
- `/settings` — settings workspace placeholder

Run `npm run dev` from `frontend/` for Vite development with hot module
replacement, or `npm run build` to verify the production bundle.

### Development Commands

#### Backend integration test

The PostgreSQL health integration test requires an isolated test database. Start
the local PostgreSQL service, then set `TEST_DATABASE_URL` to the
`miniclass_test` database created by Docker Compose:

```bash
docker compose up -d postgres
export TEST_DATABASE_URL='postgres://miniclass:miniclass_dev_password@localhost:5432/miniclass_test?sslmode=disable'
cd backend
make test
```

Each test run creates a unique schema, applies the migrations, and drops that
schema during cleanup. Do not point `TEST_DATABASE_URL` at a development or
production database.

**Backend:**
```bash
cd backend
make help              # Show all available commands
make dev               # Run with hot reload
make test              # Run integration tests
make migrate-create NAME=my_migration  # Create new migration
make reset-db          # Reset database to clean state
make sqlc              # Regenerate DB code
```

**Frontend:**
```bash
cd frontend
npm run dev            # Start dev server
npm run build          # Production build
npm run test           # Run tests
npm run lint           # Lint code
```

## Parallel Development (Multiple Worktrees)

To run multiple instances in parallel:

1. Create a new worktree
2. Copy `.env.example` to `.env` in the worktree
3. Edit `.env` to use different ports:
   ```bash
   PORT=8081
   VITE_PORT=5174
   POSTGRES_PORT=5433
   ```
4. Start services with the custom `.env`

## Testing

The reproducible quality gates are:

```bash
cd backend && make test
cd frontend && npm ci && npm run test -- --run
cd frontend && npm ci && npm run build
cd frontend && npm ci && npm run lint
git diff --check
```

CI runs the backend integration test against PostgreSQL and publishes separate
checks for backend tests, frontend tests, frontend build, frontend lint, and
repository formatting.

**Backend Integration Tests:**
```bash
cd backend
make test
```

Tests use the `miniclass_test` database automatically created by Docker.

**Frontend Tests:**
```bash
cd frontend
npm run test
```

## Database Management

**Create migration:**
```bash
cd backend
make migrate-create NAME=add_users_table
```

**Apply migrations:**
```bash
make migrate-up
```

**Rollback last migration:**
```bash
make migrate-down
```

**Reset database:**
```bash
make reset-db  # Drops schema, re-runs migrations, seeds data
```

## Health Check

Verify the stack is running:
```bash
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

## Production Deployment

- Backend: Render
- Frontend: Render Static Site
- Database: Supabase Postgres
- Auth: Supabase Auth

See deployment documentation (TBD) for details.

## Contributing

1. Create feature branch
2. Make changes
3. Run tests: `make test` (backend) and `npm test` (frontend)
4. Ensure health check passes
5. Submit PR

## License

TBD
