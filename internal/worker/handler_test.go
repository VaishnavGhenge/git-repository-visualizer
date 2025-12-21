package worker

import (
	"context"
	"testing"
	"time"

	"git-repository-visualizer/internal/auth"
	"git-repository-visualizer/internal/database"
	"git-repository-visualizer/internal/queue"

	"github.com/pashagolub/pgxmock/v3"
	"golang.org/x/oauth2"
)

type mockProvider struct {
	repos []auth.RemoteRepo
}

func (m *mockProvider) Name() string                 { return "mock" }
func (m *mockProvider) LoginURL(state string) string { return "" }
func (m *mockProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return nil, nil
}
func (m *mockProvider) FetchProfile(ctx context.Context, token *oauth2.Token) (*auth.Profile, error) {
	return nil, nil
}
func (m *mockProvider) FetchRepositories(ctx context.Context, token *oauth2.Token) ([]auth.RemoteRepo, error) {
	return m.repos, nil
}

func TestHandleDiscoverJob(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mockPool.Close()

	db := database.NewTestDB(mockPool)
	registry := auth.NewRegistry()

	mp := &mockProvider{
		repos: []auth.RemoteRepo{
			{ID: "1", FullName: "test/repo1", URL: "url1", DefaultBranch: "main"},
		},
	}
	registry.Register(mp)

	handler := NewJobHandler(db, "/tmp", registry)
	ctx := context.Background()

	userID := int64(42)
	job := &queue.Job{
		Type: queue.JobTypeDiscover,
		Payload: map[string]interface{}{
			"user_id":  float64(userID),
			"provider": "mock",
		},
	}

	// 1. Get identity
	accessToken := "secret-token"
	mockPool.ExpectQuery("SELECT id, user_id, provider, provider_user_id").
		WithArgs(userID, "mock").
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "provider", "provider_user_id", "access_token", "refresh_token", "token_expiry", "created_at", "provider_username"}).
			AddRow(int64(1), userID, "mock", "uid", &accessToken, nil, nil, time.Now(), nil))

	// 2. Check existing repo
	mockPool.ExpectQuery("SELECT id, url, local_path, default_branch, status, last_indexed_at, created_at, updated_at, user_id, name, description, is_private, provider").
		WithArgs("url1").
		WillReturnError(database.ErrNotFound) // Using a constant if available or string

	// 3. Create repo
	mockPool.ExpectQuery("INSERT INTO repositories").
		WithArgs("url1", database.StatusDiscovered, "main", &userID, "test/repo1", "", false, "mock").
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(int64(100), time.Now(), time.Now()))

	err = handler.HandleJob(ctx, job)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
