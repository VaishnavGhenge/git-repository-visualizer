package config

type ProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type AuthConfig struct {
	JWTSecret string
	Providers map[string]ProviderConfig
}

func loadAuthConfig() AuthConfig {
	return AuthConfig{
		JWTSecret: getEnv("JWT_SECRET", "super-secret-key-change-it"),
		Providers: map[string]ProviderConfig{
			"google": {
				ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
				ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/google/callback"),
			},
			"github": {
				ClientID:     getEnv("GITHUB_CLIENT_ID", ""),
				ClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GITHUB_REDIRECT_URL", "http://localhost:8080/api/v1/auth/github/callback"),
			},
		},
	}
}
