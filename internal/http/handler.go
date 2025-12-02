package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	router *chi.Mux
}

func NewHandler() *Handler {
	h := &Handler{
		router: chi.NewRouter(),
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
	h.router.Route("/repositories/stats", func(r chi.Router) {
		r.Get("/contributors", h.ListContributors)
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
