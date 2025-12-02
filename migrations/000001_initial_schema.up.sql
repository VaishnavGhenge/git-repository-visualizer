-- Create repositories table
CREATE TABLE repositories (
    id SERIAL PRIMARY KEY,
    url TEXT UNIQUE NOT NULL,
    local_path TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    last_indexed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create index on status for efficient querying of pending/indexing repos
CREATE INDEX idx_repositories_status ON repositories(status);
CREATE INDEX idx_repositories_last_indexed_at ON repositories(last_indexed_at);

-- Create contributors table
CREATE TABLE contributors (
    id SERIAL PRIMARY KEY,
    repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    name TEXT NOT NULL,
    commit_count INTEGER NOT NULL DEFAULT 0,
    lines_added INTEGER NOT NULL DEFAULT 0,
    lines_deleted INTEGER NOT NULL DEFAULT 0,
    first_commit_at TIMESTAMP,
    last_commit_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(repository_id, email)
);

-- Create indexes for efficient querying
CREATE INDEX idx_contributors_repository_id ON contributors(repository_id);
CREATE INDEX idx_contributors_commit_count ON contributors(commit_count DESC);

-- Create file_stats table
CREATE TABLE file_stats (
    id SERIAL PRIMARY KEY,
    repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    total_changes INTEGER NOT NULL DEFAULT 0,
    lines_added INTEGER NOT NULL DEFAULT 0,
    lines_deleted INTEGER NOT NULL DEFAULT 0,
    last_modified_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(repository_id, file_path)
);

-- Create indexes for file stats
CREATE INDEX idx_file_stats_repository_id ON file_stats(repository_id);
CREATE INDEX idx_file_stats_total_changes ON file_stats(total_changes DESC);

-- Create commits table (optional, for detailed history)
CREATE TABLE commits (
    id SERIAL PRIMARY KEY,
    repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    hash TEXT NOT NULL,
    author_email TEXT NOT NULL,
    author_name TEXT NOT NULL,
    message TEXT,
    committed_at TIMESTAMP NOT NULL,
    additions INTEGER NOT NULL DEFAULT 0,
    deletions INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(repository_id, hash)
);

-- Create indexes for commits
CREATE INDEX idx_commits_repository_id ON commits(repository_id);
CREATE INDEX idx_commits_author_email ON commits(author_email);
CREATE INDEX idx_commits_committed_at ON commits(committed_at DESC);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers to auto-update updated_at
CREATE TRIGGER update_repositories_updated_at BEFORE UPDATE ON repositories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_contributors_updated_at BEFORE UPDATE ON contributors
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_file_stats_updated_at BEFORE UPDATE ON file_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
