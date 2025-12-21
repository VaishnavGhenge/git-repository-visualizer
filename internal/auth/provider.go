package auth

import (
	"context"
	"fmt"
	"sync"

	"git-repository-visualizer/internal/config"

	"golang.org/x/oauth2"
)

// Profile represents a normalized user profile from any provider
type Profile struct {
	ID        string
	Email     string
	Name      string
	Username  string // e.g. GitHub username
	AvatarURL string
}

// Provider defines the interface for an OAuth2 authentication provider
type Provider interface {
	Name() string
	LoginURL(state string) string
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	FetchProfile(ctx context.Context, token *oauth2.Token) (*Profile, error)
	// FetchRepositories returns a list of repositories for the authenticated user (optional)
	FetchRepositories(ctx context.Context, token *oauth2.Token) ([]RemoteRepo, error)
}

// RemoteRepo represents a repository found on a git hosting provider
type RemoteRepo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	URL           string `json:"url"`
	DefaultBranch string `json:"default_branch"`
	IsPrivate     bool   `json:"is_private"`
}

// Registry maintains a set of available authentication providers
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return p, nil
}

// InitializeProviders registers all configured providers
func (r *Registry) InitializeProviders(cfg config.AuthConfig) {
	if gCfg, ok := cfg.Providers["google"]; ok && gCfg.ClientID != "" {
		r.Register(NewGoogleProvider(gCfg))
	}
	if ghCfg, ok := cfg.Providers["github"]; ok && ghCfg.ClientID != "" {
		r.Register(NewGitHubProvider(ghCfg))
	}
}
