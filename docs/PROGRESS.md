# HousePoints Go Migration - Progress Tracker

**Last Updated**: 2025-10-07

## Current Status: Phase 1 - Foundation Setup ✅

### Deployment Status

#### Staging Environment
- **URL**: https://staging.housepoints.ai
- **Status**: ✅ Running (1 pod)
- **Health**: `{"status":"healthy","version":"dev"}`
- **Resources**: 50m CPU, 64Mi RAM
- **Redis**: DB 3

#### Production Environment
- **URLs**:
  - https://housepoints.ai
  - https://api.housepoints.ai
  - https://*.housepoints.ai (wildcard for families)
- **Status**: ✅ Running (2 pods with HPA)
- **Health**: `{"status":"healthy","version":"dev"}`
- **Resources**: 200m CPU, 256Mi RAM per pod
- **Auto-scaling**: 2-20 replicas based on CPU (70%) and memory (80%)
- **Current utilization**: CPU 0%, Memory 0%
- **Redis**: DB 4

### Completed Tasks

#### Infrastructure (✅ Complete)
- [x] Create GitHub repository (github.com/JunoAX/housepoints-go)
- [x] Initialize Go module with Gin framework
- [x] Create Dockerfile with multi-stage build (amd64 platform)
- [x] Build and push Docker image v0.1.0 to ghcr.io
- [x] Create k3s manifests for staging and production
- [x] Configure Flux GitOps auto-deployment
- [x] Set up wildcard ingress for *.housepoints.ai
- [x] Deploy to staging environment
- [x] Deploy to production environment
- [x] Verify health endpoints working

#### Multi-Tenant Architecture (✅ Complete)
- [x] Design family slug-based subdomain strategy
- [x] Create Family, User, and FamilySettings models
- [x] Implement subdomain extraction middleware
- [x] Build FamilyRepository with slug validation
- [x] Add wildcard subdomain support to ingress
- [x] Configure TLS for *.housepoints.ai

### In Progress Tasks

#### Database Layer (🚧 In Progress)
- [ ] Create database migrations for multi-tenant schema
  - [ ] families table with slug, plan, active
  - [ ] family_settings table
  - [ ] family_members table (user-to-family relationships)
  - [ ] Add family_id columns to existing tables
  - [ ] Create RLS policies for tenant isolation
- [ ] Implement database connection pooling (pgxpool)
- [ ] Add database health checks
- [ ] Create migration runner

#### Core API (⏳ Not Started)
- [ ] Integrate family middleware into main.go
- [ ] Implement JWT authentication
- [ ] Create family signup/onboarding flow
- [ ] Add family-scoped API endpoints
- [ ] Build user management endpoints

### Upcoming Tasks (Prioritized)

#### Phase 2: Authentication & Core Features (Week 1-2)
1. [ ] Database connection and pooling
2. [ ] JWT authentication service
3. [ ] Family signup and onboarding API
4. [ ] User registration and login
5. [ ] Family member management
6. [ ] Session management

#### Phase 3: Business Logic Migration (Week 3-6)
1. [ ] Chores API endpoints
2. [ ] Points/rewards system
3. [ ] Assignments and scheduling
4. [ ] Notifications service
5. [ ] Reports and analytics

#### Phase 4: Testing & Optimization (Week 7-8)
1. [ ] Load testing with 100 concurrent users
2. [ ] Database query optimization
3. [ ] Caching strategy with Redis
4. [ ] Error handling and logging
5. [ ] Monitoring and alerts

#### Phase 5: Production Migration (Month 3)
1. [ ] Migrate Gamull family to Go backend
2. [ ] Parallel run with Python backend
3. [ ] Gradual traffic shift (10% → 50% → 100%)
4. [ ] Data validation and reconciliation
5. [ ] Python backend decommissioning

## Technical Decisions

### URL Strategy: Subdomain-based Routing ✅
**Decision**: Use family slugs as subdomains (e.g., `gamull.housepoints.ai`)

**Rationale**:
- Each family feels ownership of their subdomain
- Clean separation from API routes
- Natural isolation for theming/branding
- Scales well with wildcard SSL
- Easy to share: "Visit us at smith.housepoints.ai"

**Handling Duplicates**:
- First family: `smith.housepoints.ai`
- Subsequent: `smith-nyc.housepoints.ai`, `smith2.housepoints.ai`, etc.
- Families choose their slug during signup with validation

**Reserved Subdomains**: `api`, `www`, `app`, `admin`, `staging`, `dev`

### Architecture: Single Service Start → Microservices
**Current**: Monolithic Go service (0-500 families)
**Scale**: Horizontal pod autoscaling (500-5,000 families)
**Future**: Split to microservices (5,000-10,000 families)

### Database: PostgreSQL with Row-Level Security
**Strategy**: Multi-tenant with family_id isolation via RLS policies
**Connection**: pgxpool for connection pooling
**Migration**: Dual-write mode during transition

## Blockers & Issues

### Active Issues
- None currently

### Resolved Issues
1. ✅ Docker image architecture mismatch (ARM64 vs AMD64) - Fixed by adding `--platform=linux/amd64`
2. ✅ Flux not deploying manifests - Fixed by adding files to kustomization.yaml
3. ✅ Pods crashing with "exec format error" - Fixed by rebuilding for correct architecture

## Metrics & Goals

### Performance Targets
- **Response Time**: < 100ms for 95th percentile
- **Throughput**: 1,000 requests/second per pod
- **Uptime**: 99.9% availability
- **Scale**: Support 10,000 families

### Cost Projections
- **Current**: $80/month (2 pods + staging)
- **At Scale** (10,000 families): $2,200/month
- **Savings vs Python**: 80% reduction (from $18k/month)

## Resources

### Documentation
- [Deployment Guide](/Users/tom/projects/housepoints-go/docs/DEPLOYMENT.md)
- [Migration Plan](/Users/tom/projects/housepoints/docs/GO_MIGRATION_PLAN.md)
- [Development Setup](/Users/tom/projects/housepoints-go/CLAUDE.md)

### Repositories
- **Go Backend**: github.com/JunoAX/housepoints-go
- **Python Backend**: github.com/JunoAX/housepoints
- **K3s Manifests**: ~/projects/rocky/cluster/apps/production/

### Monitoring
```bash
# Check deployment status
kubectl get pods -n production -l app=housepoints-go

# View logs
kubectl logs -n production -l app=housepoints-go,env=staging -f

# Check HPA status
kubectl get hpa -n production housepoints-go-production-hpa

# Test health
curl https://staging.housepoints.ai/health
curl https://api.housepoints.ai/health
```

## Next Session Priorities

### High Priority (Do First)
1. Create database migration files for families schema
2. Implement database connection pooling in main.go
3. Integrate family middleware into application
4. Add basic health check with database connectivity

### Medium Priority
1. Build family signup API endpoint
2. Implement JWT authentication
3. Create user registration flow

### Low Priority
1. Set up monitoring dashboards
2. Write integration tests
3. Document API endpoints
