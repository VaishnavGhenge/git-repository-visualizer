package database

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v3"
)

func TestCreateRepository(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := NewTestDB(mock)
	ctx := context.Background()

	userID := int64(1)
	name := "test/repo"
	description := "A test repo"
	provider := "github"
	repo := &Repository{
		UserID:      &userID,
		URL:         "https://github.com/test/repo",
		Name:        &name,
		Description: &description,
		IsPrivate:   true,
		Provider:    &provider,
		Status:      StatusDiscovered,
	}

	mock.ExpectQuery("INSERT INTO repositories").
		WithArgs(repo.URL, repo.Status, "main", repo.UserID, repo.Name, repo.Description, repo.IsPrivate, repo.Provider).
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(int64(10), time.Now(), time.Now()))

	err = db.CreateRepository(ctx, repo)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if repo.ID != 10 {
		t.Errorf("expected repo ID 10, got %d", repo.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetRepository(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := NewTestDB(mock)
	ctx := context.Background()

	expectedID := int64(10)
	mock.ExpectQuery("SELECT id, url, local_path, default_branch, status, last_indexed_at, created_at, updated_at, user_id, name, description, is_private, provider").
		WithArgs(expectedID).
		WillReturnRows(pgxmock.NewRows([]string{"id", "url", "local_path", "default_branch", "status", "last_indexed_at", "created_at", "updated_at", "user_id", "name", "description", "is_private", "provider"}).
			AddRow(expectedID, "url", nil, "main", StatusPending, nil, time.Now(), time.Now(), int64(1), "name", "desc", false, "github"))

	repo, err := db.GetRepository(ctx, expectedID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if repo.ID != expectedID {
		t.Errorf("expected ID %d, got %d", expectedID, repo.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
