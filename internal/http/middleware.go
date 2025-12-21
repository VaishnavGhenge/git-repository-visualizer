package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git-repository-visualizer/internal/database"
	"git-repository-visualizer/internal/validation"

	"github.com/go-chi/chi/v5"
)

type contextKey string

const (
	userKey contextKey = "user"
)

// GetUserFromContext retrieves the user from the request context
func GetUserFromContext(ctx context.Context) *database.User {
	user, _ := ctx.Value(userKey).(*database.User)
	return user
}

type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *wrappedWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &wrappedWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		log.Printf("%d %s %s %s", wrapped.statusCode, r.Method, r.URL.Path, time.Since(start))
	})
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware extracts the JWT from the Authorization header and injects the user into the context
func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			Error(w, fmt.Errorf("missing authorization header"), http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			Error(w, fmt.Errorf("invalid authorization header format"), http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]
		claims, err := h.jwtManager.VerifyToken(tokenString)
		if err != nil {
			Error(w, fmt.Errorf("invalid token: %w", err), http.StatusUnauthorized)
			return
		}

		// Inject user info into context
		user := &database.User{
			ID:    claims.UserID,
			Email: claims.Email,
			Name:  claims.Name,
		}

		ctx := context.WithValue(r.Context(), userKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRepoOwnership middleware checks if the authenticated user owns the repository
func (h *Handler) RequireRepoOwnership(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		repoIDStr := chi.URLParam(r, "repoID")
		if repoIDStr == "" {
			// Fallback to "id" if "repoID" is not found
			repoIDStr = chi.URLParam(r, "id")
		}

		if repoIDStr == "" {
			// If no repo ID in path, potentially skip or error?
			// Assuming this middleware is only used on routes with :id or :repoID
			Error(w, fmt.Errorf("repository ID required"), http.StatusBadRequest)
			return
		}

		repoID, err := strconv.ParseInt(repoIDStr, 10, 64)
		if err != nil {
			Error(w, fmt.Errorf("invalid repository ID: %w", err), http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		user := GetUserFromContext(ctx)
		if user == nil {
			Error(w, fmt.Errorf("unauthorized"), http.StatusUnauthorized)
			return
		}

		repo, err := h.db.GetRepository(ctx, repoID)
		if err != nil {
			parsedErr := validation.ParseDatabaseError(err)
			// Return 404 to avoid leaking existence of repos not owned by user?
			// Or simple not found.
			Error(w, parsedErr, http.StatusNotFound)
			return
		}

		// Check ownership
		if repo.UserID == nil || *repo.UserID != user.ID {
			// Return 404 to assume privacy by default
			Error(w, fmt.Errorf("repository not found"), http.StatusNotFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}
