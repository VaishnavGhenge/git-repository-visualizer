#!/bin/bash

# Database Migration Script
# Usage: ./scripts/migrate.sh [up|down|version|force N]

set -e

# Find migrate binary
MIGRATE_BIN=$(which migrate 2>/dev/null || echo "$HOME/go/bin/migrate")
if [ ! -x "$MIGRATE_BIN" ]; then
    echo "Error: migrate binary not found"
    echo "Install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
    echo "Or add ~/go/bin to your PATH"
    exit 1
fi

# Load environment variables if .env exists
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Database URL
DATABASE_URL=${DATABASE_URL:-"postgres://localhost/git_analytics?sslmode=disable"}
MIGRATIONS_PATH="migrations"

case "$1" in
    up)
        echo "Running migrations..."
        "$MIGRATE_BIN" -path "$MIGRATIONS_PATH" -database "$DATABASE_URL" up
        echo "Migrations completed!"
        ;;
    down)
        STEPS=${2:-1}
        echo "Rolling back $STEPS migration(s)..."
        "$MIGRATE_BIN" -path "$MIGRATIONS_PATH" -database "$DATABASE_URL" down "$STEPS"
        echo "Rollback completed!"
        ;;
    version)
        echo "Current migration version:"
        "$MIGRATE_BIN" -path "$MIGRATIONS_PATH" -database "$DATABASE_URL" version
        ;;
    force)
        if [ -z "$2" ]; then
            echo "Error: version number required"
            echo "Usage: ./scripts/migrate.sh force VERSION"
            exit 1
        fi
        echo "Forcing database to version $2..."
        "$MIGRATE_BIN" -path "$MIGRATIONS_PATH" -database "$DATABASE_URL" force "$2"
        echo "Version forced!"
        ;;
    create)
        if [ -z "$2" ]; then
            echo "Error: migration name required"
            echo "Usage: ./scripts/migrate.sh create migration_name"
            exit 1
        fi
        "$MIGRATE_BIN" create -ext sql -dir "$MIGRATIONS_PATH" -seq "$2"
        echo "Migration files created!"
        ;;
    *)
        echo "Database Migration Tool"
        echo ""
        echo "Usage: $0 {up|down|version|force|create} [args]"
        echo ""
        echo "Commands:"
        echo "  up              - Run all pending migrations"
        echo "  down [N]        - Rollback N migrations (default: 1)"
        echo "  version         - Show current migration version"
        echo "  force VERSION   - Force database to specific version"
        echo "  create NAME     - Create new migration files"
        echo ""
        echo "Environment:"
        echo "  DATABASE_URL    - PostgreSQL connection string"
        echo "                    (default: postgres://localhost/git_analytics?sslmode=disable)"
        exit 1
        ;;
esac
