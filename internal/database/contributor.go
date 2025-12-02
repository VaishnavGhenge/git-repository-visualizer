package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// UpsertContributor inserts or updates a contributor
func (db *DB) UpsertContributor(ctx context.Context, contributor *Contributor) error {
	query := `
		INSERT INTO contributors (repository_id, email, name, commit_count, lines_added, lines_deleted, first_commit_at, last_commit_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (repository_id, email)
		DO UPDATE SET
			name = EXCLUDED.name,
			commit_count = EXCLUDED.commit_count,
			lines_added = EXCLUDED.lines_added,
			lines_deleted = EXCLUDED.lines_deleted,
			first_commit_at = LEAST(contributors.first_commit_at, EXCLUDED.first_commit_at),
			last_commit_at = GREATEST(contributors.last_commit_at, EXCLUDED.last_commit_at)
		RETURNING id, created_at, updated_at
	`

	err := db.pool.QueryRow(ctx, query,
		contributor.RepositoryID,
		contributor.Email,
		contributor.Name,
		contributor.CommitCount,
		contributor.LinesAdded,
		contributor.LinesDeleted,
		contributor.FirstCommitAt,
		contributor.LastCommitAt,
	).Scan(&contributor.ID, &contributor.CreatedAt, &contributor.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to upsert contributor: %w", err)
	}

	return nil
}

// UpsertContributors batch inserts or updates multiple contributors
func (db *DB) UpsertContributors(ctx context.Context, contributors []*Contributor) error {
	if len(contributors) == 0 {
		return nil
	}

	// Use a transaction for batch operation
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO contributors (repository_id, email, name, commit_count, lines_added, lines_deleted, first_commit_at, last_commit_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (repository_id, email)
		DO UPDATE SET
			name = EXCLUDED.name,
			commit_count = contributors.commit_count + EXCLUDED.commit_count,
			lines_added = contributors.lines_added + EXCLUDED.lines_added,
			lines_deleted = contributors.lines_deleted + EXCLUDED.lines_deleted,
			first_commit_at = LEAST(contributors.first_commit_at, EXCLUDED.first_commit_at),
			last_commit_at = GREATEST(contributors.last_commit_at, EXCLUDED.last_commit_at)
	`

	batch := &pgx.Batch{}
	for _, c := range contributors {
		batch.Queue(query, c.RepositoryID, c.Email, c.Name, c.CommitCount, c.LinesAdded, c.LinesDeleted, c.FirstCommitAt, c.LastCommitAt)
	}

	br := tx.SendBatch(ctx, batch)
	defer br.Close()

	for range contributors {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to execute batch: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetContributorsByRepository retrieves all contributors for a repository
func (db *DB) GetContributorsByRepository(ctx context.Context, repositoryID int64, limit, offset int) ([]*Contributor, error) {
	query := `
		SELECT id, repository_id, email, name, commit_count, lines_added, lines_deleted, 
		       first_commit_at, last_commit_at, created_at, updated_at
		FROM contributors
		WHERE repository_id = $1
		ORDER BY commit_count DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.pool.Query(ctx, query, repositoryID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get contributors: %w", err)
	}
	defer rows.Close()

	contributors := []*Contributor{}
	for rows.Next() {
		c := &Contributor{}
		err := rows.Scan(
			&c.ID,
			&c.RepositoryID,
			&c.Email,
			&c.Name,
			&c.CommitCount,
			&c.LinesAdded,
			&c.LinesDeleted,
			&c.FirstCommitAt,
			&c.LastCommitAt,
			&c.CreatedAt,
			&c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan contributor: %w", err)
		}
		contributors = append(contributors, c)
	}

	return contributors, nil
}

// DeleteContributorsByRepository deletes all contributors for a repository
func (db *DB) DeleteContributorsByRepository(ctx context.Context, repositoryID int64) error {
	query := `DELETE FROM contributors WHERE repository_id = $1`

	_, err := db.pool.Exec(ctx, query, repositoryID)
	if err != nil {
		return fmt.Errorf("failed to delete contributors: %w", err)
	}

	return nil
}
