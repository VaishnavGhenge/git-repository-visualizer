package config

import (
	"time"
)

type WorkerConfig struct {
	Concurrency  int
	StoragePath  string
	PollInterval time.Duration
}

func loadWorkerConfig() WorkerConfig {
	return WorkerConfig{
		Concurrency:  getEnvInt("WORKER_CONCURRENCY", 5),
		StoragePath:  getEnv("GIT_STORAGE_PATH", "/var/lib/git-analytics/repos"),
		PollInterval: getEnvDuration("POLL_INTERVAL", 6*time.Hour),
	}
}
