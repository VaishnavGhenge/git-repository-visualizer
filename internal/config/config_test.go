package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Set test environment variables
	os.Setenv("PORT", "9000")
	os.Setenv("DATABASE_URL", "postgres://test/db")
	os.Setenv("REDIS_ADDR", "redis:6379")
	os.Setenv("WORKER_CONCURRENCY", "10")
	os.Setenv("POLL_INTERVAL", "1h")

	cfg := Load()

	// Test server config
	if cfg.Port != "9000" {
		t.Errorf("Expected Port to be 9000, got %s", cfg.Port)
	}

	// Test database config
	if cfg.Database.ConnectionString != "postgres://test/db" {
		t.Errorf("Expected Database.ConnectionString to be postgres://test/db, got %s", cfg.Database.ConnectionString)
	}

	// Test Redis config
	if cfg.Redis.Address != "redis:6379" {
		t.Errorf("Expected Redis.Address to be redis:6379, got %s", cfg.Redis.Address)
	}

	// Test worker config
	if cfg.Worker.Concurrency != 10 {
		t.Errorf("Expected Worker.Concurrency to be 10, got %d", cfg.Worker.Concurrency)
	}

	if cfg.Worker.PollInterval != 1*time.Hour {
		t.Errorf("Expected Worker.PollInterval to be 1h, got %v", cfg.Worker.PollInterval)
	}

	// Clean up
	os.Unsetenv("PORT")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("REDIS_ADDR")
	os.Unsetenv("WORKER_CONCURRENCY")
	os.Unsetenv("POLL_INTERVAL")
}

func TestLoadDefaults(t *testing.T) {
	// Clear any existing env vars
	os.Unsetenv("PORT")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("REDIS_ADDR")

	cfg := Load()

	// Test defaults
	if cfg.Port != "8080" {
		t.Errorf("Expected default Port to be 8080, got %s", cfg.Port)
	}

	if cfg.Database.ConnectionString != "postgres://localhost/git_analytics?sslmode=disable" {
		t.Errorf("Expected default Database.ConnectionString, got %s", cfg.Database.ConnectionString)
	}

	if cfg.Redis.Address != "localhost:6379" {
		t.Errorf("Expected default Redis.Address to be localhost:6379, got %s", cfg.Redis.Address)
	}

	if cfg.Worker.Concurrency != 5 {
		t.Errorf("Expected default Worker.Concurrency to be 5, got %d", cfg.Worker.Concurrency)
	}

	if cfg.Worker.PollInterval != 6*time.Hour {
		t.Errorf("Expected default Worker.PollInterval to be 6h, got %v", cfg.Worker.PollInterval)
	}
}
