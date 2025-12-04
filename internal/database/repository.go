package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// CreateRepository creates a new repository record
func (db *DB) CreateRepository(ctx context.Context, repo *Repository) error {
	query := `
		INSERT INTO repositories (url, status, default_branch)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	defaultBranch := repo.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	err := db.pool.QueryRow(ctx, query, repo.URL, repo.Status, defaultBranch).
		Scan(&repo.ID, &repo.CreatedAt, &repo.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	return nil
}

// GetRepository retrieves a repository by ID
func (db *DB) GetRepository(ctx context.Context, id int64) (*Repository, error) {
	query := `
		SELECT id, url, local_path, default_branch, status, last_indexed_at, created_at, updated_at
		FROM repositories
		WHERE id = $1
	`

	repo := &Repository{}
	err := db.pool.QueryRow(ctx, query, id).Scan(
		&repo.ID,
		&repo.URL,
		&repo.LocalPath,
		&repo.DefaultBranch,
		&repo.Status,
		&repo.LastIndexedAt,
		&repo.CreatedAt,
		&repo.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("repository not found")
		}
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return repo, nil
}

// GetRepositoryByURL retrieves a repository by its URL
func (db *DB) GetRepositoryByURL(ctx context.Context, url string) (*Repository, error) {
	query := `
		SELECT id, url, local_path, default_branch, status, last_indexed_at, created_at, updated_at
		FROM repositories
		WHERE url = $1
	`

	repo := &Repository{}
	err := db.pool.QueryRow(ctx, query, url).Scan(
		&repo.ID,
		&repo.URL,
		&repo.LocalPath,
		&repo.DefaultBranch,
		&repo.Status,
		&repo.LastIndexedAt,
		&repo.CreatedAt,
		&repo.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("repository not found")
		}
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return repo, nil
}

// ListRepositories retrieves all repositories with pagination
func (db *DB) ListRepositories(ctx context.Context, limit, offset int) ([]*Repository, error) {
	query := `
		SELECT id, url, local_path, default_branch, status, last_indexed_at, created_at, updated_at
		FROM repositories
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := db.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	defer rows.Close()

	repositories := []*Repository{}
	for rows.Next() {
		repo := &Repository{}
		err := rows.Scan(
			&repo.ID,
			&repo.URL,
			&repo.LocalPath,
			&repo.DefaultBranch,
			&repo.Status,
			&repo.LastIndexedAt,
			&repo.CreatedAt,
			&repo.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan repository: %w", err)
		}
		repositories = append(repositories, repo)
	}

	return repositories, nil
}

// UpdateRepositoryStatus updates the status of a repository
func (db *DB) UpdateRepositoryStatus(ctx context.Context, id int64, status RepositoryStatus) error {
	query := `
		UPDATE repositories
		SET status = $1
		WHERE id = $2
	`

	result, err := db.pool.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update repository status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("repository not found")
	}

	return nil
}

// UpdateRepository updates repository fields
func (db *DB) UpdateRepository(ctx context.Context, repo *Repository) error {
	query := `
		UPDATE repositories
		SET local_path = $1, status = $2, last_indexed_at = $3, default_branch = $4
		WHERE id = $5
		RETURNING updated_at
	`

	err := db.pool.QueryRow(ctx, query, repo.LocalPath, repo.Status, repo.LastIndexedAt, repo.DefaultBranch, repo.ID).
		Scan(&repo.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	return nil
}

// GetRepositoriesByStatus retrieves repositories filtered by status
func (db *DB) GetRepositoriesByStatus(ctx context.Context, status RepositoryStatus) ([]*Repository, error) {
	query := `
		SELECT id, url, local_path, status, last_indexed_at, created_at, updated_at
		FROM repositories
		WHERE status = $1
		ORDER BY created_at ASC
	`

	rows, err := db.pool.Query(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories by status: %w", err)
	}
	defer rows.Close()

	repositories := []*Repository{}
	for rows.Next() {
		repo := &Repository{}
		err := rows.Scan(
			&repo.ID,
			&repo.URL,
			&repo.LocalPath,
			&repo.DefaultBranch,
			&repo.Status,
			&repo.LastIndexedAt,
			&repo.CreatedAt,
			&repo.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan repository: %w", err)
		}
		repositories = append(repositories, repo)
	}

	return repositories, nil
}

// GetStaleRepositories retrieves repositories that need re-indexing
func (db *DB) GetStaleRepositories(ctx context.Context, olderThan time.Duration) ([]*Repository, error) {
	threshold := time.Now().Add(-olderThan)
	query := `
		SELECT id, url, local_path, status, last_indexed_at, created_at, updated_at
		FROM repositories
		WHERE status = $1 
		AND (last_indexed_at IS NULL OR last_indexed_at < $2)
		ORDER BY last_indexed_at ASC NULLS FIRST
	`

	rows, err := db.pool.Query(ctx, query, StatusCompleted, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to get stale repositories: %w", err)
	}
	defer rows.Close()

	repositories := []*Repository{}
	for rows.Next() {
		repo := &Repository{}
		err := rows.Scan(
			&repo.ID,
			&repo.URL,
			&repo.LocalPath,
			&repo.DefaultBranch,
			&repo.Status,
			&repo.LastIndexedAt,
			&repo.CreatedAt,
			&repo.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan repository: %w", err)
		}
		repositories = append(repositories, repo)
	}

	return repositories, nil
}
