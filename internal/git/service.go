package git

import (
	"context"
	"fmt"
	"log"
	"strings"

	"git-repository-visualizer/internal/database"

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

// IndexRepository clones a repository and processes its commit history
func IndexRepository(ctx context.Context, db *database.DB, repoID int64, repoPath string, localPath string) error {
	// Clone repository from remote and store to local path designated
	service := getService(repoPath)
	if service == nil {
		return fmt.Errorf("unsupported repository type")
	}

	log.Printf("Cloning repository %s to %s", repoPath, localPath)
	repo, err := service.CloneRepository(ctx, repoPath, localPath)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	log.Printf("Processing repository commits...")
	result, err := ProcessRepository(ctx, db, repoID, repo)
	if err != nil {
		return fmt.Errorf("failed to process repository: %w", err)
	}

	log.Printf("Repository indexed: %d commits, %d contributors in %v",
		result.CommitsProcessed, result.ContributorsFound, result.ProcessingDuration)

	return nil
}
