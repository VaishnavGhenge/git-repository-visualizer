-- Create users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    avatar_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
-- Create user_identities table to support multiple OAuth providers per user
CREATE TABLE user_identities (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    -- 'google', 'github', 'gitlab', 'bitbucket'
    provider_user_id TEXT NOT NULL,
    access_token TEXT,
    refresh_token TEXT,
    token_expiry TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(provider, provider_user_id)
);
-- Index for efficient lookup during login
CREATE INDEX idx_user_identities_user_id ON user_identities(user_id);
-- Link repositories to users (can be NULL for public or unassigned repos)
ALTER TABLE repositories
ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE
SET NULL;
-- Trigger to update updated_at for users
CREATE TRIGGER update_users_updated_at BEFORE
UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();