package http

import (
	"fmt"
	"git-repository-visualizer/internal/git"
	"net/http"
)

func (h *Handler) ListContributors(w http.ResponseWriter, r *http.Request) {
	repoPath := r.URL.Query().Get("repoPath")

	if repoPath == "" {
		Error(w, fmt.Errorf("repoPath query parameter is required"), http.StatusBadRequest)
		return
	}

	contributorList, err := git.GetContributors(repoPath)
	if err != nil {
		Error(w, err, http.StatusInternalServerError)
		return
	}

	JSON(w, http.StatusOK, map[string][]*git.Contributor{
		"contributors": contributorList,
	})
}
