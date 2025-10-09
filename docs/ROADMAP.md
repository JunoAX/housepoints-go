# HousePoints Go Migration Roadmap

## Current Status (v0.4.0)

✅ **Completed:**
- Multi-tenant database architecture (platform DB + family DBs)
- Subdomain-based routing (*.housepoints.ai)
- Connection pooling for family databases
- JWT authentication with login endpoint
- Password-based authentication (bcrypt)
- Authentication middleware
- Demo family setup for testing
- First protected endpoint (GET /api/chores)
- Production deployment infrastructure (Kubernetes/Flux)

🎯 **Currently Deployed:**
- Production: `demo.housepoints.ai` with JWT auth
- Staging: `staging.housepoints.ai` with JWT auth
- Legacy: `chores.gamull.com` (Python backend, untouched)

## Phase 1: Core API Development (Current Phase)

### 1.1 Essential Read Endpoints
Priority: **HIGH** | Estimated: 2-3 days

Build out read-only endpoints for core data:
- [ ] Users endpoints
  - `GET /api/users` - List all family users
  - `GET /api/users/:id` - Get user details
  - `GET /api/users/me` - Get current authenticated user
- [ ] Assignments endpoints
  - `GET /api/assignments` - List assignments (with filters: user, date range, status)
  - `GET /api/assignments/:id` - Get assignment details
- [ ] Points/Balance endpoints
  - `GET /api/users/:id/points` - Get user point balance
  - `GET /api/users/:id/history` - Get point history

**Why First:** These are the most frequently accessed endpoints and needed for mobile app read functionality.

### 1.2 Write Endpoints (Mutations)
Priority: **HIGH** | Estimated: 3-4 days

Add ability to modify data:
- [ ] Assignment mutations
  - `POST /api/assignments` - Create assignment (parents only)
  - `PATCH /api/assignments/:id` - Update assignment
  - `DELETE /api/assignments/:id` - Delete assignment (parents only)
  - `POST /api/assignments/:id/complete` - Mark as completed
  - `POST /api/assignments/:id/verify` - Verify completion (parents only)
- [ ] Chore mutations
  - `POST /api/chores` - Create chore (parents only)
  - `PATCH /api/chores/:id` - Update chore (parents only)
  - `DELETE /api/chores/:id` - Delete chore (parents only)

**Why Next:** Enables full CRUD operations for daily usage.

### 1.3 File Upload (Photo Verification)
Priority: **MEDIUM** | Estimated: 2 days

- [ ] Image upload endpoint for assignment verification
- [ ] S3/Object storage integration
- [ ] Image processing/thumbnails
- [ ] Cleanup old images

**Why:** Required for chores that need photo verification.

## Phase 2: User Management & Onboarding

### 2.1 Family Registration
Priority: **HIGH** | Estimated: 3-4 days

Build signup flow for new families:
- [ ] Family registration endpoint
  - Create family database (copy schema)
  - Register in platform database
  - Create initial parent user
- [ ] Email verification (optional)
- [ ] Onboarding wizard data
  - Initial chores setup
  - Add children
  - Configure settings

**Why:** Needed to onboard new families beyond demo.

### 2.2 User Management
Priority: **MEDIUM** | Estimated: 2 days

- [ ] Add child users (parents only)
- [ ] Update user profiles
- [ ] Password reset flow
- [ ] Disable/enable users

**Why:** Families need to manage their children's accounts.

### 2.3 OAuth Integration
Priority: **LOW** | Estimated: 3-4 days

- [ ] Google OAuth
- [ ] Apple Sign In
- [ ] Link OAuth accounts to existing users
- [ ] Account merging

**Why:** Better user experience, but password auth works for now.

## Phase 3: Migration of Gamull Family

### 3.1 Gamull Authentication Setup
Priority: **HIGH** | Estimated: 1 day

- [ ] Apply password migration to gamull_chores database
- [ ] Create bcrypt passwords for gamull users
- [ ] Test login with gamull users
- [ ] Verify gamull.housepoints.ai routing

**Why:** First step to migrate real family.

### 3.2 Parallel Operation
Priority: **HIGH** | Estimated: 2-3 days

- [ ] Configure mobile app to use both endpoints
- [ ] Feature flag system (Go vs Python endpoints)
- [ ] Monitoring/logging for both systems
- [ ] Error tracking

**Why:** Allows gradual migration with rollback capability.

### 3.3 Full Migration
Priority: **HIGH** | Estimated: 1 week

- [ ] Migrate all Python endpoints to Go
- [ ] Data migration verification
- [ ] End-to-end testing with real usage
- [ ] Performance comparison
- [ ] Switch mobile app fully to Go backend
- [ ] Deprecate Python backend

**Why:** Complete the migration for first real family.

## Phase 4: Advanced Features

### 4.1 Real-time Updates
Priority: **MEDIUM** | Estimated: 1 week

- [ ] WebSocket support
- [ ] Real-time assignment updates
- [ ] Live notifications
- [ ] Presence indicators

**Why:** Better UX for multi-user families.

### 4.2 Rewards & Store
Priority: **MEDIUM** | Estimated: 3-4 days

- [ ] Rewards catalog endpoints
- [ ] Redemption system
- [ ] Transaction history
- [ ] Approval workflow

**Why:** Key feature for motivation system.

### 4.3 Scheduling & Automation
Priority: **MEDIUM** | Estimated: 1 week

- [ ] Recurring chore assignment
- [ ] Schedule templates
- [ ] Auto-assignment based on rotation
- [ ] Reminder system

**Why:** Reduces manual assignment work.

### 4.4 Analytics & Reporting
Priority: **LOW** | Estimated: 3-4 days

- [ ] Weekly/monthly reports
- [ ] Completion statistics
- [ ] Point trends
- [ ] Leaderboards

**Why:** Insights for families.

## Phase 5: Scale & Performance

### 5.1 Caching Layer
Priority: **MEDIUM** | Estimated: 2-3 days

- [ ] Redis caching for family data
- [ ] User session caching
- [ ] Query result caching
- [ ] Cache invalidation strategy

**Why:** Improve response times at scale.

### 5.2 Database Optimization
Priority: **MEDIUM** | Estimated: 2 days

- [ ] Query optimization
- [ ] Index analysis
- [ ] Connection pool tuning
- [ ] Database migrations management

**Why:** Handle 10,000 families efficiently.

### 5.3 Monitoring & Observability
Priority: **HIGH** | Estimated: 2-3 days

- [ ] Prometheus metrics
- [ ] Grafana dashboards
- [ ] Error tracking (Sentry)
- [ ] Performance monitoring
- [ ] Alerting system

**Why:** Critical for production reliability.

## Phase 6: Multi-tenant Features

### 6.1 Billing Integration
Priority: **LOW** | Estimated: 1 week

- [ ] Stripe integration
- [ ] Subscription plans
- [ ] Usage tracking
- [ ] Invoice generation

**Why:** Monetization for SaaS.

### 6.2 Custom Domains
Priority: **LOW** | Estimated: 3 days

- [ ] Custom domain support
- [ ] SSL certificate management
- [ ] DNS configuration

**Why:** White-label capability.

## Decision Points

### Immediate Next Steps (Choose One)

**Option A: Continue API Development (Recommended)**
- Build out essential endpoints (users, assignments, points)
- Gets us closer to feature parity with Python backend
- Allows gradual testing with demo family
- Estimated: 1-2 weeks

**Option B: Migrate Gamull Now**
- Set up gamull authentication
- Run parallel backends
- Higher risk but validates architecture faster
- Estimated: 1 week

**Option C: Focus on Onboarding**
- Build family registration
- Enable new family signups
- Grow beyond single family
- Estimated: 1 week

### Recommended Path

1. **Week 1-2:** Complete Phase 1.1 & 1.2 (Essential endpoints)
2. **Week 3:** Test thoroughly with demo family
3. **Week 4:** Phase 3.1 & 3.2 (Gamull migration preparation)
4. **Week 5-6:** Phase 3.3 (Full Gamull migration)
5. **Week 7:** Phase 2.1 (Family registration)
6. **Week 8+:** Advanced features based on user feedback

## Success Metrics

- [ ] All Python endpoints migrated to Go
- [ ] Gamull family using Go backend exclusively
- [ ] 5+ new families onboarded
- [ ] <100ms average response time
- [ ] 99.9% uptime
- [ ] Zero data loss during migration

## Notes

- Python backend (`chores.gamull.com`) remains operational during migration
- All changes are backwards compatible
- Focus on stability over features during migration
- Regular backups before any data operations
