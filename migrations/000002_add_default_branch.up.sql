-- Add default_branch column to repositories table
ALTER TABLE repositories 
ADD COLUMN default_branch TEXT DEFAULT 'main';
