# HousePoints Go - Multi-Tenant Database Architecture

**Version**: 1.0
**Last Updated**: 2025-10-07
**Database**: PostgreSQL 15+

## Table of Contents
1. [Architecture Overview](#architecture-overview)
2. [Multi-Tenant Strategy](#multi-tenant-strategy)
3. [Schema Organization](#schema-organization)
4. [Core Tables](#core-tables)
5. [Feature Domains](#feature-domains)
6. [Platform Management](#platform-management)
7. [Migration Strategy](#migration-strategy)
8. [Security & Isolation](#security--isolation)

---

## Architecture Overview

### Design Philosophy

**Multi-Tenant with Shared Database**
- Single PostgreSQL database shared across all families
- Tenant isolation via `family_id` foreign key
- Row-Level Security (RLS) policies enforce data isolation
- Cost-effective for 10,000 families
- Centralized backups and maintenance

**Current State (Python)**
- 124 tables in single-tenant schema
- All data belongs to "Gamull" family
- No multi-tenant support

**Target State (Go)**
- All existing tables + new multi-tenant tables
- Every tenant-scoped table has `family_id` column
- Platform-level tables for cross-family management
- Migration path to reuse existing data

---

## Multi-Tenant Strategy

### Three Tiers of Data

```
┌─────────────────────────────────────────────────────────┐
│                   PLATFORM TIER                          │
│  - families                                              │
│  - users (can belong to multiple families)              │
│  - subscriptions, billing, analytics                    │
│  - system_settings (global)                              │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│                   FAMILY TIER                            │
│  - family_settings (per-family config)                  │
│  - family_members (user-to-family relationships)        │
│  - chores, assignments, rewards (scoped to family)      │
│  - All 120+ existing feature tables (+ family_id)       │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│                   USER TIER                              │
│  - user_profiles (family-specific profile data)         │
│  - user_permissions (family-scoped)                     │
│  - points, achievements (per family membership)          │
└─────────────────────────────────────────────────────────┘
```

### Key Concepts

**Family**: Top-level tenant
- Unique slug for subdomain (`gamull.housepoints.ai`)
- Subscription plan (free, premium, enterprise)
- Isolated data via RLS policies

**User**: Can belong to multiple families
- Global identity (email, username, password)
- Separate profile/points per family membership
- Example: Parent managing 2 divorced families, or nanny helping 3 families

**Family Member**: Relationship between user and family
- Role: parent, child, admin, guest
- Active/inactive status
- Created/deleted independently of user account

---

## Schema Organization

### Naming Conventions

**Platform Tables** (no family_id)
```sql
families
users
subscriptions
platform_analytics
billing_invoices
system_settings (global only)
```

**Family-Scoped Tables** (require family_id)
```sql
chores              -- Existing table + family_id column
assignments         -- Existing table + family_id column
rewards             -- Existing table + family_id column
family_settings     -- New table (family-specific config)
family_members      -- New table (user-to-family relationship)
... all 120+ feature tables
```

**Indexes Strategy**
- Every family-scoped table: `idx_{table}_family_id` on (family_id)
- Composite indexes: `idx_{table}_family_lookup` on (family_id, created_at)
- User lookups: `idx_{table}_family_user` on (family_id, user_id)

---

## Core Tables

### 1. Platform Tier

#### `families`
The top-level tenant table.

```sql
CREATE TABLE families (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug VARCHAR(50) UNIQUE NOT NULL,              -- Subdomain: gamull, smith-nyc
    name VARCHAR(100) NOT NULL,                     -- Display name: "The Gamull Family"
    plan VARCHAR(20) NOT NULL DEFAULT 'free',       -- free, premium, enterprise
    active BOOLEAN NOT NULL DEFAULT true,
    stripe_customer_id VARCHAR(255),                -- Billing integration
    trial_ends_at TIMESTAMPTZ,
    subscription_ends_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ                          -- Soft delete
);

CREATE INDEX idx_families_slug ON families(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_families_active ON families(active) WHERE deleted_at IS NULL;

-- Constraints
ALTER TABLE families ADD CONSTRAINT families_slug_format
    CHECK (slug ~ '^[a-z0-9][a-z0-9-]*[a-z0-9]$' AND slug NOT LIKE '%-%-%');
```

**Reserved Slugs**: `api`, `www`, `app`, `admin`, `staging`, `dev`, `support`, `help`, `blog`

---

#### `users`
Global user accounts that can belong to multiple families.

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(50) UNIQUE NOT NULL,           -- Globally unique
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    avatar_url VARCHAR(500),
    email_verified BOOLEAN NOT NULL DEFAULT false,
    active BOOLEAN NOT NULL DEFAULT true,
    last_login TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_username ON users(username) WHERE deleted_at IS NULL;
```

**Note**: This is separate from existing `users` table. Migration will:
1. Create `users` as global identity table
2. Rename existing `users` → `family_member_profiles` (family-scoped data)
3. Link via `family_members` join table

---

#### `family_members`
Junction table connecting users to families.

```sql
CREATE TABLE family_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'child',      -- parent, child, admin, guest
    display_name VARCHAR(100),                       -- Family-specific nickname
    active BOOLEAN NOT NULL DEFAULT true,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    UNIQUE(family_id, user_id)                      -- User can only join family once
);

CREATE INDEX idx_family_members_family ON family_members(family_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_family_members_user ON family_members(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_family_members_active ON family_members(family_id, active) WHERE deleted_at IS NULL;
```

**Roles**:
- `admin`: Full control, billing access
- `parent`: Manage children, approve chores, configure family
- `child`: Complete chores, earn points, view family data
- `guest`: Read-only access (e.g., grandparent checking in)

---

### 2. Family Configuration

#### `family_settings`
Per-family configuration and preferences.

```sql
CREATE TABLE family_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    family_id UUID NOT NULL UNIQUE REFERENCES families(id) ON DELETE CASCADE,

    -- Regional
    timezone VARCHAR(50) NOT NULL DEFAULT 'America/New_York',
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    language VARCHAR(10) NOT NULL DEFAULT 'en-US',
    week_start_day INTEGER NOT NULL DEFAULT 0,      -- 0=Sunday, 1=Monday

    -- Branding
    theme_color VARCHAR(7) DEFAULT '#3498db',
    logo_url VARCHAR(500),
    custom_domain VARCHAR(255),                      -- Premium: family.example.com

    -- Features
    enable_chores BOOLEAN NOT NULL DEFAULT true,
    enable_rewards BOOLEAN NOT NULL DEFAULT true,
    enable_calendar BOOLEAN NOT NULL DEFAULT true,
    enable_meals BOOLEAN NOT NULL DEFAULT true,
    enable_bidding BOOLEAN NOT NULL DEFAULT true,

    -- Limits (plan-based)
    max_children INTEGER NOT NULL DEFAULT 10,
    max_parents INTEGER NOT NULL DEFAULT 2,
    max_chores INTEGER NOT NULL DEFAULT 100,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_family_settings_family ON family_settings(family_id);
```

---

### 3. Family-Scoped Feature Tables

All existing tables from Python backend need `family_id` column added.

#### Example: `chores` (existing table)

**Current Schema (Python)**:
```sql
CREATE TABLE chores (
    id UUID PRIMARY KEY,
    title VARCHAR(255),
    base_points INTEGER,
    created_at TIMESTAMPTZ
);
```

**New Schema (Go + Multi-Tenant)**:
```sql
CREATE TABLE chores (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,  -- NEW
    title VARCHAR(255) NOT NULL,
    description TEXT,
    base_points INTEGER NOT NULL DEFAULT 10,
    category VARCHAR(50),
    created_by UUID REFERENCES users(id),
    updated_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_chores_family ON chores(family_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_chores_family_category ON chores(family_id, category);

-- Row-Level Security
ALTER TABLE chores ENABLE ROW LEVEL SECURITY;

CREATE POLICY chores_family_isolation ON chores
    USING (family_id = current_setting('app.current_family_id')::UUID);
```

**Pattern applies to all 120+ tables**:
- `assignments`, `rewards`, `meal_plans`, `school_events`, etc.
- Add `family_id UUID NOT NULL` column
- Add index on `(family_id)`
- Enable RLS policy for isolation

---

## Feature Domains

### Domain 1: Chores & Assignments
**Tables**: `chores`, `assignments`, `scheduled_chores`, `chore_rotations`, `rotation_assignments`

**Multi-Tenant Changes**:
- Add `family_id` to all tables
- Assignment logic scoped to family members only
- Points earned are family-specific

---

### Domain 2: Points & Rewards
**Tables**: `point_transactions`, `points_history`, `rewards`, `reward_purchases`, `reward_redemptions`

**Multi-Tenant Changes**:
- Each user has separate point balance per family
- New table: `family_member_points`
```sql
CREATE TABLE family_member_points (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    family_id UUID NOT NULL REFERENCES families(id),
    user_id UUID NOT NULL REFERENCES users(id),
    total_points INTEGER NOT NULL DEFAULT 0,
    available_points INTEGER NOT NULL DEFAULT 0,
    lifetime_points INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(family_id, user_id)
);
```

---

### Domain 3: Calendar & Scheduling
**Tables**: `family_schedule`, `family_events`, `school_calendars`, `school_events`, `custody_transitions`

**Multi-Tenant Changes**:
- Each family has own calendar
- School calendars can be shared across families (district-level)
- New table: `shared_school_calendars` (platform tier)

---

### Domain 4: Meals & Food
**Tables**: `meal_plans`, `recipes`, `food_vendors`, `shopping_lists`, `ingredients`

**Multi-Tenant Changes**:
- Meal plans scoped to family
- Recipes can be:
  - Private (family-only)
  - Public (shared across platform)
- Add `is_public BOOLEAN` and `created_by_family_id`

---

### Domain 5: Notifications
**Tables**: `notifications`, `notification_queue`, `notification_subscriptions`

**Multi-Tenant Changes**:
- Notifications are family + user scoped
- Add composite index: `(family_id, user_id, created_at)`

---

## Platform Management

### Admin/Owner Access

**Platform Admin (You)**:
- Special user with `platform_admin` role
- Can view all families, users, analytics
- Access via `admin.housepoints.ai`

**Platform Tables** (your management):

#### `subscriptions`
```sql
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    family_id UUID NOT NULL UNIQUE REFERENCES families(id),
    plan VARCHAR(20) NOT NULL,                      -- free, premium, enterprise
    status VARCHAR(20) NOT NULL,                    -- active, trial, cancelled, expired
    stripe_subscription_id VARCHAR(255),
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    cancel_at_period_end BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

#### `platform_analytics`
```sql
CREATE TABLE platform_analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    family_id UUID REFERENCES families(id),         -- NULL = platform-wide metric
    metric_name VARCHAR(100) NOT NULL,
    metric_value NUMERIC NOT NULL,
    metric_date DATE NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_analytics_family_date ON platform_analytics(family_id, metric_date);
CREATE INDEX idx_analytics_metric ON platform_analytics(metric_name, metric_date);
```

**Common Metrics**:
- `active_families_count`
- `total_users_count`
- `chores_completed_today`
- `points_awarded_total`
- `revenue_mrr`

#### `audit_logs`
```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    family_id UUID REFERENCES families(id),
    user_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,                    -- create_family, delete_user, etc.
    resource_type VARCHAR(50),                       -- families, chores, assignments
    resource_id UUID,
    changes JSONB,                                   -- before/after values
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_family ON audit_logs(family_id, created_at DESC);
CREATE INDEX idx_audit_user ON audit_logs(user_id, created_at DESC);
```

---

## Migration Strategy

### Phase 1: Add Multi-Tenant Tables (Week 1)
```sql
-- New platform tables
CREATE TABLE families (...);
CREATE TABLE users (...);              -- New global users
CREATE TABLE family_members (...);
CREATE TABLE family_settings (...);
CREATE TABLE subscriptions (...);

-- Create Gamull family as first tenant
INSERT INTO families (id, slug, name, plan)
VALUES ('uuid-gamull', 'gamull', 'The Gamull Family', 'premium');
```

### Phase 2: Migrate Existing Users (Week 2)
```sql
-- Rename current users table
ALTER TABLE users RENAME TO legacy_user_profiles;

-- Create new global users table
CREATE TABLE users (...);

-- Migrate: Create global user for each legacy user
INSERT INTO users (id, email, username, password_hash, first_name, last_name)
SELECT
    id,
    COALESCE(email, username || '@placeholder.gamull.com'),
    username,
    'MIGRATED_' || id,  -- Reset passwords, require password reset
    split_part(display_name, ' ', 1),
    split_part(display_name, ' ', 2)
FROM legacy_user_profiles;

-- Create family memberships for all Gamull users
INSERT INTO family_members (family_id, user_id, role, display_name)
SELECT
    'uuid-gamull',
    id,
    CASE WHEN is_parent THEN 'parent' ELSE 'child' END,
    display_name
FROM legacy_user_profiles;
```

### Phase 3: Add family_id to Feature Tables (Week 3-4)
```sql
-- For each of 120+ tables:
ALTER TABLE chores ADD COLUMN family_id UUID;

-- Backfill with Gamull family ID
UPDATE chores SET family_id = 'uuid-gamull';

-- Make NOT NULL after backfill
ALTER TABLE chores ALTER COLUMN family_id SET NOT NULL;

-- Add foreign key
ALTER TABLE chores ADD CONSTRAINT chores_family_fkey
    FOREIGN KEY (family_id) REFERENCES families(id) ON DELETE CASCADE;

-- Add index
CREATE INDEX idx_chores_family ON chores(family_id);

-- Enable RLS
ALTER TABLE chores ENABLE ROW LEVEL SECURITY;
CREATE POLICY chores_family_isolation ON chores
    USING (family_id = current_setting('app.current_family_id')::UUID);
```

**Automation**: Create migration script to apply pattern to all tables
```bash
./scripts/add-family-id.sh chores
./scripts/add-family-id.sh assignments
./scripts/add-family-id.sh rewards
# ... repeat for all 120+ tables
```

### Phase 4: Test & Validate (Week 5)
- Create test family "smith"
- Verify data isolation
- Test RLS policies
- Ensure no cross-family data leaks
- Performance testing with family_id indexes

---

## Security & Isolation

### Row-Level Security (RLS)

**Every family-scoped table gets RLS policy**:

```sql
-- Enable RLS on table
ALTER TABLE {table_name} ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only see their family's data
CREATE POLICY {table}_family_isolation ON {table_name}
    USING (family_id = current_setting('app.current_family_id')::UUID);

-- Policy: Platform admins can see all
CREATE POLICY {table}_platform_admin ON {table_name}
    USING (current_setting('app.user_role', true) = 'platform_admin');
```

**Application Layer**:
```go
// Middleware sets family context for every request
func FamilyMiddleware(c *gin.Context) {
    family := extractFamilyFromSubdomain(c.Request.Host)

    // Set PostgreSQL session variable
    db.Exec("SET app.current_family_id = $1", family.ID)

    // All subsequent queries automatically filtered by RLS
    c.Next()
}
```

### Additional Security

**Connection Pooling**:
- Separate connection per family context
- Prevents session variable cross-contamination

**Foreign Key Cascades**:
```sql
-- Deleting family cascades to all related data
REFERENCES families(id) ON DELETE CASCADE
```

**Soft Deletes**:
- Never hard delete families
- Use `deleted_at` timestamp
- Allows data recovery and compliance

---

## Indexes & Performance

### Standard Indexes (Every Table)

```sql
-- Primary lookup
CREATE INDEX idx_{table}_family ON {table}(family_id)
    WHERE deleted_at IS NULL;

-- Time-based queries
CREATE INDEX idx_{table}_family_created ON {table}(family_id, created_at DESC);

-- User-scoped queries
CREATE INDEX idx_{table}_family_user ON {table}(family_id, user_id)
    WHERE deleted_at IS NULL;
```

### Query Patterns

**Efficient** (uses index):
```sql
SELECT * FROM chores
WHERE family_id = 'uuid-gamull'
  AND deleted_at IS NULL
ORDER BY created_at DESC;
```

**Inefficient** (full table scan):
```sql
-- Missing family_id filter
SELECT * FROM chores WHERE title LIKE '%clean%';
```

### Performance Targets

- **Single family query**: < 10ms
- **Index scan ratio**: > 99%
- **Table size with 10k families**: ~10GB for chores table
- **Query cost with proper indexes**: O(log n) per family

---

## Future Considerations

### Scaling Beyond 10,000 Families

**Option 1: Sharding by Family ID**
- Split database into shards (e.g., 10 shards = 1,000 families each)
- Route queries based on `family_id % 10`
- Requires connection pooling changes

**Option 2: Separate Databases for Enterprise**
- Large families (>100 users) get dedicated database
- Update `families` table: `database_url` column

**Option 3: Read Replicas**
- Read-heavy families use replica
- Write-heavy stay on primary

### Data Retention

**Policy by Plan**:
- Free: 90 days history
- Premium: 1 year history
- Enterprise: Unlimited

**Implementation**:
```sql
CREATE TABLE data_retention_policies (
    family_id UUID PRIMARY KEY REFERENCES families(id),
    chores_retention_days INTEGER NOT NULL DEFAULT 90,
    points_retention_days INTEGER NOT NULL DEFAULT 365,
    audit_retention_days INTEGER NOT NULL DEFAULT 730
);

-- Automated cleanup job
DELETE FROM assignments
WHERE family_id = $1
  AND created_at < NOW() - INTERVAL '90 days'
  AND family_id IN (SELECT family_id FROM families WHERE plan = 'free');
```

---

## Summary

### Key Design Decisions

1. **Shared Database**: Cost-effective for 10k families, centralized management
2. **Family ID Pattern**: Every table gets `family_id`, RLS for isolation
3. **Global Users**: Users can manage multiple families (divorced parents, nannies)
4. **Slug-Based Routing**: Family slug in subdomain for clean UX
5. **Backward Compatible**: Gamull family migrates seamlessly, existing data preserved

### Tables Created

**New Tables (8)**:
- `families`
- `users` (global identity)
- `family_members`
- `family_settings`
- `subscriptions`
- `platform_analytics`
- `audit_logs`
- `family_member_points`

**Modified Tables (120+)**:
- All existing feature tables get `family_id` column
- All existing tables get RLS policies
- All existing tables get family_id indexes

### Migration Effort

- **Database changes**: 2-3 weeks
- **Application changes**: 4-6 weeks
- **Testing & validation**: 1-2 weeks
- **Total**: 2-3 months to production-ready

---

## Next Steps

1. Create migration files for new tables
2. Write `add-family-id.sh` automation script
3. Implement RLS policies template
4. Set up family context middleware in Go
5. Test with 2 families (gamull + test)
6. Performance benchmark with 100 simulated families
