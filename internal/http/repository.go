package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"git-repository-visualizer/internal/database"
	"git-repository-visualizer/internal/validation"

	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"
)

// CreateRepositoryRequest represents the request body for creating a repository
type CreateRepositoryRequest struct {
	URL           string `json:"url"`
	DefaultBranch string `json:"default_branch"`
}

type UpdateRepositoryRequest struct {
	DefaultBranch string `json:"default_branch"`
}

// CreateRepository handles POST /api/v1/repositories
func (h *Handler) CreateRepository(w http.ResponseWriter, r *http.Request) {
	var req CreateRepositoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, fmt.Errorf("invalid request body"), http.StatusBadRequest)
		return
	}

	// Validate request
	v := validation.New()
	v.Required("url", req.URL).GitURL("url", req.URL)

	if req.DefaultBranch != "" {
		v.MinLength("default_branch", req.DefaultBranch, 1).
			MaxLength("default_branch", req.DefaultBranch, 255)
	}

	if err := v.Validate(); err != nil {
		Error(w, err, http.StatusBadRequest)
		return
	}

	// Set default branch if not provided
	if req.DefaultBranch == "" {
		req.DefaultBranch = "main"
	}

	ctx := r.Context()
	user := GetUserFromContext(ctx)
	if user == nil {
		Error(w, fmt.Errorf("unauthorized"), http.StatusUnauthorized)
		return
	}

	repo := &database.Repository{
		URL:           req.URL,
		DefaultBranch: req.DefaultBranch,
		Status:        database.StatusPending,
		UserID:        &user.ID,
	}

	// Create repository - database errors (like unique violations) are handled by Error()
	if err := h.db.CreateRepository(ctx, repo); err != nil {
		// Parse and return appropriate error
		parsedErr := validation.ParseDatabaseError(err)
		Error(w, parsedErr, http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusCreated, repo)
}

// UpdateRepository handles PATCH /api/v1/repositories/{id}
func (h *Handler) UpdateRepository(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Error(w, fmt.Errorf("invalid repository ID"), http.StatusBadRequest)
		return
	}

	// Validate ID is positive
	v := validation.New()
	v.GreaterThan("id", int(id), 0)
	if err := v.Validate(); err != nil {
		Error(w, err, http.StatusBadRequest)
		return
	}

	var req UpdateRepositoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, fmt.Errorf("invalid request body"), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	user := GetUserFromContext(ctx)
	if user == nil {
		Error(w, fmt.Errorf("unauthorized"), http.StatusUnauthorized)
		return
	}

	repo, err := h.db.GetRepositoryForUser(ctx, id, user.ID)
	if err != nil {
		// If not found for user, return 404 (don't leak existence)
		Error(w, fmt.Errorf("repository not found"), http.StatusNotFound)
		return
	}

	repo.DefaultBranch = req.DefaultBranch

	// Update repository
	if err := h.db.UpdateRepository(ctx, repo); err != nil {
		Error(w, fmt.Errorf("failed to update repository: %w", err), http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, repo)
}

// GetRepository handles GET /api/v1/repositories/{id}
func (h *Handler) GetRepository(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Error(w, fmt.Errorf("invalid repository ID"), http.StatusBadRequest)
		return
	}

	// Validate ID is positive
	v := validation.New()
	v.GreaterThan("id", int(id), 0)
	if err := v.Validate(); err != nil {
		Error(w, err, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	user := GetUserFromContext(ctx)
	if user == nil {
		Error(w, fmt.Errorf("unauthorized"), http.StatusUnauthorized)
		return
	}

	repo, err := h.db.GetRepositoryForUser(ctx, id, user.ID)
	if err != nil {
		parsedErr := validation.ParseDatabaseError(err)

		// Check if it's a not found error
		if validation.IsNotFound(err) {
			Error(w, parsedErr, http.StatusNotFound)
			return
		}

		Error(w, parsedErr, http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, repo)
}

// GetRepositoryStatus handles GET /api/v1/repositories/{id}/status
func (h *Handler) GetRepositoryStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Error(w, fmt.Errorf("invalid repository ID"), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	user := GetUserFromContext(ctx)
	if user == nil {
		Error(w, fmt.Errorf("unauthorized"), http.StatusUnauthorized)
		return
	}

	status, err := h.db.GetRepositoryStatus(ctx, id, user.ID)
	if err != nil {
		if validation.IsNotFound(err) {
			Error(w, fmt.Errorf("repository not found"), http.StatusNotFound)
			return
		}
		Error(w, fmt.Errorf("failed to get repository status: %w", err), http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"status":        status,
		"repository_id": id,
	})
}

// ListRepositories handles GET /api/v1/repositories
func (h *Handler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	// Get limit and offset from query parameters
	limit := 10
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Validate pagination parameters
	v := validation.New()
	v.InRange("limit", limit, 1, 100).
		GreaterThanOrEqual("offset", offset, 0)

	if err := v.Validate(); err != nil {
		Error(w, err, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	user := GetUserFromContext(ctx)
	if user == nil {
		Error(w, fmt.Errorf("unauthorized"), http.StatusUnauthorized)
		return
	}

	repos, err := h.db.ListRepositories(ctx, user.ID, limit, offset)
	if err != nil {
		parsedErr := validation.ParseDatabaseError(err)
		Error(w, parsedErr, http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, repos)
}

// IndexRepository handles POST /api/v1/repositories/{id}/index
func (h *Handler) IndexRepository(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Error(w, fmt.Errorf("invalid repository ID"), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	user := GetUserFromContext(ctx)
	if user == nil {
		Error(w, fmt.Errorf("unauthorized"), http.StatusUnauthorized)
		return
	}

	// Verify repository exists and belongs to user
	repo, err := h.db.GetRepositoryForUser(ctx, id, user.ID)
	if err != nil {
		Error(w, fmt.Errorf("repository not found"), http.StatusNotFound)
		return
	}

	// Publish index job to queue
	if err := h.publisher.PublishIndexJob(ctx, int(id)); err != nil {
		Error(w, fmt.Errorf("failed to queue index job: %w", err), http.StatusInternalServerError)
		return
	}

	// Update repository status to pending
	if err := h.db.UpdateRepositoryStatus(ctx, id, database.StatusPending); err != nil {
		Error(w, fmt.Errorf("failed to update repository status: %w", err), http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusAccepted, map[string]interface{}{
		"message":       "index job queued successfully",
		"repository_id": id,
		"repository":    repo,
	})
}

// SyncRepository handles POST /api/v1/repositories/{id}/sync
func (h *Handler) SyncRepository(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Error(w, fmt.Errorf("invalid repository ID"), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	user := GetUserFromContext(ctx)
	if user == nil {
		Error(w, fmt.Errorf("unauthorized"), http.StatusUnauthorized)
		return
	}

	// Verify repository exists and belongs to user
	repo, err := h.db.GetRepositoryForUser(ctx, id, user.ID)
	if err != nil {
		Error(w, fmt.Errorf("repository not found"), http.StatusNotFound)
		return
	}

	// Publish update job to queue
	if err := h.publisher.PublishUpdateJob(ctx, int(id)); err != nil {
		Error(w, fmt.Errorf("failed to queue update job: %w", err), http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusAccepted, map[string]interface{}{
		"message":       "Update job queued successfully",
		"repository_id": id,
		"repository":    repo,
	})
}

// GetQueueLength handles GET /api/v1/queue/length
func (h *Handler) GetQueueLength(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	length, err := h.publisher.GetQueueLength(ctx)
	if err != nil {
		Error(w, fmt.Errorf("failed to get queue length: %w", err), http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"queue_length": length,
	})
}

// SyncUserRepositories handles POST /api/v1/repositories/sync
func (h *Handler) SyncUserRepositories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := GetUserFromContext(ctx)
	if user == nil {
		Error(w, fmt.Errorf("unauthorized"), http.StatusUnauthorized)
		return
	}

	// 1. Get all user identities
	identities, err := h.db.GetUserIdentities(ctx, user.ID)
	if err != nil {
		Error(w, fmt.Errorf("failed to load user identities: %w", err), http.StatusInternalServerError)
		return
	}

	if len(identities) == 0 {
		JSON(w, http.StatusOK, map[string]interface{}{
			"message": "No linked providers found",
			"synced":  0,
		})
		return
	}

	totalSynced := 0
	errors := []string{}

	// 2. Iterate over identities and fetch repos
	for _, identity := range identities {
		provider, err := h.authRegistry.Get(identity.Provider)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: provider not configured", identity.Provider))
			continue
		}

		// Reconstruct token structure from DB
		token := &oauth2.Token{
			AccessToken:  *identity.AccessToken,
			TokenType:    "Bearer",
			RefreshToken: "", // Note: We might need refresh token handling if access token expired
		}
		if identity.RefreshToken != nil {
			token.RefreshToken = *identity.RefreshToken
		}
		if identity.TokenExpiry != nil {
			token.Expiry = *identity.TokenExpiry
		}

		// Fetch repositories from provider
		remoteRepos, err := provider.FetchRepositories(ctx, token)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", identity.Provider, err))
			continue
		}

		// 3. Upsert fetched repositories
		for _, remote := range remoteRepos {
			name := remote.FullName
			if name == "" {
				name = remote.Name
			}
			desc := remote.Description
			providerStr := identity.Provider

			repo := &database.Repository{
				URL:           remote.URL,
				UserID:        &user.ID,
				Name:          &name,
				Description:   &desc,
				IsPrivate:     remote.IsPrivate,
				Provider:      &providerStr,
				DefaultBranch: remote.DefaultBranch,
				Status:        database.StatusDiscovered,
				LastPushedAt:  remote.PushedAt,
			}

			// We use UpsertRepository which handles ON CONFLICT DO UPDATE
			if err := h.db.UpsertRepository(ctx, repo); err != nil {
				// Log error but continue syncing other repos
				// In production you might want structured logging here
				continue
			}
			totalSynced++
		}
	}

	response := map[string]interface{}{
		"message": fmt.Sprintf("Synced %d repositories", totalSynced),
		"synced":  totalSynced,
	}
	if len(errors) > 0 {
		response["errors"] = errors
	}

	JSON(w, http.StatusOK, response)
}
