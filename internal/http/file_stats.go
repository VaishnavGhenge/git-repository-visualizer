package http

import (
	"fmt"
	"git-repository-visualizer/internal/validation"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) ListFileStats(w http.ResponseWriter, r *http.Request) {
	repoIDStr := chi.URLParam(r, "repoID")
	// Validate repoID is valid repository ID
	repoID, err := strconv.ParseInt(repoIDStr, 10, 64)
	if err != nil {
		Error(w, fmt.Errorf("invalid repository ID: %w", err), http.StatusBadRequest)
		return
	}

	v := validation.New()
	v.GreaterThan("repoID", int(repoID), 0)
	if err := v.Validate(); err != nil {
		Error(w, err, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	limit, offset := h.GetLimitOffset(r)

	fileStats, err := h.db.GetFileStatsByRepository(ctx, repoID, limit, offset)
	if err != nil {
		parsedErr := validation.ParseDatabaseError(err)
		Error(w, parsedErr, http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"fileStats": fileStats,
	})
}
