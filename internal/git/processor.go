package git

import (
	"context"
	"fmt"
	"log"
	"time"

	"git-repository-visualizer/internal/database"

	"github.com/go-git/go-git/v5"
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

	// Get the default branch reference
	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	// Get commit iterator starting from HEAD
	commitIter, err := repo.Log(&git.LogOptions{
		From:  ref.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get commit log: %w", err)
	}
	defer commitIter.Close()

	// Aggregate data
	contributorMap := make(map[string]*database.Contributor)
	fileStatMap := make(map[string]*database.FileStat)
	commits := []*database.Commit{}
	commitCount := 0

	err = commitIter.ForEach(func(c *object.Commit) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		commitCount++
		commitTime := c.Author.When
		email := c.Author.Email

		// === Aggregate contributor stats ===
		contributor, exists := contributorMap[email]
		if !exists {
			contributor = &database.Contributor{
				RepositoryID:  repoID,
				Email:         email,
				Name:          c.Author.Name,
				CommitCount:   0,
				LinesAdded:    0,
				LinesDeleted:  0,
				FirstCommitAt: &commitTime,
				LastCommitAt:  &commitTime,
			}
			contributorMap[email] = contributor
		}
		contributor.CommitCount++

		// Update first/last commit timestamps
		if contributor.FirstCommitAt == nil || commitTime.Before(*contributor.FirstCommitAt) {
			contributor.FirstCommitAt = &commitTime
		}
		if contributor.LastCommitAt == nil || commitTime.After(*contributor.LastCommitAt) {
			contributor.LastCommitAt = &commitTime
		}

		// === Get commit stats (diff with parent) ===
		var additions, deletions int
		stats, err := c.Stats()
		if err == nil {
			for _, stat := range stats {
				additions += stat.Addition
				deletions += stat.Deletion

				// Update contributor line stats
				contributor.LinesAdded += stat.Addition
				contributor.LinesDeleted += stat.Deletion

				// === Aggregate file stats ===
				fileStat, exists := fileStatMap[stat.Name]
				if !exists {
					fileStat = &database.FileStat{
						RepositoryID:   repoID,
						FilePath:       stat.Name,
						TotalChanges:   0,
						LinesAdded:     0,
						LinesDeleted:   0,
						LastModifiedAt: &commitTime,
					}
					fileStatMap[stat.Name] = fileStat
				}
				fileStat.TotalChanges++
				fileStat.LinesAdded += stat.Addition
				fileStat.LinesDeleted += stat.Deletion
				if fileStat.LastModifiedAt == nil || commitTime.After(*fileStat.LastModifiedAt) {
					fileStat.LastModifiedAt = &commitTime
				}
			}
		}

		// === Build commit record ===
		commits = append(commits, &database.Commit{
			RepositoryID: repoID,
			Hash:         c.Hash.String(),
			AuthorEmail:  email,
			AuthorName:   c.Author.Name,
			Message:      c.Message,
			CommittedAt:  commitTime,
			Additions:    additions,
			Deletions:    deletions,
		})

		// Log progress every 100 commits (slower now due to diff calculation)
		if commitCount%100 == 0 {
			log.Printf("Processed %d commits for repository %d", commitCount, repoID)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	log.Printf("Finished processing %d commits, found %d contributors, %d files",
		commitCount, len(contributorMap), len(fileStatMap))

	// === Persist to database ===

	// Clear existing data before upserting (for re-indexing)
	if err := db.DeleteContributorsByRepository(ctx, repoID); err != nil {
		return nil, fmt.Errorf("failed to clear existing contributors: %w", err)
	}
	if err := db.DeleteCommitsByRepository(ctx, repoID); err != nil {
		return nil, fmt.Errorf("failed to clear existing commits: %w", err)
	}
	if err := db.DeleteFileStatsByRepository(ctx, repoID); err != nil {
		return nil, fmt.Errorf("failed to clear existing file stats: %w", err)
	}

	// Convert maps to slices
	contributors := make([]*database.Contributor, 0, len(contributorMap))
	for _, c := range contributorMap {
		contributors = append(contributors, c)
	}
	fileStats := make([]*database.FileStat, 0, len(fileStatMap))
	for _, f := range fileStatMap {
		fileStats = append(fileStats, f)
	}

	// Persist contributors
	if err := db.UpsertContributors(ctx, contributors); err != nil {
		return nil, fmt.Errorf("failed to persist contributors: %w", err)
	}
	log.Printf("Persisted %d contributors for repository %d", len(contributors), repoID)

	// Persist commits in batches of 500
	batchSize := 500
	for i := 0; i < len(commits); i += batchSize {
		end := i + batchSize
		if end > len(commits) {
			end = len(commits)
		}
		if err := db.UpsertCommits(ctx, commits[i:end]); err != nil {
			return nil, fmt.Errorf("failed to persist commits: %w", err)
		}
	}
	log.Printf("Persisted %d commits for repository %d", len(commits), repoID)

	// Persist file stats in batches of 500
	for i := 0; i < len(fileStats); i += batchSize {
		end := i + batchSize
		if end > len(fileStats) {
			end = len(fileStats)
		}
		if err := db.UpsertFileStats(ctx, fileStats[i:end]); err != nil {
			return nil, fmt.Errorf("failed to persist file stats: %w", err)
		}
	}
	log.Printf("Persisted %d file stats for repository %d", len(fileStats), repoID)

	return &ProcessResult{
		CommitsProcessed:   commitCount,
		ContributorsFound:  len(contributors),
		FilesTracked:       len(fileStats),
		ProcessingDuration: time.Since(startTime),
	}, nil
}
