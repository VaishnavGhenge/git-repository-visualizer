package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"git-repository-visualizer/internal/config"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type googleProvider struct {
	config *oauth2.Config
}

func NewGoogleProvider(cfg config.ProviderConfig) Provider {
	return &googleProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     google.Endpoint,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.profile",
				"https://www.googleapis.com/auth/userinfo.email",
			},
		},
	}
}

func (p *googleProvider) Name() string {
	return "google"
}

func (p *googleProvider) LoginURL(state string) string {
	return p.config.AuthCodeURL(state)
}

func (p *googleProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.config.Exchange(ctx, code)
}

func (p *googleProvider) FetchProfile(ctx context.Context, token *oauth2.Token) (*Profile, error) {
	client := p.config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch google profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google api returned status %d", resp.StatusCode)
	}

	var data struct {
		Sub     string `json:"sub"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode google profile: %w", err)
	}

	return &Profile{
		ID:        data.Sub,
		Email:     data.Email,
		Name:      data.Name,
		Username:  "", // Google doesn't have a "username" typically
		AvatarURL: data.Picture,
	}, nil
}

func (p *googleProvider) FetchRepositories(ctx context.Context, token *oauth2.Token) ([]RemoteRepo, error) {
	return nil, fmt.Errorf("google is not a git hosting provider")
}
