-- Add password authentication to users table
-- This allows simple password-based login alongside OAuth

ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_updated_at TIMESTAMPTZ;

-- Index for login lookups
CREATE INDEX IF NOT EXISTS idx_users_username_login ON users(username) WHERE login_enabled = true;
CREATE INDEX IF NOT EXISTS idx_users_email_login ON users(email) WHERE login_enabled = true AND email IS NOT NULL;

COMMENT ON COLUMN users.password_hash IS 'Bcrypt hash of user password for simple auth (optional, OAuth is primary)';
