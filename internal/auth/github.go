package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"git-repository-visualizer/internal/config"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type gitHubProvider struct {
	config *oauth2.Config
}

func NewGitHubProvider(cfg config.ProviderConfig) Provider {
	return &gitHubProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     github.Endpoint,
			Scopes: []string{
				"read:user",
				"user:email",
				"repo", // For fetching user repositories including private ones
			},
		},
	}
}

func (p *gitHubProvider) Name() string {
	return "github"
}

func (p *gitHubProvider) LoginURL(state string) string {
	return p.config.AuthCodeURL(state)
}

func (p *gitHubProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.config.Exchange(ctx, code)
}

func (p *gitHubProvider) FetchProfile(ctx context.Context, token *oauth2.Token) (*Profile, error) {
	client := p.config.Client(ctx, token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch github profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var data struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode github profile: %w", err)
	}

	// GitHub may not return email in /user if it's private, we might need /user/emails
	if data.Email == "" {
		emailResp, err := client.Get("https://api.github.com/user/emails")
		if err == nil {
			defer emailResp.Body.Close()
			var emails []struct {
				Email   string `json:"email"`
				Primary bool   `json:"primary"`
			}
			if err := json.NewDecoder(emailResp.Body).Decode(&emails); err == nil {
				for _, e := range emails {
					if e.Primary {
						data.Email = e.Email
						break
					}
				}
			}
		}
	}

	displayName := data.Name
	if displayName == "" {
		displayName = data.Login
	}

	return &Profile{
		ID:        fmt.Sprintf("%d", data.ID),
		Email:     data.Email,
		Name:      displayName,
		Username:  data.Login,
		AvatarURL: data.AvatarURL,
	}, nil
}

func (p *gitHubProvider) FetchRepositories(ctx context.Context, token *oauth2.Token) ([]RemoteRepo, error) {
	client := p.config.Client(ctx, token)
	resp, err := client.Get("https://api.github.com/user/repos?per_page=100&sort=updated")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch github repositories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var data []struct {
		ID            int    `json:"id"`
		Name          string `json:"name"`
		FullName      string `json:"full_name"`
		Description   string `json:"description"`
		HTMLURL       string `json:"html_url"`
		DefaultBranch string `json:"default_branch"`
		Private       bool   `json:"private"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode github repositories: %w", err)
	}

	repos := make([]RemoteRepo, len(data))
	for i, r := range data {
		repos[i] = RemoteRepo{
			ID:            fmt.Sprintf("%d", r.ID),
			Name:          r.Name,
			FullName:      r.FullName,
			Description:   r.Description,
			URL:           r.HTMLURL,
			DefaultBranch: r.DefaultBranch,
			IsPrivate:     r.Private,
		}
	}

	return repos, nil
}
