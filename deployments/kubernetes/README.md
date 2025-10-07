# Kubernetes Deployment

## Architecture

HousePoints Go uses **Flux CD** for GitOps-based deployment to k3s.

### Deployment Repository

Kubernetes manifests are stored in the **rocky** repository:
- Location: `~/projects/rocky/cluster/apps/production/`
- File: `housepoints-go-backend.yaml`

### Flux GitOps Workflow

```
1. Push code to github.com/JunoAX/housepoints-go
2. GitHub Actions builds Docker image → ghcr.io/junoax/housepoints-go:vX.X.X
3. Update image tag in rocky/cluster/apps/production/housepoints-go-backend.yaml
4. Push to github.com/JunoAX/rocky
5. Flux automatically syncs and deploys to k3s cluster
```

## Deployment Process

### Manual Deployment

```bash
# 1. Build and push Docker image
cd ~/projects/housepoints-go
make docker-build
make docker-push

# 2. Update version in rocky repo
cd ~/projects/rocky
# Edit cluster/apps/production/housepoints-go-backend.yaml
# Change image: ghcr.io/junoax/housepoints-go:v0.1.0 to new version

# 3. Commit and push
git add cluster/apps/production/housepoints-go-backend.yaml
git commit -m "Deploy housepoints-go v0.2.0"
git push

# 4. Flux will automatically sync within 1-5 minutes
```

### Automated Deployment (Makefile)

```bash
# Deploy with version bump
make deploy
```

## Cluster Information

### Namespace
- `production` - Production environment

### Resources
- **Deployment**: `housepoints-go-backend`
- **Service**: `housepoints-go-backend:8080`
- **HPA**: Auto-scale 1-10 replicas based on CPU/memory

### Resource Limits (Per Pod)
- **CPU**: 100m request, 1000m limit
- **Memory**: 128Mi request, 512Mi limit

### Scaling Configuration
- Min replicas: 1
- Max replicas: 10
- Scale up: CPU > 70% or Memory > 80%
- Scale down: After 5 min stabilization

## Shared Services

### Database
- **PostgreSQL**: Shared with Python backend
- **Connection**: Via `gamull-backend-secrets`
- **Database**: Uses same `gamull_chores` database

### Redis
- **Service**: `redis.production.svc.cluster.local:6379`
- **DB**: Uses Redis DB 2 (Python uses 0 and 1)

### Secrets
- **Secret Name**: `gamull-backend-secrets`
- Shared with Python backend (same DB credentials)

## Monitoring

### Health Checks
- **Liveness**: `/health` endpoint, 10s interval
- **Readiness**: `/health` endpoint, 5s interval

### Logs
```bash
# View logs
kubectl logs -n production -l app=housepoints-go-backend --tail=100 -f

# View specific pod
kubectl logs -n production housepoints-go-backend-xxxxx-xxxxx -f
```

### Status
```bash
# Check deployment status
kubectl get deployment -n production housepoints-go-backend

# Check pod status
kubectl get pods -n production -l app=housepoints-go-backend

# Check HPA status
kubectl get hpa -n production housepoints-go-backend-hpa
```

## Routing

### Internal (K8s Service)
- Service: `housepoints-go-backend.production.svc.cluster.local:8080`
- Used by other pods in cluster

### External (Ingress)
To expose externally, add Ingress rule:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: housepoints-go
  namespace: production
spec:
  rules:
  - host: api-go.chores.gamull.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: housepoints-go-backend
            port:
              number: 8080
```

## Dual Deployment Strategy

### Phase 1: Parallel Running (Current)
```
┌─────────────────────────────────────┐
│         Nginx/Ingress               │
└───────────┬─────────────────────────┘
            │
    ┌───────┴────────┐
    ↓                ↓
┌────────┐      ┌──────────┐
│Python  │      │ Go       │
│Backend │      │ Backend  │
│:8000   │      │ :8080    │
└────────┘      └──────────┘
    │                │
    └────────┬───────┘
             ↓
      ┌──────────────┐
      │  PostgreSQL  │
      └──────────────┘
```

**Routing**:
- Legacy API: `/api/legacy/*` → Python backend
- New API: `/api/v2/*` → Go backend
- New families onboard to Go
- Existing families stay on Python

### Phase 2: Migration (Months 3-6)
- Gradual family-by-family migration
- Both systems write to same database
- Verification and rollback capability

### Phase 3: Go Only (Month 6+)
- Python backend decommissioned
- All traffic to Go backend
- 80% resource reduction

## Troubleshooting

### Pod Won't Start
```bash
# Check pod events
kubectl describe pod -n production housepoints-go-backend-xxxxx

# Check logs
kubectl logs -n production housepoints-go-backend-xxxxx
```

### Image Pull Errors
```bash
# Verify image exists
docker pull ghcr.io/junoax/housepoints-go:v0.1.0

# Check image pull secret
kubectl get secret -n production ghcr-pull-secret
```

### Database Connection Issues
```bash
# Test from pod
kubectl exec -it -n production housepoints-go-backend-xxxxx -- sh
# Inside pod:
# wget -O- http://localhost:8080/health
```

### Flux Not Syncing
```bash
# Check Flux status
flux get kustomizations

# Force reconciliation
flux reconcile kustomization cluster --with-source
```

## Next Steps

1. ✅ Kubernetes manifests created
2. ⏳ Build and push first Docker image
3. ⏳ Deploy to k3s via Flux
4. ⏳ Verify health checks
5. ⏳ Configure Ingress for external access
6. ⏳ Set up monitoring/alerts
