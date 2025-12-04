package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
)

type Service interface {
	CloneRepository(ctx context.Context, repoPath string, localPath string) (*git.Repository, error)
}

func getService(repoPath string) Service {
	if strings.HasPrefix(repoPath, "https://github.com/") {
		return NewGitHub()
	}
	if strings.HasPrefix(repoPath, "https://bitbucket.org/") {
		return NewBitBucket()
	}
	return nil
}

func IndexRepository(ctx context.Context, repoPath string, localPath string) error {
	// Clone repository from remote and store to local path desgnated and process repo to database
	service := getService(repoPath)
	if service == nil {
		return fmt.Errorf("unsupported repository type")
	}
	repo, err := service.CloneRepository(ctx, repoPath, localPath)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// TODO: Process repo
	_ = repo

	return nil
}
