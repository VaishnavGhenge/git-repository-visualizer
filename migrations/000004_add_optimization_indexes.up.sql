-- Create index for sorting files by lines (size)
CREATE INDEX idx_files_repository_lines ON files(repository_id, lines DESC);
-- Create index for sorting contributors by name
CREATE INDEX idx_contributors_repository_name ON contributors(repository_id, name);