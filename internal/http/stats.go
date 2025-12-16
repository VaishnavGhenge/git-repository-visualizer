package http

import (
	"fmt"
	"git-repository-visualizer/internal/validation"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// GetBusFactor returns the bus factor analysis for a repository
func (h *Handler) GetBusFactor(w http.ResponseWriter, r *http.Request) {
	repoIDStr := chi.URLParam(r, "repoID")
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

	// Parse optional threshold parameter (default 0.5 = 50%)
	threshold := 0.5
	if thresholdStr := r.URL.Query().Get("threshold"); thresholdStr != "" {
		if parsed, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
			if parsed > 0 && parsed <= 1 {
				threshold = parsed
			}
		}
	}

	ctx := r.Context()
	result, err := h.db.GetBusFactor(ctx, repoID, threshold)
	if err != nil {
		parsedErr := validation.ParseDatabaseError(err)
		Error(w, parsedErr, http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, result)
}
