package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// UpsertCommitFiles batch inserts commit file stats (Granular History)
func (db *DB) UpsertCommitFiles(ctx context.Context, commitFiles []*CommitFile) error {
	if len(commitFiles) == 0 {
		return nil
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Since CommitFiles are immutable events, we just INSERT.
	// If re-indexing, we might want to delete old ones first, but for now we assume clean run or handled by repository deletion.
	// We'll use ON CONFLICT DO NOTHING just in case.
	// We need a unique constraint on (commit_hash, file_path) or (repository_id, commit_hash, file_path)

	query := `
		INSERT INTO commit_files (repository_id, commit_hash, file_path, additions, deletions)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (commit_hash, file_path) DO NOTHING
	`

	batch := &pgx.Batch{}
	for _, cf := range commitFiles {
		batch.Queue(query, cf.RepositoryID, cf.CommitHash, cf.FilePath, cf.Additions, cf.Deletions)
	}

	br := tx.SendBatch(ctx, batch)

	for range commitFiles {
		_, err := br.Exec()
		if err != nil {
			br.Close()
			return fmt.Errorf("failed to execute batch: %w", err)
		}
	}

	if err := br.Close(); err != nil {
		return fmt.Errorf("failed to close batch: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteCommitFilesByRepository deletes all commit files for a repository
func (db *DB) DeleteCommitFilesByRepository(ctx context.Context, repositoryID int64) error {
	query := `DELETE FROM commit_files WHERE repository_id = $1`

	_, err := db.pool.Exec(ctx, query, repositoryID)
	if err != nil {
		return fmt.Errorf("failed to delete commit files: %w", err)
	}

	return nil
}
