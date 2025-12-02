package config

import (
	"time"
)

type DatabaseConfig struct {
	ConnectionString string
	MaxOpenConns     int
	MaxIdleConns     int
	ConnMaxLifetime  time.Duration
}

func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		ConnectionString: getEnv("DATABASE_URL", "postgres://localhost/git_analytics?sslmode=disable"),
		MaxOpenConns:     getEnvInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:     getEnvInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime:  getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
	}
}
