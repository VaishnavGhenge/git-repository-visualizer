package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// UpsertCommit inserts or updates a single commit
func (db *DB) UpsertCommit(ctx context.Context, commit *Commit) error {
	query := `
		INSERT INTO commits (repository_id, hash, author_email, author_name, message, committed_at, additions, deletions)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (repository_id, hash)
		DO UPDATE SET
			author_email = EXCLUDED.author_email,
			author_name = EXCLUDED.author_name,
			message = EXCLUDED.message,
			additions = EXCLUDED.additions,
			deletions = EXCLUDED.deletions
		RETURNING id, created_at
	`

	err := db.pool.QueryRow(ctx, query,
		commit.RepositoryID,
		commit.Hash,
		commit.AuthorEmail,
		commit.AuthorName,
		commit.Message,
		commit.CommittedAt,
	).Scan(&commit.ID, &commit.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to upsert commit: %w", err)
	}

	return nil
}

// UpsertCommits batch inserts or updates multiple commits
func (db *DB) UpsertCommits(ctx context.Context, commits []*Commit) error {
	if len(commits) == 0 {
		return nil
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO commits (repository_id, hash, author_email, author_name, message, committed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (repository_id, hash)
		DO UPDATE SET
			author_email = EXCLUDED.author_email,
			author_name = EXCLUDED.author_name,
			message = EXCLUDED.message
	`

	batch := &pgx.Batch{}
	for _, c := range commits {
		batch.Queue(query, c.RepositoryID, c.Hash, c.AuthorEmail, c.AuthorName, c.Message, c.CommittedAt)
	}

	br := tx.SendBatch(ctx, batch)

	for range commits {
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

// DeleteCommitsByRepository deletes all commits for a repository
func (db *DB) DeleteCommitsByRepository(ctx context.Context, repositoryID int64) error {
	query := `DELETE FROM commits WHERE repository_id = $1`

	_, err := db.pool.Exec(ctx, query, repositoryID)
	if err != nil {
		return fmt.Errorf("failed to delete commits: %w", err)
	}

	return nil
}

// GetCommitsByRepository retrieves commits for a repository with pagination
func (db *DB) GetCommitsByRepository(ctx context.Context, repositoryID int64, limit, offset int) ([]*Commit, error) {
	query := `
		SELECT id, repository_id, hash, author_email, author_name, message, committed_at, created_at
		FROM commits
		WHERE repository_id = $1
		ORDER BY committed_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.pool.Query(ctx, query, repositoryID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}
	defer rows.Close()

	commits := []*Commit{}
	for rows.Next() {
		c := &Commit{}
		err := rows.Scan(
			&c.ID,
			&c.RepositoryID,
			&c.Hash,
			&c.AuthorEmail,
			&c.AuthorName,
			&c.Message,
			&c.CommittedAt,
			&c.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan commit: %w", err)
		}
		commits = append(commits, c)
	}

	return commits, nil
}
