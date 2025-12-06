package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// UpsertFileStat inserts or updates a single file stat
func (db *DB) UpsertFileStat(ctx context.Context, stat *FileStat) error {
	query := `
		INSERT INTO file_stats (repository_id, file_path, total_changes, lines_added, lines_deleted, last_modified_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (repository_id, file_path)
		DO UPDATE SET
			total_changes = file_stats.total_changes + EXCLUDED.total_changes,
			lines_added = file_stats.lines_added + EXCLUDED.lines_added,
			lines_deleted = file_stats.lines_deleted + EXCLUDED.lines_deleted,
			last_modified_at = GREATEST(file_stats.last_modified_at, EXCLUDED.last_modified_at)
		RETURNING id, created_at, updated_at
	`

	err := db.pool.QueryRow(ctx, query,
		stat.RepositoryID,
		stat.FilePath,
		stat.TotalChanges,
		stat.LinesAdded,
		stat.LinesDeleted,
		stat.LastModifiedAt,
	).Scan(&stat.ID, &stat.CreatedAt, &stat.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to upsert file stat: %w", err)
	}

	return nil
}

// UpsertFileStats batch inserts or updates multiple file stats
func (db *DB) UpsertFileStats(ctx context.Context, stats []*FileStat) error {
	if len(stats) == 0 {
		return nil
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO file_stats (repository_id, file_path, total_changes, lines_added, lines_deleted, last_modified_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (repository_id, file_path)
		DO UPDATE SET
			total_changes = file_stats.total_changes + EXCLUDED.total_changes,
			lines_added = file_stats.lines_added + EXCLUDED.lines_added,
			lines_deleted = file_stats.lines_deleted + EXCLUDED.lines_deleted,
			last_modified_at = GREATEST(file_stats.last_modified_at, EXCLUDED.last_modified_at)
	`

	batch := &pgx.Batch{}
	for _, s := range stats {
		batch.Queue(query, s.RepositoryID, s.FilePath, s.TotalChanges, s.LinesAdded, s.LinesDeleted, s.LastModifiedAt)
	}

	br := tx.SendBatch(ctx, batch)

	for range stats {
		_, err := br.Exec()
		if err != nil {
			br.Close()
			return fmt.Errorf("failed to execute batch: %w", err)
		}
	}

	// Must close batch reader before committing transaction
	if err := br.Close(); err != nil {
		return fmt.Errorf("failed to close batch: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteFileStatsByRepository deletes all file stats for a repository
func (db *DB) DeleteFileStatsByRepository(ctx context.Context, repositoryID int64) error {
	query := `DELETE FROM file_stats WHERE repository_id = $1`

	_, err := db.pool.Exec(ctx, query, repositoryID)
	if err != nil {
		return fmt.Errorf("failed to delete file stats: %w", err)
	}

	return nil
}

// GetFileStatsByRepository retrieves file stats for a repository with pagination
func (db *DB) GetFileStatsByRepository(ctx context.Context, repositoryID int64, limit, offset int) ([]*FileStat, error) {
	query := `
		SELECT id, repository_id, file_path, total_changes, lines_added, lines_deleted, last_modified_at, created_at, updated_at
		FROM file_stats
		WHERE repository_id = $1
		ORDER BY total_changes DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.pool.Query(ctx, query, repositoryID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get file stats: %w", err)
	}
	defer rows.Close()

	stats := []*FileStat{}
	for rows.Next() {
		s := &FileStat{}
		err := rows.Scan(
			&s.ID,
			&s.RepositoryID,
			&s.FilePath,
			&s.TotalChanges,
			&s.LinesAdded,
			&s.LinesDeleted,
			&s.LastModifiedAt,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file stat: %w", err)
		}
		stats = append(stats, s)
	}

	return stats, nil
}
