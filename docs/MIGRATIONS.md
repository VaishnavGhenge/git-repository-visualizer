# Database Migration Guide

## Prerequisites

1. **PostgreSQL must be running**
   ```bash
   # Check if PostgreSQL is running
   pg_isready
   
   # Create database if it doesn't exist
   createdb git_analytics
   ```

2. **Install migrate CLI** (one-time setup)
   ```bash
   # macOS
   brew install golang-migrate
   
   # Or using Go
   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   ```

## Running Migrations

### Method 1: Using the Migration Script (Easiest)

```bash
# Run all pending migrations
./scripts/migrate.sh up

# Check current version
./scripts/migrate.sh version

# Rollback last migration
./scripts/migrate.sh down

# Rollback 2 migrations
./scripts/migrate.sh down 2

# Create new migration
./scripts/migrate.sh create add_new_field
```

### Method 2: Using migrate CLI directly

```bash
# Set database URL (or add to .env file)
export DATABASE_URL="postgres://localhost/git_analytics?sslmode=disable"

# Run migrations
migrate -path migrations -database "$DATABASE_URL" up

# Check version
migrate -path migrations -database "$DATABASE_URL" version

# Rollback
migrate -path migrations -database "$DATABASE_URL" down 1
```

### Method 3: Programmatically (from your Go code)

Add to your `cmd/api/main.go`:

```go
import "git-repository-visualizer/internal/database"

func main() {
    cfg := config.Load()
    
    // Run migrations on startup
    migrationsPath := "migrations"
    if err := database.RunMigrations(cfg.Database.ConnectionString, migrationsPath); err != nil {
        log.Fatalf("Failed to run migrations: %v", err)
    }
    
    // Continue with app initialization...
}
```

## Migration Files

Current migrations:
- `000001_initial_schema.up.sql` - Creates repositories, contributors, file_stats, commits tables
- `000002_add_default_branch.up.sql` - Adds default_branch column to repositories

## Troubleshooting

### Error: "Dirty database version"
This means a migration failed halfway. Fix with:
```bash
# Force to last known good version (check with version command first)
./scripts/migrate.sh force 1
```

### Error: "Database does not exist"
```bash
createdb git_analytics
```

### Error: "No such table"
You need to run migrations:
```bash
./scripts/migrate.sh up
```

## Verifying Migrations

```bash
# Connect to database
psql -d git_analytics

# List tables
\dt

# Check repositories table structure
\d repositories

# Exit psql
\q
```

Expected tables after running all migrations:
- `repositories`
- `contributors`
- `file_stats`
- `commits`
- `schema_migrations` (created automatically by migrate tool)
