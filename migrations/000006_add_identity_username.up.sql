-- Add provider_username to user_identities
ALTER TABLE user_identities
ADD COLUMN provider_username TEXT;