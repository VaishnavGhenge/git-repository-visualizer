-- Remove default_branch column from repositories table
ALTER TABLE repositories 
DROP COLUMN default_branch;
