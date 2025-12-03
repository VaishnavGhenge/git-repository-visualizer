# Queue System Usage

This document explains how to use the queue-based architecture for processing git repositories.

## Architecture Overview

The system consists of two main components:

1. **API Service** (`cmd/api`) - HTTP API that accepts repository management requests and publishes jobs to Redis queue
2. **Worker Service** (`cmd/worker`) - Background worker that consumes jobs from Redis queue and processes repositories

## Services

### API Service

The API service provides HTTP endpoints for managing repositories and triggering indexing jobs.

**Start the API:**
```bash
go run cmd/api/main.go
```

**Environment Variables:**
```bash
PORT=8080
DATABASE_URL=postgres://localhost/git_analytics?sslmode=disable
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_QUEUE_NAME=git_index_jobs
```

### Worker Service

The worker service processes jobs from the Redis queue.

**Start the Worker:**
```bash
go run cmd/worker/main.go
```

**Environment Variables:**
```bash
DATABASE_URL=postgres://localhost/git_analytics?sslmode=disable
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_QUEUE_NAME=git_index_jobs
WORKER_CONCURRENCY=5
GIT_STORAGE_PATH=/var/lib/git-analytics/repos
```

## API Endpoints

### Create Repository
```bash
POST /api/v1/repositories
Content-Type: application/json

{
  "url": "https://github.com/user/repo.git",
  "default_branch": "main"
}
```

### Get Repository
```bash
GET /api/v1/repositories/{id}
```

### Trigger Repository Indexing
```bash
POST /api/v1/repositories/{id}/index
```

This endpoint:
1. Validates the repository exists
2. Publishes an index job to the Redis queue
3. Updates repository status to "pending"
4. Returns immediately (job runs asynchronously)

### Trigger Repository Update
```bash
POST /api/v1/repositories/{id}/update
```

This endpoint:
1. Validates the repository exists
2. Publishes an update job to the Redis queue
3. Returns immediately (job runs asynchronously)

### Check Queue Length
```bash
GET /api/v1/queue/length
```

Returns the current number of jobs in the queue.

## Job Types

The system supports three job types:

1. **Index Job** - Clone and analyze a repository for the first time
2. **Update Job** - Pull latest changes and re-analyze an existing repository
3. **Delete Job** - Clean up repository data

## Job Flow

1. API receives request (e.g., POST /api/v1/repositories/1/index)
2. API creates a job and pushes it to Redis queue (RPUSH)
3. API updates repository status to "pending"
4. Worker pops job from queue (BLPOP with timeout)
5. Worker updates repository status to "indexing"
6. Worker processes the repository
7. Worker updates repository status to "completed" or "failed"

## Repository Statuses

- `pending` - Job queued, waiting to be processed
- `indexing` - Currently being processed by a worker
- `completed` - Successfully processed
- `failed` - Processing failed

## Example Usage

```bash
# 1. Start Redis
docker run -d -p 6379:6379 redis:7-alpine

# 2. Start the database
docker run -d -p 5432:5432 -e POSTGRES_DB=git_analytics -e POSTGRES_PASSWORD=postgres postgres:15

# 3. Start the API
PORT=8080 DATABASE_URL="postgres://postgres:postgres@localhost/git_analytics?sslmode=disable" go run cmd/api/main.go

# 4. Start the worker (in another terminal)
DATABASE_URL="postgres://postgres:postgres@localhost/git_analytics?sslmode=disable" WORKER_CONCURRENCY=5 go run cmd/worker/main.go

# 5. Create a repository
curl -X POST http://localhost:8080/api/v1/repositories \
  -H "Content-Type: application/json" \
  -d '{"url": "https://github.com/torvalds/linux.git", "default_branch": "master"}'

# 6. Trigger indexing
curl -X POST http://localhost:8080/api/v1/repositories/1/index

# 7. Check queue length
curl http://localhost:8080/api/v1/queue/length

# 8. Check repository status
curl http://localhost:8080/api/v1/repositories/1
```

## Scaling

### Horizontal Scaling

You can run multiple worker instances to process jobs in parallel:

```bash
# Terminal 1
WORKER_CONCURRENCY=5 go run cmd/worker/main.go

# Terminal 2
WORKER_CONCURRENCY=5 go run cmd/worker/main.go

# Terminal 3
WORKER_CONCURRENCY=5 go run cmd/worker/main.go
```

Each worker will consume jobs independently from the shared Redis queue.

### Vertical Scaling

Adjust the `WORKER_CONCURRENCY` environment variable to control how many goroutines each worker process uses:

```bash
WORKER_CONCURRENCY=10 go run cmd/worker/main.go
```

## Monitoring

Check worker logs for job processing status:
```
2024/01/15 10:00:00 Worker 0 started
2024/01/15 10:00:05 Worker 0 processing job abc123 (type: index, repo: 1)
2024/01/15 10:00:07 Indexing repository: https://github.com/user/repo.git (ID: 1)
2024/01/15 10:00:10 Successfully indexed repository 1
2024/01/15 10:00:10 Worker 0 completed job abc123
```

## TODO

The current implementation includes placeholders for actual git operations:

- [ ] Implement git clone logic in `handleIndexJob`
- [ ] Implement git pull logic in `handleUpdateJob`
- [ ] Implement repository cleanup in `handleDeleteJob`
- [ ] Add retry logic for failed jobs
- [ ] Implement dead letter queue for permanently failed jobs
- [ ] Add metrics and monitoring
