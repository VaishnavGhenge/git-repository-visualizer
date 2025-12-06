package git

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/go-git/go-git/v5"
)

type GitHub struct{}

func NewGitHub() *GitHub {
	return &GitHub{}
}

func (g *GitHub) CloneRepository(ctx context.Context, repoPath string, localPath string) (*git.Repository, error) {
	// Use bare clone (isBare: true) to only clone .git directory without working tree
	// This saves disk space and is faster since we only need git history for analysis
	r, err := git.PlainClone(localPath, true, &git.CloneOptions{
		URL:      repoPath,
		Progress: os.Stdout,
	})
	if err != nil {
		// If repository already exists, open and fetch updates
		if errors.Is(err, git.ErrRepositoryAlreadyExists) {
			r, err = git.PlainOpen(localPath)
			if err != nil {
				return nil, err
			}
			log.Printf("Repository exists, fetching updates...")
			err = r.Fetch(&git.FetchOptions{
				Progress: os.Stdout,
			})
			// "already up-to-date" is not an error for our use case
			if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
				return nil, err
			}
			return r, nil
		}
		return nil, err
	}
	return r, nil
}
