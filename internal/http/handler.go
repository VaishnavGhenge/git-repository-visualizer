package http

import (
	"fmt"
	"net/http"
	"strconv"

	"git-repository-visualizer/internal/auth"
	"git-repository-visualizer/internal/config"
	"git-repository-visualizer/internal/database"
	"git-repository-visualizer/internal/queue"
	"git-repository-visualizer/internal/validation"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	router       *chi.Mux
	db           *database.DB
	publisher    queue.IPublisher
	httpCfg      config.HTTPConfig
	authRegistry *auth.Registry
	jwtManager   *auth.JWTManager
}

func NewHandler(db *database.DB, publisher queue.IPublisher, cfg *config.Config) *Handler {
	registry := auth.NewRegistry()
	registry.InitializeProviders(cfg.Auth)

	h := &Handler{
		router:       chi.NewRouter(),
		db:           db,
		publisher:    publisher,
		httpCfg:      cfg.HTTP,
		authRegistry: registry,
		jwtManager:   auth.NewJWTManager(cfg.Auth.JWTSecret),
	}

	// Apply global middleware
	h.router.Use(CORS)
	h.router.Use(Logger)

	h.registerRoutes()
	return h
}

func (h *Handler) registerRoutes() {
	// Health check
	h.router.Get("/ping", h.Ping)

	// API routes - grouped under /api/v1
	h.router.Route("/api/v1", func(r chi.Router) {
		// Auth routes
		r.Route("/auth/{provider}", func(r chi.Router) {
			r.Get("/login", h.AuthLogin)
			r.Get("/callback", h.AuthCallback)
		})
		// Repository management
		// Repository management (Protected)
		r.Group(func(r chi.Router) {
			r.Use(h.AuthMiddleware)

			// Provider specific
			r.Get("/providers/{provider}/repositories", h.GetProviderRepositories)

			r.Post("/repositories", h.CreateRepository)
			r.Post("/repositories/sync", h.SyncUserRepositories)
			r.Get("/repositories", h.ListRepositories)

			// Routes requiring ownership
			r.Group(func(r chi.Router) {
				// Internal handlers now check ownership strictly using GetRepositoryForUser
				r.Patch("/repositories/{id}", h.UpdateRepository)
				r.Get("/repositories/{id}", h.GetRepository)
				r.Get("/repositories/{id}/status", h.GetRepositoryStatus)
				r.Post("/repositories/{id}/index", h.IndexRepository)
				r.Post("/repositories/{id}/sync", h.SyncRepository)

				// Repository stats
				r.Route("/repositories/{repoID}/stats", func(r chi.Router) {
					// We still rely on middleware here as stats handlers are not yet updated
					r.Use(h.RequireRepoOwnership)
					r.Use(h.ValidateRepositoryStatus)
					r.Get("/contributors", h.ListContributors)
					r.Get("/files", h.ListFiles)
					r.Get("/bus-factor", h.GetBusFactor)
					r.Get("/churn", h.GetChurnStats)
					r.Get("/commit-activity", h.GetCommitActivity)
				})
			})
		})

		// Queue management
		r.Get("/queue/length", h.GetQueueLength)
	})
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]string{
		"message": "pong",
	})
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

// ValidateRepositoryStatus middleware ensures that statistics endpoints only serve data
// for repositories that have been fully indexed.
func (h *Handler) ValidateRepositoryStatus(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		repoIDStr := chi.URLParam(r, "repoID")
		if repoIDStr == "" {
			// Fallback to "id" if "repoID" is not found
			repoIDStr = chi.URLParam(r, "id")
		}

		if repoIDStr == "" {
			next.ServeHTTP(w, r)
			return
		}

		repoID, err := strconv.ParseInt(repoIDStr, 10, 64)
		if err != nil {
			Error(w, fmt.Errorf("invalid repository ID: %w", err), http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		repo, err := h.db.GetRepository(ctx, repoID)
		if err != nil {
			parsedErr := validation.ParseDatabaseError(err)
			Error(w, parsedErr, http.StatusNotFound)
			return
		}

		if repo.Status != database.StatusCompleted {
			Error(w, fmt.Errorf("repository indexing is not completed (current status: %s). please wait for indexing to finish", repo.Status), http.StatusConflict)
			return
		}

		next.ServeHTTP(w, r)
	})
}
