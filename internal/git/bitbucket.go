package git

import (
	"context"
	"os"

	"github.com/go-git/go-git/v5"
)

type BitBucket struct{}

func NewBitBucket() *BitBucket {
	return &BitBucket{}
}

func (b *BitBucket) CloneRepository(ctx context.Context, repoPath string, localPath string) (*git.Repository, error) {
	// Use bare clone (isBare: true) to only clone .git directory without working tree
	// This saves disk space and is faster since we only need git history for analysis
	r, err := git.PlainClone(localPath, true, &git.CloneOptions{
		URL:      repoPath,
		Progress: os.Stdout,
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}
