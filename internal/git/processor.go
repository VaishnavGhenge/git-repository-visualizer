package git

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"git-repository-visualizer/internal/database"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ProcessResult contains the results of processing a repository
type ProcessResult struct {
	CommitsProcessed   int
	ContributorsFound  int
	FilesTracked       int
	ProcessingDuration time.Duration
}

// ProcessRepository extracts commit data from a cloned repository and persists to database
func ProcessRepository(ctx context.Context, db *database.DB, repoID int64, repo *git.Repository) (*ProcessResult, error) {
	startTime := time.Now()

	// 1. Get HEAD reference
	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	headCommit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD commit: %w", err)
	}

	// 2. Snapshot Phase: Capture current file state (Inventory)
	filesTracked, err := processSnapshot(ctx, db, repoID, headCommit)
	if err != nil {
		return nil, fmt.Errorf("snapshot failed: %w", err)
	}

	// 3. History Phase: Walk Commits
	commitsProcessed, contributorsFound, err := processHistory(ctx, db, repoID, repo, ref)
	if err != nil {
		return nil, fmt.Errorf("history processing failed: %w", err)
	}

	return &ProcessResult{
		CommitsProcessed:   commitsProcessed,
		ContributorsFound:  contributorsFound,
		FilesTracked:       filesTracked,
		ProcessingDuration: time.Since(startTime),
	}, nil
}

// processSnapshot handles the inventory of files at HEAD
func processSnapshot(ctx context.Context, db *database.DB, repoID int64, headCommit *object.Commit) (int, error) {
	log.Printf("Snapshotting file inventory for repository %d...", repoID)

	files := []*database.File{}

	headTree, err := headCommit.Tree()
	if err != nil {
		return 0, fmt.Errorf("failed to get HEAD tree: %w", err)
	}

	err = headTree.Files().ForEach(func(f *object.File) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Simple language detection by extension
		parts := strings.Split(f.Name, ".")
		lang := "Plain Text"
		if len(parts) > 1 {
			lang = parts[len(parts)-1]
		}

		// Count lines
		lines := 0
		if !f.Mode.IsFile() {
			return nil
		}

		r, err := f.Reader()
		if err == nil {
			scanner := bufio.NewScanner(r)
			buf := make([]byte, ScannerInitialBufferSize)
			scanner.Buffer(buf, ScannerMaxBufferSize)
			for scanner.Scan() {
				lines++
			}
			r.Close()
		}

		files = append(files, &database.File{
			RepositoryID: repoID,
			Path:         f.Name,
			Language:     lang,
			Lines:        lines,
		})
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to walk file tree: %w", err)
	}

	// Persist Files (Clear old ones first)
	if err := db.DeleteFilesByRepository(ctx, repoID); err != nil {
		return 0, fmt.Errorf("failed to clear existing files: %w", err)
	}

	// Batch Insert
	for i := 0; i < len(files); i += FileBatchSize {
		end := i + FileBatchSize
		if end > len(files) {
			end = len(files)
		}
		if err := db.UpsertFiles(ctx, files[i:end]); err != nil {
			return 0, fmt.Errorf("failed to persist files: %w", err)
		}
	}
	log.Printf("Persisted %d files inventory", len(files))

	return len(files), nil
}

// processHistory handles walking the commit log and extracting granular events
func processHistory(ctx context.Context, db *database.DB, repoID int64, repo *git.Repository, ref *plumbing.Reference) (int, int, error) {
	// Clear old history first
	if err := db.DeleteContributorsByRepository(ctx, repoID); err != nil {
		return 0, 0, fmt.Errorf("failed to clear contributors: %w", err)
	}
	if err := db.DeleteCommitsByRepository(ctx, repoID); err != nil {
		return 0, 0, fmt.Errorf("failed to clear commits: %w", err)
	}
	if err := db.DeleteCommitFilesByRepository(ctx, repoID); err != nil {
		return 0, 0, fmt.Errorf("failed to clear commit files: %w", err)
	}

	commitIter, err := repo.Log(&git.LogOptions{
		From:  ref.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get commit log: %w", err)
	}
	defer commitIter.Close()

	// Temporary aggregators
	contributorMap := make(map[string]*database.Contributor)
	commitsBatch := []*database.Commit{}
	commitFilesBatch := []*database.CommitFile{}

	commitCount := 0

	err = commitIter.ForEach(func(c *object.Commit) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		commitCount++
		// Process individual commit
		processSingleCommit(repoID, c, contributorMap, &commitsBatch, &commitFilesBatch)

		// Batch Flushing
		if len(commitsBatch) >= CommitBatchSize {
			if err := flushBatches(ctx, db, commitsBatch, commitFilesBatch); err != nil {
				return err
			}
			// Reset slices (keeping capacity)
			commitsBatch = commitsBatch[:0]
			commitFilesBatch = commitFilesBatch[:0]
			log.Printf("Processed %d commits...", commitCount)
		}
		return nil
	})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to iterate commits: %w", err)
	}

	// Flush remaining
	if len(commitsBatch) > 0 {
		if err := flushBatches(ctx, db, commitsBatch, commitFilesBatch); err != nil {
			return 0, 0, err
		}
	}

	// Persist Contributors
	contributors := make([]*database.Contributor, 0, len(contributorMap))
	for _, c := range contributorMap {
		contributors = append(contributors, c)
	}

	for i := 0; i < len(contributors); i += ContributorBatchSize {
		end := i + ContributorBatchSize
		if end > len(contributors) {
			end = len(contributors)
		}
		if err := db.UpsertContributors(ctx, contributors[i:end]); err != nil {
			return 0, 0, fmt.Errorf("failed to persist contributors: %w", err)
		}
	}

	log.Printf("Finished history processing: %d commits, %d contributors", commitCount, len(contributors))
	return commitCount, len(contributors), nil
}

func processSingleCommit(repoID int64, c *object.Commit, contributorMap map[string]*database.Contributor, commitsBatch *[]*database.Commit, commitFilesBatch *[]*database.CommitFile) {
	commitTime := c.Author.When
	email := c.Author.Email

	// 1. Contributor Tracking
	contributor, exists := contributorMap[email]
	if !exists {
		contributor = &database.Contributor{
			RepositoryID:  repoID,
			Email:         email,
			Name:          c.Author.Name,
			FirstCommitAt: &commitTime,
			LastCommitAt:  &commitTime,
		}
		contributorMap[email] = contributor
	}
	if contributor.FirstCommitAt == nil || commitTime.Before(*contributor.FirstCommitAt) {
		contributor.FirstCommitAt = &commitTime
	}
	if contributor.LastCommitAt == nil || commitTime.After(*contributor.LastCommitAt) {
		contributor.LastCommitAt = &commitTime
	}

	// 2. Commit Record
	dbCommit := &database.Commit{
		RepositoryID: repoID,
		Hash:         c.Hash.String(),
		AuthorEmail:  email,
		AuthorName:   c.Author.Name,
		Message:      c.Message,
		CommittedAt:  commitTime,
	}
	*commitsBatch = append(*commitsBatch, dbCommit)

	// 3. Diff / CommitFiles
	// go-git Stats() builds patches and counts stats
	stats, err := c.Stats()
	if err == nil {
		for _, stat := range stats {
			cf := &database.CommitFile{
				RepositoryID: repoID,
				CommitHash:   c.Hash.String(),
				FilePath:     stat.Name,
				Additions:    stat.Addition,
				Deletions:    stat.Deletion,
			}
			*commitFilesBatch = append(*commitFilesBatch, cf)
		}
	}
}

func flushBatches(ctx context.Context, db *database.DB, commits []*database.Commit, commitFiles []*database.CommitFile) error {
	if err := db.UpsertCommits(ctx, commits); err != nil {
		return fmt.Errorf("failed to persist batch commits: %w", err)
	}
	if err := db.UpsertCommitFiles(ctx, commitFiles); err != nil {
		return fmt.Errorf("failed to persist batch commit files: %w", err)
	}
	return nil
}
