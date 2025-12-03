package redis

import (
	"context"
	"testing"
	"time"

	"git-repository-visualizer/internal/config"
)

func TestNewClient(t *testing.T) {
	// This test requires a running Redis instance
	// Skip if REDIS_ADDR is not set in the environment
	cfg := config.RedisConfig{
		Address:  "localhost:6379",
		Password: "",
		DB:       0,
		UseTLS:   false,
		Username: "default",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
		return
	}
	defer client.Close()

	// Test basic operations
	ctx := context.Background()

	// Set a value
	err = client.Set(ctx, "test_key", "test_value", 10*time.Second).Err()
	if err != nil {
		t.Fatalf("Failed to SET: %v", err)
	}

	// Get the value
	val, err := client.Get(ctx, "test_key").Result()
	if err != nil {
		t.Fatalf("Failed to GET: %v", err)
	}

	if val != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", val)
	}

	// Delete the key
	err = client.Del(ctx, "test_key").Err()
	if err != nil {
		t.Fatalf("Failed to DEL: %v", err)
	}
}

func TestHealthCheck(t *testing.T) {
	cfg := config.RedisConfig{
		Address:  "localhost:6379",
		Password: "",
		DB:       0,
		UseTLS:   false,
		Username: "default",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()
	err = client.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}
