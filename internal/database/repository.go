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
		INSERT INTO repositories (url, status, default_branch, user_id, name, description, is_private, provider)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	defaultBranch := repo.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	err := db.pool.QueryRow(ctx, query,
		repo.URL, repo.Status, defaultBranch, repo.UserID,
		repo.Name, repo.Description, repo.IsPrivate, repo.Provider,
	).Scan(&repo.ID, &repo.CreatedAt, &repo.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	return nil
}

// UpsertRepository creates or updates a repository record
func (db *DB) UpsertRepository(ctx context.Context, repo *Repository) error {
	// Try to find existing repository by URL first
	// Note: We're using URL as the unique constraint here, which might be slightly risky if providers change URLs,
	// but it's the most common stable identifier for us currently.

	query := `
		INSERT INTO repositories (url, status, default_branch, user_id, name, description, is_private, provider, local_path, last_pushed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (url) DO UPDATE SET
			default_branch = EXCLUDED.default_branch,
			user_id = EXCLUDED.user_id,
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			is_private = EXCLUDED.is_private,
			provider = EXCLUDED.provider,
			last_pushed_at = EXCLUDED.last_pushed_at,
			updated_at = NOW()
		RETURNING id, status, local_path, last_indexed_at, created_at, updated_at
	`

	defaultBranch := repo.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	err := db.pool.QueryRow(ctx, query,
		repo.URL,
		repo.Status, // If new, use this status (likely 'discovered' or 'pending')
		defaultBranch,
		repo.UserID,
		repo.Name,
		repo.Description,
		repo.IsPrivate,
		repo.Provider,
		repo.LocalPath,
		repo.LastPushedAt,
	).Scan(
		&repo.ID, &repo.Status, &repo.LocalPath, &repo.LastIndexedAt,
		&repo.CreatedAt, &repo.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert repository: %w", err)
	}

	return nil
}

// GetRepository retrieves a repository by ID
func (db *DB) GetRepository(ctx context.Context, id int64) (*Repository, error) {
	query := `
		SELECT id, url, local_path, default_branch, status, last_indexed_at, created_at, updated_at, 
		       user_id, name, description, is_private, provider
		FROM repositories
		WHERE id = $1
	`

	repo := &Repository{}
	err := db.pool.QueryRow(ctx, query, id).Scan(
		&repo.ID, &repo.URL, &repo.LocalPath, &repo.DefaultBranch, &repo.Status, &repo.LastIndexedAt,
		&repo.CreatedAt, &repo.UpdatedAt, &repo.UserID, &repo.Name, &repo.Description, &repo.IsPrivate, &repo.Provider,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return repo, nil
}

// GetRepositoryForUser retrieves a repository by ID ensuring it belongs to the given user
func (db *DB) GetRepositoryForUser(ctx context.Context, id int64, userID int64) (*Repository, error) {
	query := `
		SELECT id, url, local_path, default_branch, status, last_indexed_at, created_at, updated_at, 
		       user_id, name, description, is_private, provider
		FROM repositories
		WHERE id = $1 AND user_id = $2
	`

	repo := &Repository{}
	err := db.pool.QueryRow(ctx, query, id, userID).Scan(
		&repo.ID, &repo.URL, &repo.LocalPath, &repo.DefaultBranch, &repo.Status, &repo.LastIndexedAt,
		&repo.CreatedAt, &repo.UpdatedAt, &repo.UserID, &repo.Name, &repo.Description, &repo.IsPrivate, &repo.Provider,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get repository for user: %w", err)
	}

	return repo, nil
}

// GetRepositoryStatus retrieves just the status of a repository for a user
func (db *DB) GetRepositoryStatus(ctx context.Context, id int64, userID int64) (RepositoryStatus, error) {
	query := `
		SELECT status
		FROM repositories
		WHERE id = $1 AND user_id = $2
	`

	var status RepositoryStatus
	err := db.pool.QueryRow(ctx, query, id, userID).Scan(&status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to get repository status: %w", err)
	}

	return status, nil
}

// GetRepositoryByURL retrieves a repository by its URL
func (db *DB) GetRepositoryByURL(ctx context.Context, url string) (*Repository, error) {
	query := `
		SELECT id, url, local_path, default_branch, status, last_indexed_at, created_at, updated_at,
		       user_id, name, description, is_private, provider
		FROM repositories
		WHERE url = $1
	`

	repo := &Repository{}
	err := db.pool.QueryRow(ctx, query, url).Scan(
		&repo.ID, &repo.URL, &repo.LocalPath, &repo.DefaultBranch, &repo.Status, &repo.LastIndexedAt,
		&repo.CreatedAt, &repo.UpdatedAt, &repo.UserID, &repo.Name, &repo.Description, &repo.IsPrivate, &repo.Provider,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return repo, nil
}

// ListRepositories retrieves all repositories for a user with pagination
func (db *DB) ListRepositories(ctx context.Context, userID int64, limit, offset int) ([]*Repository, error) {
	query := `
		SELECT id, url, local_path, default_branch, status, last_indexed_at, created_at, updated_at,
		       user_id, name, description, is_private, provider
		FROM repositories
		WHERE user_id = $1
		ORDER BY last_pushed_at DESC NULLS LAST, created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	defer rows.Close()

	repositories := []*Repository{}
	for rows.Next() {
		repo := &Repository{}
		err := rows.Scan(
			&repo.ID, &repo.URL, &repo.LocalPath, &repo.DefaultBranch, &repo.Status, &repo.LastIndexedAt,
			&repo.CreatedAt, &repo.UpdatedAt, &repo.UserID, &repo.Name, &repo.Description, &repo.IsPrivate, &repo.Provider,
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
		SET local_path = $1, status = $2, last_indexed_at = $3, default_branch = $4,
		    name = $5, description = $6, is_private = $7, provider = $8, user_id = $9
		WHERE id = $10
		RETURNING updated_at
	`

	err := db.pool.QueryRow(ctx, query,
		repo.LocalPath, repo.Status, repo.LastIndexedAt, repo.DefaultBranch,
		repo.Name, repo.Description, repo.IsPrivate, repo.Provider, repo.UserID,
		repo.ID,
	).Scan(&repo.UpdatedAt)
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
