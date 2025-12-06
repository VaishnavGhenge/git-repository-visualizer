package config

type HTTPConfig struct {
	LIMIT  int
	OFFSET int
}

func loadHTTPConfig() HTTPConfig {
	return HTTPConfig{
		LIMIT:  getEnvInt("HTTP_LIMIT", 10),
		OFFSET: getEnvInt("HTTP_OFFSET", 0),
	}
}