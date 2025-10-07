# HousePoints Go Deployment Guide

## Architecture Overview

### Multi-Tenant Strategy with housepoints.ai

```
┌─────────────────────────────────────────────────────────┐
│                     Domain Strategy                      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  chores.gamull.com     → Python Backend (legacy)        │
│                          Single family (Gamull)          │
│                                                          │
│  housepoints.ai        → Go Backend (multi-tenant)      │
│  ├─ app.housepoints.ai → Frontend for all families      │
│  └─ api.housepoints.ai → API for all families           │
│                                                          │
│  staging.housepoints.ai → Go Backend (testing)          │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### Environment Strategy

| Environment | Domain | Purpose | Replicas | Resources |
|-------------|--------|---------|----------|-----------|
| **Staging** | staging.housepoints.ai | Migration testing, new features | 1 | 50m CPU, 64Mi RAM |
| **Production** | housepoints.ai | Multi-tenant production | 2-20 (HPA) | 200m CPU, 256Mi RAM |
| **Legacy** | chores.gamull.com | Gamull family only (Python) | 1 | 500m CPU, 512Mi RAM |

## Kubernetes Configuration

### Files in rocky Repository

```
~/projects/rocky/cluster/apps/production/
├── housepoints-go-staging.yaml       # Staging environment
├── housepoints-go-production.yaml    # Production multi-tenant
└── housepoints-go-backend.yaml       # (deprecated - use above)
```

### Resource Allocation

**Staging**:
- CPU: 50m request, 500m limit
- Memory: 64Mi request, 256Mi limit
- Replicas: 1 (no auto-scaling)
- Redis DB: 3

**Production**:
- CPU: 200m request, 2000m limit
- Memory: 256Mi request, 1Gi limit
- Replicas: 2-20 (auto-scales on CPU > 70% or Memory > 80%)
- Redis DB: 4

## Deployment Process

### 1. Build and Push Docker Image

```bash
cd ~/projects/housepoints-go

# Build image
docker build -t ghcr.io/junoax/housepoints-go:v0.1.0 .

# Login to GitHub Container Registry
echo $GITHUB_PAT | docker login ghcr.io -u junoax --password-stdin

# Push image
docker push ghcr.io/junoax/housepoints-go:v0.1.0
```

### 2. Deploy to Staging

```bash
cd ~/projects/rocky

# Update image version in staging manifest
# Edit: cluster/apps/production/housepoints-go-staging.yaml
# Change: image: ghcr.io/junoax/housepoints-go:v0.1.0

# Commit and push
git add cluster/apps/production/housepoints-go-staging.yaml
git commit -m "Deploy housepoints-go v0.1.0 to staging"
git push

# Flux will auto-deploy in 1-5 minutes
```

### 3. Verify Staging Deployment

```bash
# Check pod status
kubectl get pods -n production -l app=housepoints-go,env=staging

# Check logs
kubectl logs -n production -l app=housepoints-go,env=staging --tail=100 -f

# Test health endpoint
curl https://staging.housepoints.ai/health

# Test API
curl https://staging.housepoints.ai/api/version
```

### 4. Deploy to Production (After Staging Verification)

```bash
cd ~/projects/rocky

# Update image version in production manifest
# Edit: cluster/apps/production/housepoints-go-production.yaml
# Change: image: ghcr.io/junoax/housepoints-go:v0.1.0

# Commit and push
git add cluster/apps/production/housepoints-go-production.yaml
git commit -m "Deploy housepoints-go v0.1.0 to production"
git push

# Flux will auto-deploy
```

### 5. Monitor Production Deployment

```bash
# Watch rollout
kubectl rollout status deployment/housepoints-go-production -n production

# Check pod status
kubectl get pods -n production -l app=housepoints-go,env=production

# Check HPA status
kubectl get hpa -n production housepoints-go-production-hpa

# Monitor logs
kubectl logs -n production -l app=housepoints-go,env=production --tail=100 -f
```

## Domain Setup

### DNS Configuration

Configure these A/CNAME records pointing to your k3s ingress IP:

```
# Staging
staging.housepoints.ai          A     <K3S_INGRESS_IP>
api-staging.housepoints.ai      CNAME staging.housepoints.ai

# Production
housepoints.ai                  A     <K3S_INGRESS_IP>
app.housepoints.ai              CNAME housepoints.ai
api.housepoints.ai              CNAME housepoints.ai
www.housepoints.ai              CNAME housepoints.ai
```

### SSL Certificates

Cert-manager automatically provisions Let's Encrypt certificates:

```bash
# Check certificate status
kubectl get certificate -n production | grep housepoints

# Check certificate details
kubectl describe certificate housepoints-ai-staging-tls -n production
kubectl describe certificate housepoints-ai-prod-tls -n production
```

## Migration Strategy

### Phase 1: Staging Testing (Week 1-2)

1. Deploy Go backend to `staging.housepoints.ai`
2. Test with isolated test database
3. Verify all API endpoints
4. Load test with 100 concurrent users

### Phase 2: Production Parallel (Week 3-4)

1. Deploy to `housepoints.ai` with real database
2. New families onboard to Go backend
3. Gamull family stays on `chores.gamull.com` (Python)
4. Monitor metrics and errors

### Phase 3: Dual-Write (Month 2)

1. Enable dual-write mode (write to both systems)
2. Migrate Gamull family data to Go backend
3. Run in parallel for 2 weeks for verification
4. Gradual traffic shift: 10% → 50% → 100%

### Phase 4: Full Migration (Month 3)

1. All traffic to Go backend
2. Python backend in read-only mode for 1 week
3. Decommission Python backend
4. DNS cleanup: redirect `chores.gamull.com` to `app.housepoints.ai`

## Monitoring & Debugging

### Health Checks

```bash
# Staging health
curl https://staging.housepoints.ai/health

# Production health
curl https://api.housepoints.ai/health

# Check specific pod
kubectl exec -it -n production housepoints-go-production-xxxxx -- wget -O- http://localhost:8080/health
```

### Logs

```bash
# Staging logs
kubectl logs -n production -l app=housepoints-go,env=staging --tail=100 -f

# Production logs
kubectl logs -n production -l app=housepoints-go,env=production --tail=100 -f

# Specific pod logs
kubectl logs -n production housepoints-go-production-xxxxx -f

# Previous pod logs (if crashed)
kubectl logs -n production housepoints-go-production-xxxxx --previous
```

### Metrics

```bash
# Pod metrics
kubectl top pods -n production -l app=housepoints-go

# HPA status
kubectl get hpa -n production housepoints-go-production-hpa

# Scaling events
kubectl describe hpa -n production housepoints-go-production-hpa
```

### Database Access

```bash
# Port-forward to PostgreSQL
kubectl port-forward -n production svc/postgresql 5432:5432

# Connect with psql
PGPASSWORD='HP_Sec2025_O0mZVY90R1Yg8L' psql -h localhost -U postgres -d gamull_chores

# Check family data
SELECT * FROM families;
```

## Rollback Procedure

### If Staging Deployment Fails

```bash
# Revert to previous image version in manifest
cd ~/projects/rocky
git revert HEAD
git push

# Or manually scale down
kubectl scale deployment housepoints-go-staging -n production --replicas=0
```

### If Production Deployment Fails

```bash
# Immediate rollback
kubectl rollout undo deployment/housepoints-go-production -n production

# Or revert git commit
cd ~/projects/rocky
git revert HEAD
git push

# Monitor rollback
kubectl rollout status deployment/housepoints-go-production -n production
```

## Performance Tuning

### Scaling Configuration

Edit HPA in `housepoints-go-production.yaml`:

```yaml
spec:
  minReplicas: 2      # Minimum pods
  maxReplicas: 20     # Maximum pods (adjust based on load)
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70  # Scale up when CPU > 70%
```

### Resource Limits

Adjust based on actual usage:

```yaml
resources:
  requests:
    cpu: "200m"        # Guaranteed CPU
    memory: "256Mi"    # Guaranteed memory
  limits:
    cpu: "2000m"       # Maximum CPU
    memory: "1Gi"      # Maximum memory
```

## Troubleshooting

### Pod Won't Start

```bash
# Check events
kubectl describe pod -n production housepoints-go-production-xxxxx

# Common issues:
# - ImagePullBackOff: Check image exists in ghcr.io
# - CrashLoopBackOff: Check application logs
# - Pending: Check resource availability
```

### Database Connection Errors

```bash
# Verify secrets exist
kubectl get secret -n production gamull-backend-secrets

# Check secret values
kubectl get secret -n production gamull-backend-secrets -o jsonpath='{.data.db-host}' | base64 -d

# Test DB connection from pod
kubectl exec -it -n production housepoints-go-production-xxxxx -- sh
# Inside pod: nc -zv $DB_HOST $DB_PORT
```

### Ingress Not Working

```bash
# Check ingress status
kubectl get ingress -n production housepoints-go-staging
kubectl get ingress -n production housepoints-go-production

# Check ingress controller logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx --tail=100 -f

# Test internal service
kubectl run -it --rm debug --image=alpine --restart=Never -- sh
# Inside pod: wget -O- http://housepoints-go-staging.production.svc.cluster.local:8080/health
```

## Next Steps

1. ✅ Kubernetes manifests created for staging and production
2. ⏳ Build and push first Docker image
3. ⏳ Deploy to staging environment
4. ⏳ Configure DNS for housepoints.ai
5. ⏳ Test staging deployment
6. ⏳ Deploy to production
7. ⏳ Monitor and optimize
