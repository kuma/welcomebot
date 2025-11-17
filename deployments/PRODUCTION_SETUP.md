# Production Environment Setup

Complete guide for setting up the production environment for Discord Bot.

## Prerequisites

- Access to production Kubernetes cluster
- kubectl configured with cluster access
- ArgoCD installed on cluster (see `argocd/README.md`)
- Harbor registry access (or your chosen registry)
- GitHub repository with push access
- Staging environment tested and validated

## Architecture

- **Namespace:** `welcomebot-prod`
- **Auto-deploy:** No (manual sync required)
- **Resources:** High (1Gi-2Gi RAM)
- **Redis:** Sentinel (High Availability, 3 replicas)
- **PostgreSQL:** StatefulSet with 20Gi storage

## Setup Steps

### 1. Create Namespace

```bash
kubectl create namespace welcomebot-prod
```

### 2. Create Secrets

#### a. Bot Secrets

```bash
# Copy example file
cp deployments/overlays/production/secrets.env.example deployments/overlays/production/secrets.env

# Edit with your values
vim deployments/overlays/production/secrets.env
```

Example:
```bash
DISCORD_BOT_TOKEN=your_production_bot_token
POSTGRES_USER=discord_bot_prod
POSTGRES_PASSWORD=very_strong_production_password
POSTGRES_DB=discord_bot_prod
REDIS_PASSWORD=strong_production_redis_password
```

#### b. Create Kubernetes Secret

```bash
kubectl create secret generic welcomebot-secrets \
  --from-env-file=deployments/overlays/production/secrets.env \
  -n welcomebot-prod
```

#### c. Create Registry Secret

```bash
kubectl create secret docker-registry harbor-registry \
  --docker-server=harbor.example.com \
  --docker-username=YOUR_USERNAME \
  --docker-password=YOUR_PASSWORD \
  -n welcomebot-prod
```

### 3. Configure ArgoCD Application

Edit `deployments/argocd/application-production.yaml`:

```yaml
# Change:
repoURL: https://github.com/yourusername/welcomebot-template2

# To your repository:
repoURL: https://github.com/your-org/your-repo
```

### 4. Deploy ArgoCD Application

```bash
kubectl apply -f deployments/argocd/application-production.yaml
```

### 5. Initial Deployment

For first deployment, you need to manually sync:

```bash
# Via ArgoCD CLI
argocd app sync welcomebot-production

# Or via UI
kubectl port-forward svc/argocd-server -n argocd 8080:443
# Visit: https://localhost:8080
# Click on welcomebot-production → Sync
```

### 6. Verify Deployment

```bash
# Check ArgoCD application status
kubectl get application welcomebot-production -n argocd

# Check pods
kubectl get pods -n welcomebot-prod

# View logs
kubectl logs -f deployment/welcomebot-master -n welcomebot-prod
```

## Deployment Flow

1. Developer creates version tag: `git tag v1.0.0 && git push --tags`
2. GitHub Actions workflow triggers
3. Builds Docker images (amd64 only)
4. Pushes to registry with version tag
5. Updates `deployments/overlays/production/kustomization.yaml`
6. Creates GitHub Release with changelog
7. Commits manifest change back to repo
8. ArgoCD detects Git change (shows "Out of Sync")
9. **Operator reviews change in ArgoCD UI**
10. **Operator manually clicks "Sync" button**
11. ArgoCD deploys to production
12. Pods restart with new version

## Production Deployment Process

### Step-by-Step Guide

#### 1. Prepare Release

```bash
# Ensure staging is working
# Test all features in staging environment

# Create version tag
git tag v1.0.0
git push --tags
```

#### 2. Wait for Build

```bash
# Check GitHub Actions
# https://github.com/your-org/your-repo/actions

# Wait for workflow to complete (~5-7 minutes)
# Verify GitHub Release is created
```

#### 3. Review Changes

```bash
# Via ArgoCD UI
kubectl port-forward svc/argocd-server -n argocd 8080:443
# Visit: https://localhost:8080

# Or via CLI
argocd app diff welcomebot-production
```

#### 4. Deploy to Production

```bash
# Via ArgoCD UI
# Click welcomebot-production → Sync → Synchronize

# Or via CLI
argocd app sync welcomebot-production
```

#### 5. Monitor Deployment

```bash
# Watch pods restart
kubectl get pods -n welcomebot-prod -w

# Check logs
kubectl logs -f deployment/welcomebot-master -n welcomebot-prod

# Verify bot responds to commands in Discord
```

#### 6. Rollback (if needed)

```bash
# Via ArgoCD UI
# History → Select previous version → Rollback

# Or via CLI
argocd app history welcomebot-production
argocd app rollback welcomebot-production <revision-id>
```

## Monitoring

### Check Application Status

```bash
# Via kubectl
kubectl get application welcomebot-production -n argocd

# Via ArgoCD CLI
argocd app get welcomebot-production

# Via ArgoCD UI
kubectl port-forward svc/argocd-server -n argocd 8080:443
# Visit: https://localhost:8080
```

### View Logs

```bash
# Master logs
kubectl logs -f deployment/welcomebot-master -n welcomebot-prod

# Worker logs
kubectl logs -f deployment/welcomebot-worker -n welcomebot-prod

# All pods
kubectl logs -f -l app=welcomebot -n welcomebot-prod --all-containers=true
```

### Resource Usage

```bash
# Pod resource usage
kubectl top pods -n welcomebot-prod

# Node resource usage
kubectl top nodes
```

## Troubleshooting

### Deployment Failed

```bash
# Check pod status
kubectl get pods -n welcomebot-prod

# Describe pod for events
kubectl describe pod <pod-name> -n welcomebot-prod

# Check logs
kubectl logs <pod-name> -n welcomebot-prod

# View ArgoCD sync status
argocd app get welcomebot-production
```

### Rollback Procedure

```bash
# 1. View deployment history
argocd app history welcomebot-production

# 2. Identify last working version
# 3. Rollback to that version
argocd app rollback welcomebot-production <revision-id>

# 4. Monitor rollback
kubectl get pods -n welcomebot-prod -w

# 5. Verify bot is working
# Test commands in Discord
```

### Redis Sentinel Issues

```bash
# Check Redis pods
kubectl get pods -l app=redis-sentinel -n welcomebot-prod

# Check sentinel status
kubectl exec -it redis-0 -n welcomebot-prod -- redis-cli -p 26379 sentinel master welcomebot-master

# View Redis logs
kubectl logs -f redis-0 -n welcomebot-prod -c redis
kubectl logs -f redis-0 -n welcomebot-prod -c sentinel
```

## Maintenance

### Update Secrets

```bash
# Edit secrets file
vim deployments/overlays/production/secrets.env

# Delete old secret
kubectl delete secret welcomebot-secrets -n welcomebot-prod

# Create new secret
kubectl create secret generic welcomebot-secrets \
  --from-env-file=deployments/overlays/production/secrets.env \
  -n welcomebot-prod

# Restart deployments
kubectl rollout restart deployment/welcomebot-master -n welcomebot-prod
kubectl rollout restart deployment/welcomebot-worker -n welcomebot-prod
```

### Scale Workers

```bash
# Edit patch file
vim deployments/overlays/production/patches/worker-patch.yaml
# Change replicas: 5

# Commit and push
git add deployments/overlays/production/patches/worker-patch.yaml
git commit -m "feat(prod): scale workers to 5"
git push

# Manually sync in ArgoCD
argocd app sync welcomebot-production
```

### Database Backup

```bash
# Create backup
kubectl exec -it statefulset/postgres -n welcomebot-prod -- \
  pg_dump -U discord_bot_prod discord_bot_prod > backup-$(date +%Y%m%d).sql

# Restore backup
kubectl exec -i statefulset/postgres -n welcomebot-prod -- \
  psql -U discord_bot_prod discord_bot_prod < backup-20240115.sql
```

## Security Considerations

- ✅ Use strong, unique passwords
- ✅ Enable PostgreSQL SSL (`POSTGRES_SSLMODE=require`)
- ✅ Redis password protected
- ✅ Registry credentials secured
- ✅ ArgoCD RBAC configured
- ✅ Network policies (future)
- ✅ Pod security policies (future)

## Disaster Recovery

### Full Recovery Process

```bash
# 1. Restore namespace
kubectl create namespace welcomebot-prod

# 2. Restore secrets
kubectl create secret generic welcomebot-secrets \
  --from-env-file=deployments/overlays/production/secrets.env \
  -n welcomebot-prod

kubectl create secret docker-registry harbor-registry \
  --docker-server=harbor.example.com \
  --docker-username=USER \
  --docker-password=PASS \
  -n welcomebot-prod

# 3. Restore database from backup
# (copy backup to pod, restore with psql)

# 4. Deploy via ArgoCD
kubectl apply -f deployments/argocd/application-production.yaml
argocd app sync welcomebot-production

# 5. Verify deployment
kubectl get pods -n welcomebot-prod
```

## See Also

- [Deployments README](README.md)
- [ArgoCD Guide](argocd/README.md)
- [Staging Setup](STAGING_SETUP.md)
- [Deployment Requirements](../requirements/deployment.md)

