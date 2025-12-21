package auth

import (
	"testing"
)

func TestJWTManager(t *testing.T) {
	secret := "test-secret"
	m := NewJWTManager(secret)

	userID := int64(42)
	email := "user@example.com"
	name := "Test User"

	token, err := m.GenerateToken(userID, email, name)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := m.VerifyToken(token)
	if err != nil {
		t.Fatalf("failed to verify token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected userID %d, got %d", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("expected email %s, got %s", email, claims.Email)
	}
	if claims.Name != name {
		t.Errorf("expected name %s, got %s", name, claims.Name)
	}
}

func TestVerifyInvalidToken(t *testing.T) {
	m := NewJWTManager("secret")
	_, err := m.VerifyToken("invalid.token.here")
	if err == nil {
		t.Error("expected error for invalid token, got nil")
	}
}
