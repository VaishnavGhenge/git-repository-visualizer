package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// UpsertFiles batch inserts or updates multiple files (Inventory)
func (db *DB) UpsertFiles(ctx context.Context, files []*File) error {
	if len(files) == 0 {
		return nil
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// We use Path + RepositoryID as unique constraint
	query := `
		INSERT INTO files (repository_id, path, language, lines, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (repository_id, path)
		DO UPDATE SET
			lines = EXCLUDED.lines,
			updated_at = NOW()
	`

	batch := &pgx.Batch{}
	for _, f := range files {
		batch.Queue(query, f.RepositoryID, f.Path, f.Language, f.Lines)
	}

	br := tx.SendBatch(ctx, batch)

	for range files {
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

// DeleteFilesByRepository deletes all files for a repository
func (db *DB) DeleteFilesByRepository(ctx context.Context, repositoryID int64) error {
	query := `DELETE FROM files WHERE repository_id = $1`

	_, err := db.pool.Exec(ctx, query, repositoryID)
	if err != nil {
		return fmt.Errorf("failed to delete files: %w", err)
	}

	return nil
}

// GetFilesByRepository retrieves files with pagination
func (db *DB) GetFilesByRepository(ctx context.Context, repositoryID int64, limit, offset int) ([]*File, error) {
	query := `
		SELECT id, repository_id, path, language, lines, created_at, updated_at
		FROM files
		WHERE repository_id = $1
		ORDER BY lines DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.pool.Query(ctx, query, repositoryID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get files: %w", err)
	}
	defer rows.Close()

	files := []*File{}
	for rows.Next() {
		f := &File{}
		err := rows.Scan(
			&f.ID,
			&f.RepositoryID,
			&f.Path,
			&f.Language,
			&f.Lines,
			&f.CreatedAt,
			&f.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		files = append(files, f)
	}

	return files, nil
}
