package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"git-repository-visualizer/internal/config"

	"github.com/redis/go-redis/v9"
)

// Client wraps the Redis client with application-specific methods
type Client struct {
	*redis.Client
}

// NewClient creates a new Redis client based on the configuration
func NewClient(cfg config.RedisConfig) (*Client, error) {
	opts := &redis.Options{
		Addr:       cfg.Address,
		Password:   cfg.Password,
		DB:         cfg.DB,
		Username:   cfg.Username,
		MaxRetries: 3,
	}

	// Enable TLS if configured (required for Redis Cloud)
	if cfg.UseTLS {
		fmt.Printf("Redis TLS enabled for address: %s\n", cfg.Address)
		// Extract hostname from address (remove port) for SNI
		host := strings.Split(cfg.Address, ":")[0]
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: host, // Set SNI for proper TLS handshake
		}
		fmt.Printf("Redis TLS ServerName set to: %s\n", host)
	} else {
		fmt.Printf("Redis TLS disabled for address: %s\n", cfg.Address)
	}

	client := redis.NewClient(opts)

	// Test the connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{Client: client}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.Client.Close()
}

// HealthCheck performs a health check on the Redis connection
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.Ping(ctx).Err()
}
