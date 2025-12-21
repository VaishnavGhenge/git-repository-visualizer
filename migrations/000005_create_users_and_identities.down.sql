-- Remove user_id from repositories
ALTER TABLE repositories DROP COLUMN user_id;
-- Drop tables
DROP TABLE user_identities;
DROP TABLE users;