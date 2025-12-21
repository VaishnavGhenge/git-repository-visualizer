package auth

import (
	"context"
	"testing"

	"git-repository-visualizer/internal/config"

	"golang.org/x/oauth2"
)

type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string                 { return m.name }
func (m *mockProvider) LoginURL(state string) string { return "http://login.com" }
func (m *mockProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "token"}, nil
}
func (m *mockProvider) FetchProfile(ctx context.Context, token *oauth2.Token) (*Profile, error) {
	return &Profile{ID: "123", Email: "test@test.com", Name: "Test User"}, nil
}
func (m *mockProvider) FetchRepositories(ctx context.Context, token *oauth2.Token) ([]RemoteRepo, error) {
	return []RemoteRepo{{ID: "repo1", Name: "Repo 1"}}, nil
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	p := &mockProvider{name: "test"}
	r.Register(p)

	got, err := r.Get("test")
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}
	if got.Name() != "test" {
		t.Errorf("expected test, got %s", got.Name())
	}

	_, err = r.Get("missing")
	if err == nil {
		t.Error("expected error for missing provider, got nil")
	}
}

func TestInitializeProviders(t *testing.T) {
	r := NewRegistry()
	cfg := config.AuthConfig{
		Providers: map[string]config.ProviderConfig{
			"google": {ClientID: "g-id"},
			"github": {ClientID: "gh-id"},
		},
	}
	r.InitializeProviders(cfg)

	if _, err := r.Get("google"); err != nil {
		t.Errorf("expected google provider to be registered: %v", err)
	}
	if _, err := r.Get("github"); err != nil {
		t.Errorf("expected github provider to be registered: %v", err)
	}
}
