-- Drop triggers
DROP TRIGGER IF EXISTS update_file_stats_updated_at ON file_stats;
DROP TRIGGER IF EXISTS update_contributors_updated_at ON contributors;
DROP TRIGGER IF EXISTS update_repositories_updated_at ON repositories;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (to respect foreign key constraints)
DROP TABLE IF EXISTS commits;
DROP TABLE IF EXISTS file_stats;
DROP TABLE IF EXISTS contributors;
DROP TABLE IF EXISTS repositories;
