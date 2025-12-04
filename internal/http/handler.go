package http

import (
	"net/http"

	"git-repository-visualizer/internal/database"
	"git-repository-visualizer/internal/queue"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	router    *chi.Mux
	db        *database.DB
	publisher *queue.Publisher
}

func NewHandler(db *database.DB, publisher *queue.Publisher) *Handler {
	h := &Handler{
		router:    chi.NewRouter(),
		db:        db,
		publisher: publisher,
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
		// Repository management
		r.Post("/repositories", h.CreateRepository)
		r.Patch("/repositories/{id}", h.UpdateRepository)
		r.Get("/repositories", h.ListRepositories)
		r.Get("/repositories/{id}", h.GetRepository)
		r.Post("/repositories/{id}/index", h.IndexRepository)
		r.Post("/repositories/{id}/sync", h.SyncRepository)

		// Repository stats
		r.Route("/repositories/{id}/stats", func(r chi.Router) {
			r.Get("/contributors", h.ListContributors)
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
