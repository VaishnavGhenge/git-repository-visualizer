package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"git-repository-visualizer/internal/database"
	"git-repository-visualizer/internal/validation"

	"github.com/go-chi/chi/v5"
)

// CreateRepositoryRequest represents the request body for creating a repository
type CreateRepositoryRequest struct {
	URL           string `json:"url"`
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
	repo := &database.Repository{
		URL:           req.URL,
		DefaultBranch: req.DefaultBranch,
		Status:        database.StatusPending,
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
	repo, err := h.db.GetRepository(ctx, id)
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
	repos, err := h.db.ListRepositories(ctx, limit, offset)
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

	// Verify repository exists
	repo, err := h.db.GetRepository(ctx, id)
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

// UpdateRepository handles POST /api/v1/repositories/{id}/update
func (h *Handler) UpdateRepository(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Error(w, fmt.Errorf("invalid repository ID"), http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Verify repository exists
	repo, err := h.db.GetRepository(ctx, id)
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
