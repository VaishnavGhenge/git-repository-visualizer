# Git Repository Visualizer

A Go application for indexing and analyzing Git repositories. It provides REST APIs to manage repositories and background workers to process repository data.

## Project Structure

```
├── cmd/
│   ├── api/main.go       # HTTP API server entry point
│   └── worker/main.go    # Background worker entry point
├── internal/
│   ├── config/           # Configuration loading
│   ├── database/         # PostgreSQL connection and repository operations
│   ├── git/              # Git cloning service (GitHub, BitBucket)
│   ├── http/             # HTTP handlers and middleware (chi router)
│   ├── queue/            # Redis queue publisher/consumer
│   ├── redis/            # Redis client configuration
│   ├── types/            # Shared types
│   ├── validation/       # Request validation
│   └── worker/           # Background job handler
├── migrations/           # SQL migrations (golang-migrate)
├── docs/                 # Documentation (MIGRATIONS.md, QUEUE_USAGE.md)
└── scripts/migrate.sh    # Database migration helper script
```

## Tech Stack

- **Language**: Go 1.24
- **HTTP Router**: chi/v5
- **Database**: PostgreSQL (pgx/v5)
- **Queue**: Redis (go-redis/v9)
- **Git Operations**: go-git/v5

## Running the Application

**Prerequisites**: PostgreSQL and Redis running locally (or configure via `.env`)

```bash
# 1. Copy environment template
cp .env.example .env

# 2. Run database migrations
./scripts/migrate.sh up

# 3. Start API server (terminal 1)
go run cmd/api/main.go

# 4. Start worker (terminal 2)
go run cmd/worker/main.go
```

## Key API Endpoints

| Method | Endpoint                           | Description              |
|--------|-----------------------------------|--------------------------|
| POST   | `/api/v1/repositories`            | Create repository        |
| GET    | `/api/v1/repositories`            | List repositories        |
| GET    | `/api/v1/repositories/{id}`       | Get repository           |
| PATCH  | `/api/v1/repositories/{id}`       | Update repository        |
| POST   | `/api/v1/repositories/{id}/index` | Trigger indexing job     |
| POST   | `/api/v1/repositories/{id}/sync`  | Sync repository          |
| GET    | `/api/v1/repositories/{id}/stats/contributors` | List contributors |
| GET    | `/api/v1/queue/length`            | Get queue length         |
| GET    | `/ping`                           | Health check             |

## Database Migrations

```bash
./scripts/migrate.sh up        # Apply migrations
./scripts/migrate.sh down 1    # Rollback 1 migration
./scripts/migrate.sh version   # Check current version
./scripts/migrate.sh create NAME  # Create new migration
```

## Architecture

1. **API Service** receives requests and publishes jobs to Redis queue
2. **Worker Service** consumes jobs from queue and processes repositories
3. Jobs are processed asynchronously with configurable concurrency (`WORKER_CONCURRENCY`)

## Environment Variables

Key variables (see `.env.example` for full list):

| Variable           | Description                        | Default                   |
|-------------------|------------------------------------|---------------------------|
| `PORT`            | API server port                    | 8080                      |
| `DATABASE_URL`    | PostgreSQL connection string       | postgres://localhost/git_analytics |
| `REDIS_ADDR`      | Redis host:port                    | localhost:6379            |
| `REDIS_QUEUE_NAME`| Queue name for jobs                | git_index_jobs            |
| `WORKER_CONCURRENCY` | Number of concurrent workers    | 5                         |
| `GIT_STORAGE_PATH`| Path to store cloned repos         | /var/lib/git-analytics/repos |

## Development Notes

- Git service supports GitHub and BitBucket (`internal/git/`)
- Repository statuses: `pending`, `indexing`, `completed`, `failed`
- Workers can be scaled horizontally (multiple instances) or vertically (`WORKER_CONCURRENCY`)
