package http

import (
	"fmt"
	"net/http"
)

func (h *Handler) ListContributors(w http.ResponseWriter, r *http.Request) {
	repoPath := r.URL.Query().Get("repoPath")

	if repoPath == "" {
		Error(w, fmt.Errorf("repoPath query parameter is required"), http.StatusBadRequest)
		return
	}

	contributorList := []interface{}{}

	JSON(w, http.StatusOK, map[string][]interface{}{
		"contributors": contributorList,
	})
}
