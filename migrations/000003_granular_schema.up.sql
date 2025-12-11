-- Drop old table
DROP TABLE IF EXISTS file_stats;
-- Create new files table (Inventory)
CREATE TABLE files (
    id BIGSERIAL PRIMARY KEY,
    repository_id BIGINT NOT NULL,
    path TEXT NOT NULL,
    language TEXT NOT NULL,
    lines INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(repository_id, path)
);
-- Optimization Indexes for Files
CREATE INDEX idx_files_repository_id ON files(repository_id);
CREATE INDEX idx_files_path ON files(path);
-- Create commit_files table (Granular History)
CREATE TABLE commit_files (
    id BIGSERIAL PRIMARY KEY,
    repository_id BIGINT NOT NULL,
    commit_hash TEXT NOT NULL,
    file_path TEXT NOT NULL,
    additions INTEGER NOT NULL DEFAULT 0,
    deletions INTEGER NOT NULL DEFAULT 0,
    UNIQUE(commit_hash, file_path)
);
-- Optimization Indexes for Analytics (Bus Factor, Churn)
CREATE INDEX idx_commit_files_repository_id ON commit_files(repository_id);
CREATE INDEX idx_commit_files_commit_hash ON commit_files(commit_hash);
CREATE INDEX idx_commit_files_file_path ON commit_files(file_path);
-- Compound index for specific file history queries
CREATE INDEX idx_commit_files_repo_file ON commit_files(repository_id, file_path);
-- Modify Contributors (Remove aggregates)
ALTER TABLE contributors DROP COLUMN IF EXISTS commit_count,
    DROP COLUMN IF EXISTS lines_added,
    DROP COLUMN IF EXISTS lines_deleted;
-- Modify Commits (Remove aggregates)
ALTER TABLE commits DROP COLUMN IF EXISTS additions,
    DROP COLUMN IF EXISTS deletions;