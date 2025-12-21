package config

import (
	"os"
	"strconv"
	"time"
)

// Config is the main configuration struct that consolidates all sub-configs
type Config struct {
	Port     string
	Database DatabaseConfig
	Redis    RedisConfig
	Worker   WorkerConfig
	HTTP     HTTPConfig
	Auth     AuthConfig
}

// Load reads all configuration from environment variables and returns the Config
func Load() *Config {
	return &Config{
		Port:     getEnv("PORT", "8080"),
		Database: loadDatabaseConfig(),
		Redis:    loadRedisConfig(),
		Worker:   loadWorkerConfig(),
		HTTP:     loadHTTPConfig(),
		Auth:     loadAuthConfig(),
	}
}

// Helper functions to get environment variables with defaults
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
