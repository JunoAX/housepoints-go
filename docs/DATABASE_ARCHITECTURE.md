# HousePoints Go - Database-Per-Family Architecture

**Version**: 2.0 (Database-Per-Family)
**Last Updated**: 2025-10-07
**Database**: PostgreSQL 15+

## Table of Contents
1. [Architecture Overview](#architecture-overview)
2. [Why Database-Per-Family](#why-database-per-family)
3. [Schema Organization](#schema-organization)
4. [Platform Database](#platform-database)
5. [Family Databases](#family-databases)
6. [Database Provisioning](#database-provisioning)
7. [Connection Management](#connection-management)
8. [Migration Strategy](#migration-strategy)
9. [Operations & Management](#operations--management)

---

## Architecture Overview

### Design Philosophy

**Physical Database Isolation**
- Each family gets their own PostgreSQL database
- Platform database for routing, authentication, billing
- Zero risk of cross-family data leaks
- Perfect for sensitive family data (schedules, medical info, school data)

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    PLATFORM DATABASE                         │
│              (housepoints_platform)                          │
│                                                              │
│  Tables:                                                     │
│  - families (routing table)                                 │
│  - users (global authentication)                            │
│  - subscriptions (billing)                                  │
│  - platform_analytics (aggregated metrics)                  │
│  - audit_logs (security)                                    │
└─────────────────────────────────────────────────────────────┘
                            ↓
        ┌───────────────────┴───────────────────┐
        ↓                                        ↓
┌──────────────────────┐              ┌──────────────────────┐
│  FAMILY DATABASE     │              │  FAMILY DATABASE     │
│  (family_gamull)     │              │  (family_smith)      │
│                      │              │                      │
│  All feature tables: │              │  All feature tables: │
│  - users             │              │  - users             │
│  - chores            │              │  - chores            │
│  - assignments       │              │  - assignments       │
│  - rewards           │              │  - rewards           │
│  - family_schedule   │              │  - family_schedule   │
│  - school_events     │              │  - school_events     │
│  ... 120+ tables     │              │  ... 120+ tables     │
│                      │              │                      │
│  NO family_id needed │              │  NO family_id needed │
└──────────────────────┘              └──────────────────────┘
```

### Key Benefits

✅ **Perfect Data Isolation**: Physical separation, zero risk of data leaks
✅ **Compliance Friendly**: Easy to prove FERPA/COPPA compliance for sensitive family data
✅ **Independent Scaling**: Big families get bigger DBs
✅ **Simpler Schema**: No family_id columns needed
✅ **Easy Backup/Restore**: One family at a time
✅ **Trust & Marketing**: "Your family's data is completely isolated"

---

## Why Database-Per-Family

### The Problem with Shared Databases

**Risk of Data Leakage**:
- One RLS policy bug exposes all families' data
- Complex queries across 120+ tables = high bug risk
- Developer mistakes can't be caught by tests alone

**Sensitive Data Types in HousePoints**:
- Children's school schedules and events
- Medical appointments and doctor visits
- Custody arrangements (divorced parents)
- Home addresses and contact information
- Behavioral data (chore completion, points)

**Compliance Requirements**:
- **COPPA** (Children's Online Privacy Protection Act): Kids under 13
- **FERPA** (Family Educational Rights and Privacy Act): School-related data
- **GDPR**: European families' right to data deletion
- **CCPA** (California Consumer Privacy Act): California families

### Why It Works for HousePoints

**Scale is Manageable**:
- Year 1 target: 100 families
- Year 3 target: 1,000 families
- PostgreSQL can handle 10,000 databases on one server

**Cost is Reasonable**:
- Small family DB: ~1GB storage, minimal compute
- 100 families × $7/month = $700/month (managed)
- OR: Self-host 100 DBs on one server for $40/month

**Operational Simplicity**:
- Automated database provisioning on signup
- Backups are per-family (easy restore)
- Can delete/archive inactive families cleanly

---

## Schema Organization

### Two Database Types

#### 1. Platform Database (Single Instance)
**Name**: `housepoints_platform`
**Purpose**: Routing, authentication, billing, analytics
**Size**: Small (~100MB for 10,000 families)

#### 2. Family Databases (One Per Family)
**Name Pattern**: `family_{slug}` (e.g., `family_gamull`, `family_smith`)
**Purpose**: All family-specific data
**Size**: Variable (100MB - 10GB depending on usage)

### No Family ID Columns!

**Old approach (shared DB)**:
```sql
CREATE TABLE chores (
    id UUID PRIMARY KEY,
    family_id UUID NOT NULL,  -- ❌ Not needed anymore!
    title VARCHAR(255)
);
```

**New approach (database-per-family)**:
```sql
CREATE TABLE chores (
    id UUID PRIMARY KEY,
    title VARCHAR(255)
    -- No family_id needed - entire DB belongs to one family
);
```

**Benefits**:
- Simpler queries (no family_id in WHERE clause)
- Faster queries (no family_id index needed)
- Smaller tables (one less column per row)
- Less migration work (don't need to add family_id to 120+ tables)

---

## Platform Database

### Schema: Platform Tables

#### `families`
Routing table to find family's database.

```sql
CREATE TABLE families (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug VARCHAR(50) UNIQUE NOT NULL,               -- Subdomain: gamull, smith-nyc
    name VARCHAR(100) NOT NULL,                      -- Display name: "The Gamull Family"

    -- Database connection
    db_host VARCHAR(255) NOT NULL,                   -- Database server hostname
    db_port INTEGER NOT NULL DEFAULT 5432,
    db_name VARCHAR(100) NOT NULL,                   -- family_gamull
    db_user VARCHAR(100),                            -- Optional: per-family DB user
    db_password_encrypted TEXT,                      -- Encrypted connection password

    -- Subscription
    plan VARCHAR(20) NOT NULL DEFAULT 'free',        -- free, premium, enterprise
    status VARCHAR(20) NOT NULL DEFAULT 'trial',     -- trial, active, suspended, cancelled
    trial_ends_at TIMESTAMPTZ,
    subscription_ends_at TIMESTAMPTZ,

    -- Billing
    stripe_customer_id VARCHAR(255),
    stripe_subscription_id VARCHAR(255),

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,                          -- Soft delete

    -- Statistics (cached from family DB)
    member_count INTEGER DEFAULT 0,
    storage_used_mb INTEGER DEFAULT 0,
    last_activity_at TIMESTAMPTZ
);

CREATE INDEX idx_families_slug ON families(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_families_status ON families(status) WHERE deleted_at IS NULL;

-- Reserved slugs that cannot be used
ALTER TABLE families ADD CONSTRAINT families_slug_not_reserved
    CHECK (slug NOT IN ('api', 'www', 'app', 'admin', 'staging', 'dev', 'support', 'help', 'blog', 'docs'));

-- Slug format validation
ALTER TABLE families ADD CONSTRAINT families_slug_format
    CHECK (slug ~ '^[a-z0-9][a-z0-9-]*[a-z0-9]$' AND slug NOT LIKE '%-%-%');
```

---

#### `users`
Global user accounts (can manage multiple families).

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    password_hash VARCHAR(255) NOT NULL,

    -- Profile (minimal - detailed profile in family DB)
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
CREATE INDEX idx_users_last_login ON users(last_login DESC);
```

**Note**: Username is NOT globally unique (can be same across families). Email is the unique identifier.

---

#### `family_memberships`
Links users to families they can access.

```sql
CREATE TABLE family_memberships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Role in platform (affects what they can do via API)
    role VARCHAR(20) NOT NULL DEFAULT 'member',      -- owner, admin, member

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'active',    -- active, invited, suspended
    invited_at TIMESTAMPTZ,
    joined_at TIMESTAMPTZ,
    last_accessed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(family_id, user_id)
);

CREATE INDEX idx_memberships_user ON family_memberships(user_id) WHERE status = 'active';
CREATE INDEX idx_memberships_family ON family_memberships(family_id) WHERE status = 'active';
```

**Roles**:
- `owner`: Created the family, full access including billing
- `admin`: Co-parent, full family management, no billing access
- `member`: Regular access (parent or child role defined in family DB)

---

#### `subscriptions`
Billing and subscription tracking.

```sql
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    family_id UUID NOT NULL UNIQUE REFERENCES families(id) ON DELETE CASCADE,

    plan VARCHAR(20) NOT NULL,                       -- free, premium, enterprise
    status VARCHAR(20) NOT NULL,                     -- trialing, active, past_due, cancelled

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
    amount_cents INTEGER NOT NULL,                   -- 1500 = $15.00
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_family ON subscriptions(family_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
```

---

#### `platform_analytics`
Aggregated metrics for your dashboard.

```sql
CREATE TABLE platform_analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    metric_date DATE NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    metric_value NUMERIC NOT NULL,

    -- Optional: per-family metrics
    family_id UUID REFERENCES families(id) ON DELETE CASCADE,

    -- Optional: segmentation
    plan VARCHAR(20),                                -- Which plan tier
    cohort_month DATE,                               -- When family signed up

    metadata JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(metric_date, metric_name, family_id)
);

CREATE INDEX idx_analytics_date_name ON platform_analytics(metric_date DESC, metric_name);
CREATE INDEX idx_analytics_family ON platform_analytics(family_id, metric_date DESC);
```

**Example Metrics**:
- `total_families`: Active families count
- `active_users_daily`: DAU across platform
- `chores_completed_daily`: Total chores completed
- `revenue_mrr`: Monthly recurring revenue
- `storage_used_gb`: Total storage across all families

---

#### `audit_logs`
Security and compliance logging.

```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Who
    user_id UUID REFERENCES users(id),
    family_id UUID REFERENCES families(id) ON DELETE CASCADE,

    -- What
    action VARCHAR(100) NOT NULL,                    -- signup, login, delete_family, etc.
    resource_type VARCHAR(50),                       -- families, users, subscriptions
    resource_id UUID,

    -- Context
    ip_address INET,
    user_agent TEXT,
    request_id UUID,

    -- Changes (for update actions)
    changes JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_family ON audit_logs(family_id, created_at DESC);
CREATE INDEX idx_audit_user ON audit_logs(user_id, created_at DESC);
CREATE INDEX idx_audit_action ON audit_logs(action, created_at DESC);
```

---

## Family Databases

### Schema: Family Tables

Each family database contains **all 120+ feature tables** from the existing Python backend.

**No changes needed!** The existing schema works as-is because:
- No family_id columns needed
- No RLS policies needed
- No cross-family queries
- Entire database belongs to one family

### Example: Existing Tables

```sql
-- Users table (family-specific profiles)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    age INTEGER,
    is_parent BOOLEAN DEFAULT false,
    total_points INTEGER DEFAULT 0,
    avatar_url VARCHAR(255),
    preferences JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Chores table
CREATE TABLE chores (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    base_points INTEGER DEFAULT 10,
    category VARCHAR(50),
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Assignments table
CREATE TABLE assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chore_id UUID REFERENCES chores(id),
    assigned_to UUID REFERENCES users(id),
    due_date TIMESTAMPTZ,
    status VARCHAR(20) DEFAULT 'pending',
    points_earned INTEGER,
    verified_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ... 117 more tables exactly as they exist now
```

**Migration is simple**:
1. Export existing `gamull_chores` database
2. Rename to `family_gamull`
3. Done!

---

## Database Provisioning

### Automatic Provisioning on Signup

```go
package database

import (
    "context"
    "fmt"
    "github.com/jackc/pgx/v5/pgxpool"
)

type Provisioner struct {
    platformDB    *pgxpool.Pool
    dbHost        string
    dbPort        int
    adminUser     string
    adminPassword string
}

// ProvisionFamilyDatabase creates a new database for a family
func (p *Provisioner) ProvisionFamilyDatabase(ctx context.Context, familyID uuid.UUID, slug string) error {
    dbName := fmt.Sprintf("family_%s", slug)

    // 1. Create database
    _, err := p.platformDB.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName))
    if err != nil {
        return fmt.Errorf("create database: %w", err)
    }

    // 2. Connect to new database
    connString := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
        p.adminUser, p.adminPassword, p.dbHost, p.dbPort, dbName)
    familyDB, err := pgxpool.New(ctx, connString)
    if err != nil {
        return fmt.Errorf("connect to family db: %w", err)
    }
    defer familyDB.Close()

    // 3. Run schema migrations
    if err := p.runMigrations(ctx, familyDB); err != nil {
        return fmt.Errorf("run migrations: %w", err)
    }

    // 4. Create default data (e.g., default chore categories)
    if err := p.seedDefaultData(ctx, familyDB); err != nil {
        return fmt.Errorf("seed data: %w", err)
    }

    // 5. Update platform database with connection info
    _, err = p.platformDB.Exec(ctx, `
        UPDATE families
        SET db_host = $1, db_port = $2, db_name = $3
        WHERE id = $4
    `, p.dbHost, p.dbPort, dbName, familyID)

    return err
}

// runMigrations executes all schema migrations on family DB
func (p *Provisioner) runMigrations(ctx context.Context, db *pgxpool.Pool) error {
    // Use golang-migrate or similar
    // Read all .sql files from migrations/family_schema/
    // Execute in order
    return nil
}
```

### Deprovisioning (Account Deletion)

```go
// DeprovisionFamilyDatabase handles account deletion
func (p *Provisioner) DeprovisionFamilyDatabase(ctx context.Context, familyID uuid.UUID) error {
    // 1. Get database name from platform DB
    var dbName string
    err := p.platformDB.QueryRow(ctx, `
        SELECT db_name FROM families WHERE id = $1
    `, familyID).Scan(&dbName)
    if err != nil {
        return err
    }

    // 2. Backup before deleting (compliance requirement)
    if err := p.backupDatabase(ctx, dbName); err != nil {
        return fmt.Errorf("backup failed: %w", err)
    }

    // 3. Terminate all connections
    _, err = p.platformDB.Exec(ctx, fmt.Sprintf(`
        SELECT pg_terminate_backend(pg_stat_activity.pid)
        FROM pg_stat_activity
        WHERE pg_stat_activity.datname = '%s'
          AND pid <> pg_backend_pid()
    `, dbName))

    // 4. Drop database
    _, err = p.platformDB.Exec(ctx, fmt.Sprintf("DROP DATABASE %s", dbName))
    if err != nil {
        return fmt.Errorf("drop database: %w", err)
    }

    // 5. Soft delete family record
    _, err = p.platformDB.Exec(ctx, `
        UPDATE families SET deleted_at = NOW() WHERE id = $1
    `, familyID)

    return err
}
```

---

## Connection Management

### Connection Pool Cache

```go
package database

import (
    "context"
    "fmt"
    "sync"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
)

// FamilyConnectionManager manages database connections for families
type FamilyConnectionManager struct {
    platformDB *pgxpool.Pool
    pools      sync.Map  // map[uuid.UUID]*pgxpool.Pool
    mu         sync.RWMutex
}

// GetFamilyDB returns a connection pool for the family's database
func (m *FamilyConnectionManager) GetFamilyDB(ctx context.Context, familyID uuid.UUID) (*pgxpool.Pool, error) {
    // Check cache
    if pool, ok := m.pools.Load(familyID); ok {
        return pool.(*pgxpool.Pool), nil
    }

    // Not in cache, load from platform DB
    var dbHost string
    var dbPort int
    var dbName string

    err := m.platformDB.QueryRow(ctx, `
        SELECT db_host, db_port, db_name
        FROM families
        WHERE id = $1 AND deleted_at IS NULL
    `, familyID).Scan(&dbHost, &dbPort, &dbName)

    if err != nil {
        return nil, fmt.Errorf("family not found: %w", err)
    }

    // Create connection pool
    connString := fmt.Sprintf("postgresql://housepoints_app:password@%s:%d/%s",
        dbHost, dbPort, dbName)

    pool, err := pgxpool.New(ctx, connString)
    if err != nil {
        return nil, fmt.Errorf("connect to family db: %w", err)
    }

    // Cache for future requests
    m.pools.Store(familyID, pool)

    return pool, nil
}

// GetFamilyDBBySlug is a convenience method for middleware
func (m *FamilyConnectionManager) GetFamilyDBBySlug(ctx context.Context, slug string) (*pgxpool.Pool, error) {
    var familyID uuid.UUID
    err := m.platformDB.QueryRow(ctx, `
        SELECT id FROM families WHERE slug = $1 AND deleted_at IS NULL
    `, slug).Scan(&familyID)

    if err != nil {
        return nil, fmt.Errorf("family not found: %w", err)
    }

    return m.GetFamilyDB(ctx, familyID)
}

// CloseAll closes all family connection pools
func (m *FamilyConnectionManager) CloseAll() {
    m.pools.Range(func(key, value interface{}) bool {
        pool := value.(*pgxpool.Pool)
        pool.Close()
        return true
    })
}
```

### Middleware Integration

```go
package middleware

import (
    "net/http"
    "strings"
    "github.com/gin-gonic/gin"
)

func FamilyDatabaseMiddleware(connManager *database.FamilyConnectionManager, baseDomain string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Extract family slug from subdomain
        host := c.Request.Host
        slug := extractSlug(host, baseDomain)

        if slug == "" {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": "Please access via your family subdomain (e.g., yourfamily.housepoints.ai)",
            })
            c.Abort()
            return
        }

        // Get family's database connection
        familyDB, err := connManager.GetFamilyDBBySlug(c.Request.Context(), slug)
        if err != nil {
            c.JSON(http.StatusNotFound, gin.H{
                "error": "Family not found",
                "slug":  slug,
            })
            c.Abort()
            return
        }

        // Store in context for handlers
        c.Set("family_slug", slug)
        c.Set("family_db", familyDB)

        c.Next()
    }
}

func extractSlug(host, baseDomain string) string {
    // Remove port
    if idx := strings.Index(host, ":"); idx != -1 {
        host = host[:idx]
    }

    // Check if subdomain exists
    if !strings.HasSuffix(host, "."+baseDomain) {
        return ""
    }

    slug := strings.TrimSuffix(host, "."+baseDomain)

    // Filter out reserved subdomains
    reserved := map[string]bool{
        "api": true, "www": true, "app": true,
        "admin": true, "staging": true,
    }

    if reserved[slug] {
        return ""
    }

    return slug
}
```

---

## Migration Strategy

### Phase 1: Platform Database Setup (Week 1)

```sql
-- Create platform database
CREATE DATABASE housepoints_platform;

-- Connect and create tables
\c housepoints_platform

-- Create all platform tables
-- (families, users, family_memberships, subscriptions, etc.)
```

### Phase 2: Migrate Gamull Family (Week 2)

```bash
# 1. Backup existing database
pg_dump -h 10.1.10.20 -U postgres gamull_chores > gamull_backup.sql

# 2. Create new family database
psql -h 10.1.10.20 -U postgres -c "CREATE DATABASE family_gamull"

# 3. Restore into new database
psql -h 10.1.10.20 -U postgres -d family_gamull < gamull_backup.sql

# 4. Register in platform database
psql -h 10.1.10.20 -U postgres -d housepoints_platform <<EOF
INSERT INTO families (id, slug, name, plan, db_host, db_port, db_name)
VALUES (
    uuid_generate_v4(),
    'gamull',
    'The Gamull Family',
    'premium',
    '10.1.10.20',
    5432,
    'family_gamull'
);
EOF

# 5. Create user accounts in platform DB
# (Migrate users from family_gamull.users to housepoints_platform.users)
```

### Phase 3: Database Provisioning Automation (Week 3)

```go
// Implement ProvisionFamilyDatabase function
// Test with new test family

func TestFamilyProvisioning(t *testing.T) {
    provisioner := database.NewProvisioner(platformDB, ...)

    // Create test family
    familyID := uuid.New()
    err := provisioner.ProvisionFamilyDatabase(ctx, familyID, "test-family")
    assert.NoError(t, err)

    // Verify database exists
    var exists bool
    platformDB.QueryRow(`
        SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = 'family_test-family')
    `).Scan(&exists)
    assert.True(t, exists)

    // Clean up
    provisioner.DeprovisionFamilyDatabase(ctx, familyID)
}
```

### Phase 4: Production Deployment (Week 4)

```bash
# 1. Deploy Go application with family provisioning
# 2. Point staging.housepoints.ai to Go backend
# 3. Test signup flow creates database
# 4. Verify gamull.housepoints.ai connects to family_gamull
# 5. Monitor database connections and performance
```

---

## Operations & Management

### Backup Strategy

**Per-Family Backups**:
```bash
#!/bin/bash
# backup-family.sh

FAMILY_SLUG=$1
DB_NAME="family_${FAMILY_SLUG}"
BACKUP_DIR="/backups/families/${FAMILY_SLUG}"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# Full database backup
pg_dump -h 10.1.10.20 -U postgres -Fc $DB_NAME > \
    $BACKUP_DIR/${DB_NAME}_${DATE}.dump

# Compress
gzip $BACKUP_DIR/${DB_NAME}_${DATE}.dump

# Upload to S3
aws s3 cp $BACKUP_DIR/${DB_NAME}_${DATE}.dump.gz \
    s3://housepoints-backups/families/${FAMILY_SLUG}/

# Keep only last 30 days locally
find $BACKUP_DIR -name "*.dump.gz" -mtime +30 -delete
```

**Automated Cron**:
```cron
# Daily backups at 2 AM
0 2 * * * /scripts/backup-all-families.sh

# Weekly full backup on Sunday
0 3 * * 0 /scripts/backup-all-families.sh --full
```

### Monitoring

**Database Health Check**:
```go
func (m *FamilyConnectionManager) HealthCheck(ctx context.Context) map[string]bool {
    health := make(map[string]bool)

    // Get all families from platform DB
    rows, _ := m.platformDB.Query(ctx, `
        SELECT id, slug FROM families WHERE deleted_at IS NULL
    `)
    defer rows.Close()

    for rows.Next() {
        var id uuid.UUID
        var slug string
        rows.Scan(&id, &slug)

        // Test connection to family DB
        familyDB, err := m.GetFamilyDB(ctx, id)
        if err != nil {
            health[slug] = false
            continue
        }

        // Ping database
        err = familyDB.Ping(ctx)
        health[slug] = (err == nil)
    }

    return health
}
```

### Database Metrics

**Per-Family Statistics**:
```sql
-- Get size of each family database
SELECT
    datname AS database,
    pg_size_pretty(pg_database_size(datname)) AS size
FROM pg_database
WHERE datname LIKE 'family_%'
ORDER BY pg_database_size(datname) DESC;

-- Active connections per family database
SELECT
    datname AS database,
    count(*) AS connections
FROM pg_stat_activity
WHERE datname LIKE 'family_%'
GROUP BY datname
ORDER BY connections DESC;
```

**Storage Alerts**:
```go
// Alert if family DB exceeds plan limits
func (m *FamilyConnectionManager) CheckStorageLimits(ctx context.Context) error {
    rows, _ := m.platformDB.Query(ctx, `
        SELECT f.id, f.slug, f.plan, f.db_name
        FROM families f
        WHERE f.deleted_at IS NULL
    `)
    defer rows.Close()

    for rows.Next() {
        var id uuid.UUID
        var slug, plan, dbName string
        rows.Scan(&id, &slug, &plan, &dbName)

        // Get database size
        var sizeMB int
        m.platformDB.QueryRow(ctx, `
            SELECT pg_database_size($1) / 1024 / 1024
        `, dbName).Scan(&sizeMB)

        // Check against plan limits
        limit := getPlanStorageLimit(plan)  // free: 100MB, premium: 1GB, enterprise: 10GB

        if sizeMB > limit {
            // Send alert email
            alertFamilyStorageExceeded(slug, sizeMB, limit)
        }
    }

    return nil
}
```

---

## Cost Analysis

### Year 1 (100 Families)

**Managed PostgreSQL (DigitalOcean/AWS RDS)**:
- Small DB (1GB storage, 1vCPU): $7/month each
- 100 families × $7 = $700/month
- Platform DB: $10/month (tiny)
- **Total: $710/month = $8,520/year**

**Self-Hosted (More Control)**:
- PostgreSQL server (4GB RAM, 2vCPU): $40/month
- 100 databases on one server (up to 500 possible)
- Backups to S3: $20/month
- **Total: $60/month = $720/year**

### Year 3 (1,000 Families)

**Managed**: $7,010/month = $84,120/year
**Self-Hosted**: 2 servers × $40 = $80/month + $50 backups = $1,560/year

### Recommended Pricing

To cover costs and provide 50% profit margin:

**Free Tier** (7-day trial):
- 2 parents, 5 children max
- 100MB storage
- Basic features only

**Premium** ($15/month):
- Unlimited family members
- 1GB storage
- All features
- Email support

**Enterprise** ($50/month):
- Unlimited everything
- 10GB storage
- Priority support
- Custom integrations

**Revenue Model (100 families)**:
- 20 free trials → $0
- 70 premium → $1,050/month
- 10 enterprise → $500/month
- **Total: $1,550/month revenue**
- **Profit: $1,550 - $710 = $840/month**

---

## Security Considerations

### Connection Security

**Database User Permissions**:
```sql
-- Create application user with limited permissions
CREATE USER housepoints_app WITH PASSWORD 'secure_password';

-- Grant only necessary permissions
GRANT CONNECT ON DATABASE family_gamull TO housepoints_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO housepoints_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO housepoints_app;

-- Revoke dangerous permissions
REVOKE CREATE ON SCHEMA public FROM housepoints_app;
REVOKE DROP ON ALL TABLES IN SCHEMA public FROM housepoints_app;
```

### Encryption

**At Rest**:
- Enable PostgreSQL encryption (LUKS/dm-crypt)
- Encrypt backups before uploading to S3
- Store sensitive data (SSN, credit cards) separately with field-level encryption

**In Transit**:
- Require SSL/TLS for all database connections
- Use certificate authentication for application

**Connection String Security**:
```go
// Store DB passwords encrypted in platform database
func (m *FamilyConnectionManager) getConnectionString(familyID uuid.UUID) (string, error) {
    var encryptedPassword string
    err := m.platformDB.QueryRow(`
        SELECT db_password_encrypted FROM families WHERE id = $1
    `, familyID).Scan(&encryptedPassword)

    // Decrypt using KMS or similar
    password := decrypt(encryptedPassword)

    return fmt.Sprintf("postgresql://app:%s@%s:%d/%s?sslmode=require",
        password, host, port, dbName), nil
}
```

### Audit Logging

Log all database operations that touch sensitive data:
```go
func (m *FamilyConnectionManager) AuditQuery(ctx context.Context, familyID uuid.UUID, query string, user uuid.UUID) {
    m.platformDB.Exec(ctx, `
        INSERT INTO audit_logs (family_id, user_id, action, metadata)
        VALUES ($1, $2, 'database_query', $3)
    `, familyID, user, map[string]interface{}{
        "query": query,
        "timestamp": time.Now(),
    })
}
```

---

## Summary

### Architecture Benefits

✅ **Perfect Data Isolation**: Each family's data is physically separated
✅ **Compliance Ready**: Easy to prove FERPA/COPPA compliance
✅ **Simple Schema**: No family_id columns needed
✅ **Easy Migration**: Existing schema works as-is
✅ **Scalable**: Can handle 10,000+ families
✅ **Cost Effective**: Self-hosting keeps costs under $100/month for 1,000 families

### Key Design Decisions

1. **Platform DB for Routing**: Small, fast, handles authentication and routing
2. **Family DB Per Family**: Complete isolation, existing schema unchanged
3. **Connection Pool Cache**: Reuse connections, minimal overhead
4. **Automated Provisioning**: New family = new database in seconds
5. **Soft Deletes**: Compliance and data recovery
6. **Per-Family Backups**: Restore individual families without affecting others

### Migration Path

**Week 1**: Platform database setup
**Week 2**: Migrate Gamull as `family_gamull`
**Week 3**: Implement provisioning automation
**Week 4**: Test and deploy to production

### Next Steps

1. Create platform database schema migration
2. Implement FamilyConnectionManager
3. Build database provisioning service
4. Write integration tests
5. Set up backup automation
6. Deploy to staging

---

## Appendix: Database Provisioning Script

```bash
#!/bin/bash
# provision-family-db.sh

set -e

FAMILY_SLUG=$1
DB_NAME="family_${FAMILY_SLUG}"
DB_HOST="10.1.10.20"
DB_PORT=5432
ADMIN_USER="postgres"

if [ -z "$FAMILY_SLUG" ]; then
    echo "Usage: $0 <family-slug>"
    exit 1
fi

echo "Provisioning database for family: $FAMILY_SLUG"

# 1. Create database
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $ADMIN_USER <<EOF
CREATE DATABASE $DB_NAME;
EOF

# 2. Run migrations
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $ADMIN_USER -d $DB_NAME \
    -f migrations/family_schema/001_create_tables.sql

# 3. Create application user and grant permissions
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $ADMIN_USER -d $DB_NAME <<EOF
CREATE USER housepoints_app_${FAMILY_SLUG} WITH PASSWORD '$(openssl rand -base64 32)';
GRANT CONNECT ON DATABASE $DB_NAME TO housepoints_app_${FAMILY_SLUG};
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO housepoints_app_${FAMILY_SLUG};
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO housepoints_app_${FAMILY_SLUG};
EOF

echo "✅ Database $DB_NAME provisioned successfully"
```
