package config

type RedisConfig struct {
	Address   string
	Password  string
	DB        int
	QueueName string
}

func loadRedisConfig() RedisConfig {
	return RedisConfig{
		Address:   getEnv("REDIS_ADDR", "localhost:6379"),
		Password:  getEnv("REDIS_PASSWORD", ""),
		DB:        getEnvInt("REDIS_DB", 0),
		QueueName: getEnv("REDIS_QUEUE_NAME", "git_index_jobs"),
	}
}
