package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"git-repository-visualizer/internal/auth"
	"git-repository-visualizer/internal/config"
	"git-repository-visualizer/internal/database"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"golang.org/x/oauth2"
)

type mockPublisher struct{}

func (m *mockPublisher) PublishIndexJob(ctx context.Context, repoID int) error  { return nil }
func (m *mockPublisher) PublishUpdateJob(ctx context.Context, repoID int) error { return nil }
func (m *mockPublisher) PublishDiscoverJob(ctx context.Context, userID int64, provider string) error {
	return nil
}
func (m *mockPublisher) GetQueueLength(ctx context.Context) (int64, error) { return 0, nil }

type mockAuthProvider struct {
	profile *auth.Profile
}

func (m *mockAuthProvider) Name() string                 { return "mock" }
func (m *mockAuthProvider) LoginURL(state string) string { return "http://login.url" }
func (m *mockAuthProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "token"}, nil
}
func (m *mockAuthProvider) FetchProfile(ctx context.Context, token *oauth2.Token) (*auth.Profile, error) {
	return m.profile, nil
}
func (m *mockAuthProvider) FetchRepositories(ctx context.Context, token *oauth2.Token) ([]auth.RemoteRepo, error) {
	return nil, nil
}

func TestAuthCallback(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mockPool.Close()

	db := database.NewTestDB(mockPool)
	publisher := &mockPublisher{}
	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret: "test-secret",
		},
	}

	h := NewHandler(db, publisher, cfg)

	// Register mock provider
	mp := &mockAuthProvider{
		profile: &auth.Profile{
			ID:    "uid123",
			Email: "test@test.com",
			Name:  "Test User",
		},
	}
	h.authRegistry.Register(mp)

	// Mock DB calls in UpsertUserByIdentity
	mockPool.ExpectBegin()
	mockPool.ExpectQuery("SELECT user_id FROM user_identities").
		WithArgs("mock", "uid123").
		WillReturnError(pgx.ErrNoRows)
	mockPool.ExpectQuery("SELECT id, email, name, avatar_url").
		WithArgs("test@test.com").
		WillReturnError(pgx.ErrNoRows)
	mockPool.ExpectQuery("INSERT INTO users").
		WithArgs("test@test.com", "Test User", pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(1), time.Now(), time.Now()))
	mockPool.ExpectExec("INSERT INTO user_identities").
		WithArgs(int64(1), "mock", "uid123", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mockPool.ExpectCommit()

	req, _ := http.NewRequest("GET", "/api/v1/auth/mock/callback?code=123", nil)
	rr := httptest.NewRecorder()

	// Direct call to handler to avoid complex routing setup if possible,
	// but we need chi params if we use the router.
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp["token"] == "" {
		t.Error("expected token in response, got empty")
	}

	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
