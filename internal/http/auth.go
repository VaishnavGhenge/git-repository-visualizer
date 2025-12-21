package http

import (
	"fmt"
	"log" // Added for log.Printf
	"net/http"

	"git-repository-visualizer/internal/database"
	// Added queue import
	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"
)

// GoogleLogin handles the initial redirect to Google OAuth
func (h *Handler) AuthLogin(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	p, err := h.authRegistry.Get(providerName)
	if err != nil {
		Error(w, err, http.StatusBadRequest)
		return
	}

	// In a real app, generate and store a state for CSRF protection
	state := "random-state"
	url := p.LoginURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// AuthCallback handles the OAuth2 callback from any provider
func (h *Handler) AuthCallback(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	p, err := h.authRegistry.Get(providerName)
	if err != nil {
		Error(w, err, http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		Error(w, fmt.Errorf("missing code"), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	token, err := p.Exchange(ctx, code)
	if err != nil {
		Error(w, fmt.Errorf("failed to exchange token: %w", err), http.StatusInternalServerError)
		return
	}

	profile, err := p.FetchProfile(ctx, token)
	if err != nil {
		Error(w, fmt.Errorf("failed to fetch profile: %w", err), http.StatusInternalServerError)
		return
	}

	// Prepare user and identity for upsert
	user := &database.User{
		Email:     profile.Email,
		Name:      profile.Name,
		AvatarURL: &profile.AvatarURL,
	}

	expiry := token.Expiry
	identity := &database.UserIdentity{
		Provider:         providerName,
		ProviderUserID:   profile.ID,
		ProviderUsername: &profile.Username,
		AccessToken:      &token.AccessToken,
		RefreshToken:     &token.RefreshToken,
		TokenExpiry:      &expiry,
	}

	err = h.db.UpsertUserByIdentity(ctx, user, identity)
	if err != nil {
		Error(w, fmt.Errorf("failed to sync user: %w", err), http.StatusInternalServerError)
		return
	}

	// Trigger repository discovery in background
	if err := h.publisher.PublishDiscoverJob(ctx, user.ID, providerName); err != nil {
		log.Printf("Failed to publish discovery job for user %d: %v", user.ID, err)
		// Don't fail login if discovery fails
	}

	// Generate JWT
	jwtToken, err := h.jwtManager.GenerateToken(user.ID, user.Email, user.Name)
	if err != nil {
		Error(w, fmt.Errorf("failed to generate token: %w", err), http.StatusInternalServerError)
		return
	}

	// Return token in response (or set as cookie)
	JSON(w, http.StatusOK, map[string]string{
		"token": jwtToken,
		"type":  "Bearer",
	})
}

// GetProviderRepositories fetches repositories from a git hosting provider
func (h *Handler) GetProviderRepositories(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	p, err := h.authRegistry.Get(providerName)
	if err != nil {
		Error(w, err, http.StatusBadRequest)
		return
	}

	user := GetUserFromContext(r.Context())
	if user == nil {
		Error(w, fmt.Errorf("unauthorized"), http.StatusUnauthorized)
		return
	}

	ctx := r.Context()
	identity, err := h.db.GetUserIdentity(ctx, user.ID, providerName)
	if err != nil {
		Error(w, fmt.Errorf("identity for provider %s not found: %w", providerName, err), http.StatusNotFound)
		return
	}

	if identity.AccessToken == nil {
		Error(w, fmt.Errorf("no access token for provider %s", providerName), http.StatusPreconditionFailed)
		return
	}

	// Reconstruct oauth2 token
	token := &oauth2.Token{
		AccessToken: *identity.AccessToken,
	}
	if identity.RefreshToken != nil {
		token.RefreshToken = *identity.RefreshToken
	}
	if identity.TokenExpiry != nil {
		token.Expiry = *identity.TokenExpiry
	}

	repos, err := p.FetchRepositories(ctx, token)
	if err != nil {
		Error(w, fmt.Errorf("failed to fetch repositories: %w", err), http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, repos)
}
