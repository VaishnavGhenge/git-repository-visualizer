package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"git-repository-visualizer/internal/database"
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
