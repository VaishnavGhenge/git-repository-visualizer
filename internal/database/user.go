package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GetUserByEmail retrieves a user by their email address
func (db *DB) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id, email, name, avatar_url, created_at, updated_at FROM users WHERE email = $1`
	user := &User{}
	err := db.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// GetUserByID retrieves a user by their ID
func (db *DB) GetUserByID(ctx context.Context, id int64) (*User, error) {
	query := `SELECT id, email, name, avatar_url, created_at, updated_at FROM users WHERE id = $1`
	user := &User{}
	err := db.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// UpsertUserByIdentity handles user creation/update via an OAuth identity
func (db *DB) UpsertUserByIdentity(ctx context.Context, user *User, identity *UserIdentity) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Check if identity exists
	idQuery := `SELECT user_id FROM user_identities WHERE provider = $1 AND provider_user_id = $2`
	var existingUserID int64
	err = tx.QueryRow(ctx, idQuery, identity.Provider, identity.ProviderUserID).Scan(&existingUserID)

	if err == nil {
		// Identity exists, update tokens and fetch user
		identity.UserID = existingUserID
		updateIdQuery := `
			UPDATE user_identities 
			SET access_token = $1, refresh_token = $2, token_expiry = $3, provider_username = $4
			WHERE provider = $5 AND provider_user_id = $6
		`
		_, err = tx.Exec(ctx, updateIdQuery, identity.AccessToken, identity.RefreshToken, identity.TokenExpiry, identity.ProviderUsername, identity.Provider, identity.ProviderUserID)
		if err != nil {
			return fmt.Errorf("failed to update identity: %w", err)
		}

		userQuery := `SELECT id, email, name, avatar_url, created_at, updated_at FROM users WHERE id = $1`
		err = tx.QueryRow(ctx, userQuery, existingUserID).Scan(
			&user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to get existing user: %w", err)
		}
	} else if err == pgx.ErrNoRows {
		// Identity doesn't exist, check if user with email exists
		userQuery := `SELECT id, email, name, avatar_url, created_at, updated_at FROM users WHERE email = $1`
		err = tx.QueryRow(ctx, userQuery, user.Email).Scan(
			&user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt,
		)

		if err == pgx.ErrNoRows {
			// New user
			insertUserQuery := `
				INSERT INTO users (email, name, avatar_url) 
				VALUES ($1, $2, $3) 
				RETURNING id, created_at, updated_at
			`
			err = tx.QueryRow(ctx, insertUserQuery, user.Email, user.Name, user.AvatarURL).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
			if err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to check existing email: %w", err)
		}

		// Create identity for the user
		insertIdQuery := `
			INSERT INTO user_identities (user_id, provider, provider_user_id, access_token, refresh_token, token_expiry, provider_username)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`
		_, err = tx.Exec(ctx, insertIdQuery, user.ID, identity.Provider, identity.ProviderUserID, identity.AccessToken, identity.RefreshToken, identity.TokenExpiry, identity.ProviderUsername)
		if err != nil {
			return fmt.Errorf("failed to create identity: %w", err)
		}
	} else {
		return fmt.Errorf("failed to check identity: %w", err)
	}

	return tx.Commit(ctx)
}

// GetUserIdentity retrieves an identity for a user and provider
func (db *DB) GetUserIdentity(ctx context.Context, userID int64, provider string) (*UserIdentity, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, access_token, refresh_token, token_expiry, created_at, provider_username
		FROM user_identities
		WHERE user_id = $1 AND provider = $2
	`
	identity := &UserIdentity{}
	err := db.pool.QueryRow(ctx, query, userID, provider).Scan(
		&identity.ID, &identity.UserID, &identity.Provider, &identity.ProviderUserID,
		&identity.AccessToken, &identity.RefreshToken, &identity.TokenExpiry, &identity.CreatedAt,
		&identity.ProviderUsername,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get identity: %w", err)
	}
	return identity, nil
}

// GetUserIdentities retrieves all identities for a user
func (db *DB) GetUserIdentities(ctx context.Context, userID int64) ([]*UserIdentity, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, access_token, refresh_token, token_expiry, created_at, provider_username
		FROM user_identities
		WHERE user_id = $1
	`
	rows, err := db.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user identities: %w", err)
	}
	defer rows.Close()

	var identities []*UserIdentity
	for rows.Next() {
		identity := &UserIdentity{}
		err := rows.Scan(
			&identity.ID, &identity.UserID, &identity.Provider, &identity.ProviderUserID,
			&identity.AccessToken, &identity.RefreshToken, &identity.TokenExpiry, &identity.CreatedAt,
			&identity.ProviderUsername,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan identity: %w", err)
		}
		identities = append(identities, identity)
	}
	return identities, nil
}
