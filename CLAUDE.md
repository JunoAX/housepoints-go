# HousePoints Go - Development Guide

## Project Context

This is the Go rewrite of HousePoints for multi-tenant support, targeting 10,000 families.
The legacy Python backend remains at `/Users/tom/projects/housepoints`.

## Deployment Strategy

- **Staging**: staging.housepoints.ai (testing environment)
- **Production**: housepoints.ai, app.housepoints.ai, api.housepoints.ai (multi-tenant)
- **Legacy**: chores.gamull.com (Python backend, single family)

## Deployment Process

When deploying:
1. Update version in code if needed
2. Build Docker image: `docker build -t ghcr.io/junoax/housepoints-go:vX.Y.Z .`
3. Push to registry: `docker push ghcr.io/junoax/housepoints-go:vX.Y.Z`
4. Update image tag in rocky repository manifests:
   - Staging: `~/projects/rocky/cluster/apps/production/housepoints-go-staging.yaml`
   - Production: `~/projects/rocky/cluster/apps/production/housepoints-go-production.yaml`
5. Commit and push rocky changes - Flux will auto-deploy in 1-5 minutes
6. Verify deployment:
   ```bash
   kubectl get pods -n production -l app=housepoints-go,env=staging
   kubectl logs -n production -l app=housepoints-go,env=staging --tail=100 -f
   curl https://staging.housepoints.ai/health
   ```

## K3s Cluster Access

All deployments go through k3s cluster on rocky.gamull.com:
- Namespace: `production`
- Service account: `gamull-backend-sa`
- Secrets: `gamull-backend-secrets` (contains DB credentials, JWT secret)
- Redis: `redis.production.svc.cluster.local:6379`
  - Staging uses DB 3
  - Production uses DB 4

## Local Development

Use docker-compose for local PostgreSQL/Redis:
```bash
docker-compose up -d
cp .env.example .env
go run cmd/server/main.go
```

Ports are configured to not conflict with production port-forwarding:
- PostgreSQL: 5433 (local) vs 5432 (production)
- Redis: 6380 (local) vs 6379 (production)

## Migration Context

This Go backend runs **alongside** the Python backend during migration:
- New families onboard to Go backend (housepoints.ai)
- Gamull family stays on Python backend (chores.gamull.com)
- See `/Users/tom/projects/housepoints/docs/GO_MIGRATION_PLAN.md` for full migration roadmap

## Architecture

- **Framework**: Gin (HTTP)
- **Database**: PostgreSQL via pgx/pgxpool
- **Cache**: Redis
- **Multi-tenancy**: Row-Level Security (RLS) with family_id isolation
- **Scaling**: HPA in production (2-20 replicas based on CPU/memory)

## Important Notes

- Never create documentation files unless explicitly requested
- Commit messages should be concise and human-written (no "by Claude")
- Always prefer editing existing files to creating new ones
- When deploying, update BOTH code version AND k8s manifests in rocky repo
