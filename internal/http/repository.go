package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"git-repository-visualizer/internal/database"

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

	if req.URL == "" {
		Error(w, fmt.Errorf("repository URL is required"), http.StatusBadRequest)
		return
	}

	if req.DefaultBranch == "" {
		req.DefaultBranch = "main"
	}

	ctx := r.Context()
	repo := &database.Repository{
		URL:           req.URL,
		DefaultBranch: req.DefaultBranch,
		Status:        database.StatusPending,
	}

	if err := h.db.CreateRepository(ctx, repo); err != nil {
		Error(w, fmt.Errorf("failed to create repository"), http.StatusInternalServerError)
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

	ctx := r.Context()
	repo, err := h.db.GetRepository(ctx, id)
	if err != nil {
		Error(w, fmt.Errorf("repository not found"), http.StatusNotFound)
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

	ctx := r.Context()
	repos, err := h.db.ListRepositories(ctx, limit, offset)
	if err != nil {
		Error(w, fmt.Errorf("failed to list repositories"), http.StatusInternalServerError)
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
