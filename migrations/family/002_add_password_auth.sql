-- Migration: Add password authentication support to family databases
-- This adds the password_hash column needed for Go backend authentication
-- Run this for each family database when migrating from Python to Go backend

-- Add password_hash column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'users' AND column_name = 'password_hash'
    ) THEN
        ALTER TABLE users ADD COLUMN password_hash TEXT;

        -- Create index for faster lookups
        CREATE INDEX IF NOT EXISTS idx_users_password_hash ON users(password_hash) WHERE password_hash IS NOT NULL;

        RAISE NOTICE 'Added password_hash column to users table';
    ELSE
        RAISE NOTICE 'password_hash column already exists';
    END IF;
END $$;

-- Ensure login_enabled column exists (should already be there)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'users' AND column_name = 'login_enabled'
    ) THEN
        ALTER TABLE users ADD COLUMN login_enabled BOOLEAN DEFAULT true;
        RAISE NOTICE 'Added login_enabled column to users table';
    ELSE
        RAISE NOTICE 'login_enabled column already exists';
    END IF;
END $$;

-- Show summary of users and their auth status
SELECT
    username,
    is_parent,
    login_enabled,
    password_hash IS NOT NULL as has_password_hash,
    CASE
        WHEN password_hash IS NOT NULL THEN 'Ready for Go backend'
        WHEN login_enabled = true THEN 'Needs password hash'
        ELSE 'Login disabled'
    END as auth_status
FROM users
ORDER BY is_parent DESC, username;
