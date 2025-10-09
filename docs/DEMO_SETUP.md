# Demo Family Setup

This document describes the demo family setup for public testing and documentation purposes.

## Overview

The demo family allows public access to test the HousePoints API without requiring user registration. It's designed for:
- API documentation and examples
- Public demos and presentations
- Frontend development testing
- Integration testing

## Database Setup

### Platform Database Entry

The demo family is registered in `housepoints_platform.families`:

```sql
-- Family: Demo Family
-- ID: e5934973-ef64-4e75-bcd6-2a78ac42c833
-- Slug: demo
-- Database: family_demo
```

### Family Database

Database name: `family_demo`
- Created as a copy of the gamull_chores schema (124 tables)
- Contains sample chores, users, and assignments
- Isolated from production family data

## Demo User Credentials

**Username:** `demo`
**Password:** `demo123`
**Email:** `demo@housepoints.ai`
**User ID:** `502cbd81-db5b-4449-a97d-24ea2ec17c59`
**Is Parent:** `true`
**Login Enabled:** `true`

The password is hashed using bcrypt and stored in `users.password_hash`.

## API Access

### Domain Access

Production: `https://demo.housepoints.ai`
Staging: `https://demo.staging.housepoints.ai` (if configured)

### Authentication

#### 1. Login to Get JWT Token

```bash
curl -X POST https://demo.housepoints.ai/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"demo","password":"demo123"}'
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user_id": "502cbd81-db5b-4449-a97d-24ea2ec17c59",
  "username": "demo",
  "is_parent": true,
  "family_id": "e5934973-ef64-4e75-bcd6-2a78ac42c833"
}
```

#### 2. Use Token for Protected Endpoints

```bash
TOKEN="your_jwt_token_here"

curl -H "Authorization: Bearer $TOKEN" \
  https://demo.housepoints.ai/api/chores
```

### Demo-Only Endpoints

For public testing without authentication, demo-only endpoints are available:

```bash
# List chores (demo family only, no auth required)
curl https://demo.housepoints.ai/api/demo/chores
```

These endpoints use the `DemoOnly` middleware which restricts access to the demo family slug.

## Middleware Implementation

### DemoOnly Middleware

Located in `internal/middleware/demo_only.go`:

```go
func DemoOnly() gin.HandlerFunc {
    return func(c *gin.Context) {
        slug, exists := GetFamilySlug(c)
        if !exists {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": "Family context required",
            })
            c.Abort()
            return
        }

        if slug != "demo" {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "This endpoint is currently only available for the demo family at demo.housepoints.ai",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}
```

### Usage in Routes

```go
// Public demo endpoint (no auth)
r.GET("/api/demo/chores", middleware.RequireFamily(), middleware.DemoOnly(), handlers.ListChores)

// Protected endpoint (requires auth)
protected := r.Group("/api")
protected.Use(middleware.RequireFamily(), middleware.RequireAuth(jwtService))
{
    protected.GET("/chores", handlers.ListChores)
}
```

## Sample Data

The demo database contains:
- **47 chores** across various categories (bathroom, kitchen, bedroom, etc.)
- Sample users (demo user + child users)
- Sample assignments
- Sample points/rewards data

## Security Considerations

1. **Read-Only Recommended**: Demo endpoints should ideally be read-only to prevent abuse
2. **Rate Limiting**: Consider adding rate limits to demo endpoints
3. **Data Reset**: Plan to periodically reset demo data to a clean state
4. **No PII**: Never store real personally identifiable information in demo database

## Recreating Demo Database

To recreate the demo database from scratch:

```bash
# Connect to PostgreSQL
psql -h 10.1.10.20 -U postgres

# Create database
CREATE DATABASE family_demo;

# Copy schema from gamull (or use migration files)
pg_dump -h 10.1.10.20 -U postgres --schema-only gamull_chores | \
  psql -h 10.1.10.20 -U postgres family_demo

# Apply password auth migration
psql -h 10.1.10.20 -U postgres -d family_demo -f migrations/family/001_add_password_auth.sql

# Create demo user
psql -h 10.1.10.20 -U postgres -d family_demo <<EOF
UPDATE users
SET
  username = 'demo',
  email = 'demo@housepoints.ai',
  password_hash = '\$2b\$12\$keak54LkhXH1H.q7/FuFS.UAuoxAlb2zqTBidaePuDpIZ4CCmZzpO',
  is_parent = true,
  login_enabled = true,
  password_updated_at = NOW()
WHERE id = '502cbd81-db5b-4449-a97d-24ea2ec17c59';
EOF

# Register in platform database
psql -h 10.1.10.20 -U postgres -d housepoints_platform <<EOF
INSERT INTO families (id, slug, name, subdomain, plan, status, database_name, database_host, database_port)
VALUES (
  'e5934973-ef64-4e75-bcd6-2a78ac42c833',
  'demo',
  'Demo Family',
  'demo',
  'free',
  'active',
  'family_demo',
  '10.1.10.20',
  5432
)
ON CONFLICT (id) DO UPDATE SET
  slug = EXCLUDED.slug,
  name = EXCLUDED.name,
  database_name = EXCLUDED.database_name;
EOF
```

## Testing Checklist

After deploying changes, verify demo access:

- [ ] Login works: `POST /api/auth/login` with demo credentials
- [ ] JWT token is valid and not expired
- [ ] Protected endpoints work with token: `GET /api/chores`
- [ ] Protected endpoints reject requests without token
- [ ] Demo-only endpoints work without auth: `GET /api/demo/chores`
- [ ] Demo-only endpoints block non-demo families
- [ ] Family info endpoint works: `GET /api/family/info`

## Maintenance

### Resetting Demo Data

Consider creating a script to reset demo data weekly/monthly to keep it clean:

```bash
# Script: reset_demo_data.sh
# Reset assignments, clear old data, restore sample chores
```

### Monitoring

Monitor demo usage to:
- Detect abuse or unusual traffic patterns
- Understand which endpoints are most used
- Gather feedback for API improvements

## Related Documentation

- [JWT Authentication](./AUTHENTICATION.md) (if created)
- [API Documentation](./API.md) (if created)
- [Multi-Tenant Architecture](../GO_MIGRATION_PLAN.md)
