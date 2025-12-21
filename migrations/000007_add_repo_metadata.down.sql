-- Remove metadata columns
ALTER TABLE repositories DROP COLUMN name;
ALTER TABLE repositories DROP COLUMN description;
ALTER TABLE repositories DROP COLUMN is_private;
ALTER TABLE repositories DROP COLUMN provider;
-- Remove index
DROP INDEX idx_repositories_user_id_status;