-- Add metadata columns to repositories
ALTER TABLE repositories
ADD COLUMN name TEXT;
ALTER TABLE repositories
ADD COLUMN description TEXT;
ALTER TABLE repositories
ADD COLUMN is_private BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories
ADD COLUMN provider TEXT;
-- 'github', 'gitlab', etc.
-- Update RepositoryStatus enum is not possible in standard SQL easily if it's a domain or just text check.
-- Existing check constraint needs to be updated if it exists. 
-- In our schema it's just TEXT.
-- Add index for user's discovered repos
CREATE INDEX idx_repositories_user_id_status ON repositories(user_id, status);