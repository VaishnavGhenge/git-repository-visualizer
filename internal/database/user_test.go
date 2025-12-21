package database

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
)

func TestUpsertUserByIdentity(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := NewTestDB(mock)
	ctx := context.Background()

	user := &User{
		Email: "test@example.com",
		Name:  "Test User",
	}
	identity := &UserIdentity{
		Provider:       "github",
		ProviderUserID: "12345",
	}

	// 1. Test New User Insertion
	mock.ExpectBegin()
	// Check identity
	mock.ExpectQuery("SELECT user_id FROM user_identities").
		WithArgs(identity.Provider, identity.ProviderUserID).
		WillReturnError(pgx.ErrNoRows)

	// Check user by email
	mock.ExpectQuery("SELECT id, email, name, avatar_url").
		WithArgs(user.Email).
		WillReturnError(pgx.ErrNoRows)

	// User insertion
	mock.ExpectQuery("INSERT INTO users").
		WithArgs(user.Email, user.Name, user.AvatarURL).
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(int64(1), time.Now(), time.Now()))

	// Identity insertion
	mock.ExpectExec("INSERT INTO user_identities").
		WithArgs(int64(1), identity.Provider, identity.ProviderUserID, identity.AccessToken, identity.RefreshToken, identity.TokenExpiry, identity.ProviderUsername).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	err = db.UpsertUserByIdentity(ctx, user, identity)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if user.ID != 1 {
		t.Errorf("expected user ID 1, got %d", user.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetUserIdentity(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	db := NewTestDB(mock)
	ctx := context.Background()

	expectedID := int64(1)
	expectedProvider := "github"
	expectedProviderUserID := "12345"

	mock.ExpectQuery("SELECT id, user_id, provider, provider_user_id").
		WithArgs(expectedID, expectedProvider).
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "provider", "provider_user_id", "access_token", "refresh_token", "token_expiry", "created_at", "provider_username"}).
			AddRow(int64(10), expectedID, expectedProvider, expectedProviderUserID, nil, nil, nil, time.Now(), nil))

	identity, err := db.GetUserIdentity(ctx, expectedID, expectedProvider)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if identity.ProviderUserID != expectedProviderUserID {
		t.Errorf("expected provider user ID %s, got %s", expectedProviderUserID, identity.ProviderUserID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
