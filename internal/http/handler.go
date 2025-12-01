package http

import (
	"net/http"
)

type Handler struct {
	mux     *http.ServeMux
	handler http.Handler
}

func NewHandler() *Handler {
	h := &Handler{
		mux: http.NewServeMux(),
	}
	h.registerRoutes()
	h.handler = Logger(CORS(h.mux))
	return h
}

func (h *Handler) registerRoutes() {
	// Health check
	h.mux.HandleFunc("/ping", h.Ping)

	// API routes
	h.mux.HandleFunc("/api/v1/contributors", h.ListContributors)
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]string{
		"message": "pong",
	})
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}
