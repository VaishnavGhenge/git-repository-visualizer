package http

import (
	"fmt"
	"git-repository-visualizer/internal/stats"
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

	// Parse query parameters with defaults
	opts := stats.BusFactorOptions{
		Threshold:       0.5,  // Default 50%
		ActiveDays:      0,    // Default: all time
		ExcludePatterns: true, // Default: exclude generated files
	}

	// Parse threshold (0-1)
	if thresholdStr := r.URL.Query().Get("threshold"); thresholdStr != "" {
		if parsed, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
			if parsed > 0 && parsed <= 1 {
				opts.Threshold = parsed
			}
		}
	}

	// Parse days filter (active contributors in last N days)
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if parsed, err := strconv.Atoi(daysStr); err == nil {
			if parsed > 0 {
				opts.ActiveDays = parsed
			}
		}
	}

	// Parse exclude filter (whether to exclude generated files)
	if excludeStr := r.URL.Query().Get("exclude"); excludeStr != "" {
		opts.ExcludePatterns = excludeStr != "false"
	}

	ctx := r.Context()
	result, err := stats.CalculateBusFactor(ctx, h.db.Pool(), repoID, opts)
	if err != nil {
		parsedErr := validation.ParseDatabaseError(err)
		Error(w, parsedErr, http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, result)
}

// GetChurnStats returns the high churn files for a repository
func (h *Handler) GetChurnStats(w http.ResponseWriter, r *http.Request) {
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

	// Parse query parameters
	opts := stats.ChurnOptions{
		Limit: 10, // Default
		Days:  0,  // Default: all time
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			opts.Limit = parsed
		}
	}

	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if parsed, err := strconv.Atoi(daysStr); err == nil && parsed > 0 {
			opts.Days = parsed
		}
	}

	ctx := r.Context()
	result, err := stats.GetHighChurnFiles(ctx, h.db.Pool(), repoID, opts)
	if err != nil {
		Error(w, err, http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, result)
}
