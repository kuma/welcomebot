# Staging Environment Setup

Complete guide for setting up the staging environment for Discord Bot.

## Prerequisites

- Access to production Kubernetes cluster
- kubectl configured with cluster access
- ArgoCD installed on cluster (see `argocd/README.md`)
- Harbor registry access (or your chosen registry)
- GitHub repository with push access

## Architecture

- **Namespace:** `welcomebot-staging`
- **Auto-deploy:** Yes (on push to `main` branch)
- **Resources:** Medium (512Mi-1Gi RAM)
- **Redis:** Single instance with persistence
- **PostgreSQL:** StatefulSet with 5Gi storage

## Setup Steps

### 1. Create Namespace

```bash
kubectl create namespace welcomebot-staging
```

### 2. Create Secrets

#### a. Bot Secrets

```bash
# Copy example file
cp deployments/overlays/staging/secrets.env.example deployments/overlays/staging/secrets.env

# Edit with your values
vim deployments/overlays/staging/secrets.env
```

Example:
```bash
DISCORD_BOT_TOKEN=your_staging_bot_token
POSTGRES_USER=discord_bot_staging
POSTGRES_PASSWORD=strong_staging_password
POSTGRES_DB=discord_bot_staging
REDIS_PASSWORD=staging_redis_password
```

#### b. Create Kubernetes Secret

```bash
kubectl create secret generic welcomebot-secrets \
  --from-env-file=deployments/overlays/staging/secrets.env \
  -n welcomebot-staging
```

#### c. Create Registry Secret

```bash
kubectl create secret docker-registry harbor-registry \
  --docker-server=harbor.example.com \
  --docker-username=YOUR_USERNAME \
  --docker-password=YOUR_PASSWORD \
  -n welcomebot-staging
```

### 3. Configure GitHub Actions

Add these secrets to GitHub repository (Settings → Secrets → Actions):

- `HARBOR_USERNAME`: Your Harbor username
- `HARBOR_PASSWORD`: Your Harbor password

### 4. Update Registry URLs

Edit `deployments/base/master-deployment.yaml` and `worker-deployment.yaml`:

```yaml
# Change:
image: harbor.example.com/welcomebot/welcomebot-master:latest

# To your registry:
image: your-registry.com/your-org/welcomebot-master:latest
```

Edit `.github/workflows/deploy-staging.yml`:

```yaml
# Change:
REGISTRY: harbor.example.com
IMAGE_BASE: welcomebot

# To your values:
REGISTRY: your-registry.com
IMAGE_BASE: your-org
```

### 5. Configure ArgoCD Application

Edit `deployments/argocd/application-staging.yaml`:

```yaml
# Change:
repoURL: https://github.com/yourusername/welcomebot-template2

# To your repository:
repoURL: https://github.com/your-org/your-repo
```

### 6. Deploy ArgoCD Application

```bash
kubectl apply -f deployments/argocd/application-staging.yaml
```

### 7. Verify Deployment

```bash
# Check ArgoCD application status
kubectl get application welcomebot-staging -n argocd

# Check pods
kubectl get pods -n welcomebot-staging

# View logs
kubectl logs -f deployment/welcomebot-master -n welcomebot-staging
```

## Deployment Flow

1. Developer pushes code to `main` branch
2. GitHub Actions workflow triggers
3. Builds Docker images (amd64 + arm64)
4. Pushes to registry with commit SHA tag
5. Updates `deployments/overlays/staging/kustomization.yaml`
6. Commits manifest change back to repo
7. ArgoCD detects Git change (polls every 3 minutes)
8. ArgoCD syncs new images to staging
9. Pods restart with new version

## Monitoring

### Check Application Status

```bash
# Via kubectl
kubectl get application welcomebot-staging -n argocd

# Via ArgoCD CLI
argocd app get welcomebot-staging

# Via ArgoCD UI
kubectl port-forward svc/argocd-server -n argocd 8080:443
# Visit: https://localhost:8080
```

### View Logs

```bash
# Master logs
kubectl logs -f deployment/welcomebot-master -n welcomebot-staging

# Worker logs
kubectl logs -f deployment/welcomebot-worker -n welcomebot-staging

# All pods
kubectl logs -f -l app=welcomebot -n welcomebot-staging --all-containers=true
```

### Check Pod Status

```bash
kubectl get pods -n welcomebot-staging
kubectl describe pod <pod-name> -n welcomebot-staging
```

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status
kubectl get pods -n welcomebot-staging

# Describe pod for events
kubectl describe pod <pod-name> -n welcomebot-staging

# Check logs
kubectl logs <pod-name> -n welcomebot-staging
```

Common causes:
- Missing or invalid secrets
- Image pull errors (registry credentials)
- Database not ready
- Redis not accessible

### ArgoCD Not Syncing

```bash
# Check sync status
argocd app get welcomebot-staging

# Check if out of sync
argocd app diff welcomebot-staging

# Manual sync
argocd app sync welcomebot-staging

# Force sync
argocd app sync welcomebot-staging --force
```

### Image Pull Errors

```bash
# Check if secret exists
kubectl get secret harbor-registry -n welcomebot-staging

# Recreate registry secret
kubectl delete secret harbor-registry -n welcomebot-staging
kubectl create secret docker-registry harbor-registry \
  --docker-server=harbor.example.com \
  --docker-username=USER \
  --docker-password=PASS \
  -n welcomebot-staging
```

### Database Connection Issues

```bash
# Check PostgreSQL pod
kubectl get pods -l app=postgres -n welcomebot-staging

# View PostgreSQL logs
kubectl logs -l app=postgres -n welcomebot-staging

# Shell into PostgreSQL
kubectl exec -it -l app=postgres -n welcomebot-staging -- psql -U discord_bot_staging -d discord_bot_staging
```

## Maintenance

### Update Secrets

```bash
# Edit secrets file
vim deployments/overlays/staging/secrets.env

# Delete old secret
kubectl delete secret welcomebot-secrets -n welcomebot-staging

# Create new secret
kubectl create secret generic welcomebot-secrets \
  --from-env-file=deployments/overlays/staging/secrets.env \
  -n welcomebot-staging

# Restart deployments
kubectl rollout restart deployment/welcomebot-master -n welcomebot-staging
kubectl rollout restart deployment/welcomebot-worker -n welcomebot-staging
```

### Scale Workers

```bash
# Edit patch file
vim deployments/overlays/staging/patches/worker-patch.yaml
# Change replicas: 4

# Commit and push
git add deployments/overlays/staging/patches/worker-patch.yaml
git commit -m "feat(staging): scale workers to 4"
git push

# ArgoCD will sync automatically
```

### Manual Deployment

If you need to deploy without ArgoCD:

```bash
kubectl apply -k deployments/overlays/staging
```

## Cleanup

```bash
# Delete namespace (removes all resources)
kubectl delete namespace welcomebot-staging

# Delete ArgoCD application
kubectl delete application welcomebot-staging -n argocd
```

## See Also

- [Deployments README](README.md)
- [ArgoCD Guide](argocd/README.md)
- [Production Setup](PRODUCTION_SETUP.md)
- [Deployment Requirements](../requirements/deployment.md)

