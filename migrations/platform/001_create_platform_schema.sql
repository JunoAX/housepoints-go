-- Platform Database Schema
-- Version: 001
-- Description: Initial platform tables for multi-family routing

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- TABLE: families
-- Purpose: Top-level tenant table, routes to family databases
-- ============================================================================

CREATE TABLE families (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,

    -- Database connection info
    db_host VARCHAR(255) NOT NULL DEFAULT '10.1.10.20',
    db_port INTEGER NOT NULL DEFAULT 5432,
    db_name VARCHAR(100) NOT NULL,
    db_user VARCHAR(100),
    db_password_encrypted TEXT,

    -- Subscription
    plan VARCHAR(20) NOT NULL DEFAULT 'free',
    status VARCHAR(20) NOT NULL DEFAULT 'trial',
    trial_ends_at TIMESTAMPTZ,
    subscription_ends_at TIMESTAMPTZ,

    -- Billing
    stripe_customer_id VARCHAR(255),
    stripe_subscription_id VARCHAR(255),

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    -- Cached statistics
    member_count INTEGER DEFAULT 0,
    storage_used_mb INTEGER DEFAULT 0,
    last_activity_at TIMESTAMPTZ
);

CREATE INDEX idx_families_slug ON families(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_families_status ON families(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_families_plan ON families(plan) WHERE deleted_at IS NULL;

-- Slug validation constraints
ALTER TABLE families ADD CONSTRAINT families_slug_format
    CHECK (slug ~ '^[a-z0-9][a-z0-9-]*[a-z0-9]$' AND slug NOT LIKE '%-%-%');

ALTER TABLE families ADD CONSTRAINT families_slug_not_reserved
    CHECK (slug NOT IN ('api', 'www', 'app', 'admin', 'staging', 'dev', 'support', 'help', 'blog', 'docs'));

-- ============================================================================
-- TABLE: users
-- Purpose: Global user accounts (can belong to multiple families)
-- ============================================================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    password_hash VARCHAR(255) NOT NULL,

    -- Minimal profile (detailed profile in family DB)
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    avatar_url VARCHAR(500),

    -- Security
    mfa_enabled BOOLEAN NOT NULL DEFAULT false,
    mfa_secret VARCHAR(255),
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMPTZ,

    -- Activity
    last_login TIMESTAMPTZ,
    last_login_ip INET,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_last_login ON users(last_login DESC) WHERE deleted_at IS NULL;

-- ============================================================================
-- TABLE: family_memberships
-- Purpose: Links users to families they can access
-- ============================================================================

CREATE TABLE family_memberships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Platform role
    role VARCHAR(20) NOT NULL DEFAULT 'member',

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    invited_at TIMESTAMPTZ,
    joined_at TIMESTAMPTZ,
    last_accessed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(family_id, user_id)
);

CREATE INDEX idx_memberships_user ON family_memberships(user_id) WHERE status = 'active';
CREATE INDEX idx_memberships_family ON family_memberships(family_id) WHERE status = 'active';

-- ============================================================================
-- TABLE: subscriptions
-- Purpose: Billing and subscription tracking
-- ============================================================================

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    family_id UUID NOT NULL UNIQUE REFERENCES families(id) ON DELETE CASCADE,

    plan VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,

    -- Stripe integration
    stripe_subscription_id VARCHAR(255) UNIQUE,
    stripe_customer_id VARCHAR(255),

    -- Billing cycle
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    cancel_at_period_end BOOLEAN DEFAULT false,
    cancelled_at TIMESTAMPTZ,

    -- Trial
    trial_start TIMESTAMPTZ,
    trial_end TIMESTAMPTZ,

    -- Pricing
    amount_cents INTEGER NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_family ON subscriptions(family_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);

-- ============================================================================
-- TABLE: platform_analytics
-- Purpose: Aggregated metrics for admin dashboard
-- ============================================================================

CREATE TABLE platform_analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    metric_date DATE NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    metric_value NUMERIC NOT NULL,

    -- Optional per-family metrics
    family_id UUID REFERENCES families(id) ON DELETE CASCADE,

    -- Segmentation
    plan VARCHAR(20),
    cohort_month DATE,

    metadata JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_analytics_unique_global ON platform_analytics(metric_date, metric_name)
    WHERE family_id IS NULL;
CREATE UNIQUE INDEX idx_analytics_unique_family ON platform_analytics(metric_date, metric_name, family_id)
    WHERE family_id IS NOT NULL;
CREATE INDEX idx_analytics_date_name ON platform_analytics(metric_date DESC, metric_name);
CREATE INDEX idx_analytics_family ON platform_analytics(family_id, metric_date DESC);

-- ============================================================================
-- TABLE: audit_logs
-- Purpose: Security and compliance logging
-- ============================================================================

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Who
    user_id UUID REFERENCES users(id),
    family_id UUID REFERENCES families(id) ON DELETE CASCADE,

    -- What
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id UUID,

    -- Context
    ip_address INET,
    user_agent TEXT,
    request_id UUID,

    -- Changes
    changes JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_family ON audit_logs(family_id, created_at DESC);
CREATE INDEX idx_audit_user ON audit_logs(user_id, created_at DESC);
CREATE INDEX idx_audit_action ON audit_logs(action, created_at DESC);
CREATE INDEX idx_audit_created ON audit_logs(created_at DESC);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_families_updated_at BEFORE UPDATE ON families
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_memberships_updated_at BEFORE UPDATE ON family_memberships
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_subscriptions_updated_at BEFORE UPDATE ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- INITIAL DATA
-- ============================================================================

-- Create platform admin user (you)
INSERT INTO users (id, email, password_hash, first_name, last_name, email_verified)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'tom@gamull.com',
    'CHANGEME_SET_PASSWORD',
    'Tom',
    'Gamull',
    true
);

COMMENT ON DATABASE housepoints_platform IS 'Platform database for HousePoints multi-tenant routing';
COMMENT ON TABLE families IS 'Top-level tenant table with database routing information';
COMMENT ON TABLE users IS 'Global user accounts that can belong to multiple families';
COMMENT ON TABLE family_memberships IS 'Junction table linking users to families';
COMMENT ON TABLE subscriptions IS 'Billing and subscription tracking via Stripe';
COMMENT ON TABLE platform_analytics IS 'Aggregated metrics for platform monitoring';
COMMENT ON TABLE audit_logs IS 'Security audit log for compliance';
