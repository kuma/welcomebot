# Deployment Quick Start

Fast-track guide to get your Discord Bot deployed across all environments.

## 📋 Prerequisites Checklist

- [ ] Docker Desktop installed with Kubernetes enabled
- [ ] kubectl installed and configured
- [ ] Go 1.24+ installed
- [ ] Discord bot tokens (dev, staging, prod)
- [ ] GitHub repository access
- [ ] Harbor registry access (or your chosen registry)
- [ ] ArgoCD installed on production cluster (staging/prod only)

## 🚀 10-Minute Local Setup

Get the bot running on your local machine:

```bash
# 1. Enable Kubernetes in Docker Desktop
# Docker Desktop → Settings → Kubernetes → Check "Enable Kubernetes"

# 2. Clone and enter project
cd welcomebot-template2

# 3. Deploy locally
./scripts/dev-local.sh

# 4. Edit secrets (opens automatically on first run)
vim deployments/overlays/local/secrets.env
# Add your dev Discord bot token

# 5. Reload with secrets
./scripts/dev-reload.sh

# 6. View logs
./scripts/dev-logs.sh
# Select option 1 (Master bot live)

# 7. Test in Discord
# Invite bot to test server and try: /ping
```

**Done!** Your bot is now running locally.

**Daily workflow:**
```bash
# Make code changes
vim internal/features/ping/feature.go

# Reload (30-60 seconds)
./scripts/dev-reload.sh

# View logs
./scripts/dev-logs.sh
```

**Cleanup:**
```bash
./scripts/dev-clean.sh
```

## 📦 30-Minute Staging Setup

Deploy to staging environment with auto-deploy on push to main:

### Step 1: Configure Registry

Edit `.github/workflows/deploy-staging.yml`:
```yaml
env:
  REGISTRY: your-registry.com  # Change from harbor.example.com
  IMAGE_BASE: your-org          # Change from welcomebot
```

Edit `deployments/base/*-deployment.yaml` files:
```yaml
# Change all occurrences of:
image: harbor.example.com/welcomebot/...
# To:
image: your-registry.com/your-org/...
```

### Step 2: Add GitHub Secrets

Go to GitHub → Settings → Secrets → Actions → Add:
- `HARBOR_USERNAME`: Your registry username
- `HARBOR_PASSWORD`: Your registry password

### Step 3: Create Secrets

```bash
# Copy example
cp deployments/overlays/staging/secrets.env.example \
   deployments/overlays/staging/secrets.env

# Edit secrets
vim deployments/overlays/staging/secrets.env
# Add staging Discord bot token and strong passwords

# Apply to cluster
kubectl create secret generic welcomebot-secrets \
  --from-env-file=deployments/overlays/staging/secrets.env \
  -n welcomebot-staging

# Create registry secret
kubectl create secret docker-registry harbor-registry \
  --docker-server=your-registry.com \
  --docker-username=YOUR_USERNAME \
  --docker-password=YOUR_PASSWORD \
  -n welcomebot-staging
```

### Step 4: Configure ArgoCD

Edit `deployments/argocd/application-staging.yaml`:
```yaml
spec:
  source:
    repoURL: https://github.com/your-org/your-repo  # Update this
```

Apply ArgoCD application:
```bash
kubectl apply -f deployments/argocd/application-staging.yaml
```

### Step 5: Deploy

```bash
# Commit and push to main
git add .
git commit -m "chore: configure staging deployment"
git push origin main

# GitHub Actions will:
# 1. Build images (~5 minutes)
# 2. Push to registry
# 3. Update manifests
# 4. Commit changes

# ArgoCD will:
# 1. Detect changes (~3 minutes)
# 2. Auto-sync to staging
# 3. Restart pods

# Monitor progress
kubectl get pods -n welcomebot-staging -w
```

**Done!** Staging will now auto-deploy on every push to main.

**Monitor:**
```bash
# Via ArgoCD UI
kubectl port-forward svc/argocd-server -n argocd 8080:443
# Visit: https://localhost:8080

# Via kubectl
kubectl get application welcomebot-staging -n argocd
```

## 🚢 45-Minute Production Setup

Deploy to production with manual approval:

### Step 1: Create Secrets

```bash
# Copy example
cp deployments/overlays/production/secrets.env.example \
   deployments/overlays/production/secrets.env

# Edit secrets
vim deployments/overlays/production/secrets.env
# Add production Discord bot token and VERY strong passwords

# Apply to cluster
kubectl create secret generic welcomebot-secrets \
  --from-env-file=deployments/overlays/production/secrets.env \
  -n welcomebot-prod

# Create registry secret
kubectl create secret docker-registry harbor-registry \
  --docker-server=your-registry.com \
  --docker-username=YOUR_USERNAME \
  --docker-password=YOUR_PASSWORD \
  -n welcomebot-prod
```

### Step 2: Configure ArgoCD

Edit `deployments/argocd/application-production.yaml`:
```yaml
spec:
  source:
    repoURL: https://github.com/your-org/your-repo  # Update this
```

Apply ArgoCD application:
```bash
kubectl apply -f deployments/argocd/application-production.yaml
```

### Step 3: First Production Deployment

```bash
# Create version tag
git tag v1.0.0
git push --tags

# GitHub Actions will:
# 1. Build images (~5 minutes)
# 2. Push to registry with version tag
# 3. Update manifests
# 4. Create GitHub Release with changelog

# Check GitHub Release
# https://github.com/your-org/your-repo/releases

# Review changes in ArgoCD UI
kubectl port-forward svc/argocd-server -n argocd 8080:443
# Visit: https://localhost:8080

# Manually sync production
# Click welcomebot-production → Sync → Synchronize

# Monitor deployment
kubectl get pods -n welcomebot-prod -w

# Verify in production Discord server
```

**Done!** Production is now deployed.

**Future deployments:**
```bash
# 1. Create new version tag
git tag v1.1.0
git push --tags

# 2. Wait for GitHub Actions (~5 minutes)
# 3. Review GitHub Release
# 4. Review in ArgoCD UI
# 5. Manually sync in ArgoCD
# 6. Monitor deployment
```

## 🔄 Daily Workflow

### Development Flow

```
Local Development
       ↓
   Push to main
       ↓
Staging (auto-deploy, ~10 min)
       ↓
   Test staging
       ↓
 Create version tag
       ↓
Production (manual sync)
```

### Commands

```bash
# Local development
./scripts/dev-reload.sh   # Rebuild and restart
./scripts/dev-logs.sh     # View logs
./scripts/dev-shell.sh    # Shell access
./scripts/dev-clean.sh    # Cleanup

# Staging deployment
git push origin main      # Auto-deploys to staging

# Production deployment
git tag v1.2.3           # Create version
git push --tags          # Trigger build
# Then: Manual sync in ArgoCD UI

# Monitoring
kubectl get pods -n welcomebot-staging
kubectl get pods -n welcomebot-prod
kubectl logs -f deployment/welcomebot-master -n welcomebot-{env}
```

## 🛠️ Configuration Checklist

Before deploying, update these configuration files:

### Registry Configuration
- [ ] `.github/workflows/deploy-staging.yml` (REGISTRY, IMAGE_BASE)
- [ ] `.github/workflows/deploy-production.yml` (REGISTRY, IMAGE_BASE)
- [ ] `deployments/base/master-deployment.yaml` (image URLs)
- [ ] `deployments/base/worker-deployment.yaml` (image URLs)

### Repository Configuration
- [ ] `deployments/argocd/application-staging.yaml` (repoURL)
- [ ] `deployments/argocd/application-production.yaml` (repoURL)

### Secrets Configuration
- [ ] `deployments/overlays/local/secrets.env` (dev bot token)
- [ ] `deployments/overlays/staging/secrets.env` (staging bot token)
- [ ] `deployments/overlays/production/secrets.env` (prod bot token)

### GitHub Secrets
- [ ] `HARBOR_USERNAME`
- [ ] `HARBOR_PASSWORD`

## 📊 Verification Commands

### Local
```bash
kubectl get pods -n welcomebot-local
kubectl logs -f deployment/welcomebot-master -n welcomebot-local
```

### Staging
```bash
kubectl get application welcomebot-staging -n argocd
kubectl get pods -n welcomebot-staging
kubectl logs -f deployment/welcomebot-master -n welcomebot-staging
```

### Production
```bash
kubectl get application welcomebot-production -n argocd
kubectl get pods -n welcomebot-prod
kubectl logs -f deployment/welcomebot-master -n welcomebot-prod
```

## 🚨 Troubleshooting Quick Fixes

### Pods Not Starting
```bash
# Check status
kubectl get pods -n welcomebot-{env}

# Check events
kubectl describe pod <pod-name> -n welcomebot-{env}

# Check logs
kubectl logs <pod-name> -n welcomebot-{env}
```

### Image Pull Errors
```bash
# Recreate registry secret
kubectl delete secret harbor-registry -n welcomebot-{env}
kubectl create secret docker-registry harbor-registry \
  --docker-server=your-registry.com \
  --docker-username=USER \
  --docker-password=PASS \
  -n welcomebot-{env}
```

### Bot Not Responding
```bash
# Check if bot token is correct
kubectl get secret welcomebot-secrets -n welcomebot-{env} -o yaml

# Restart master
kubectl rollout restart deployment/welcomebot-master -n welcomebot-{env}

# View logs
kubectl logs -f deployment/welcomebot-master -n welcomebot-{env}
```

### ArgoCD Not Syncing
```bash
# Check sync status
argocd app get welcomebot-{env}

# Force refresh
argocd app refresh welcomebot-{env}

# Manual sync
argocd app sync welcomebot-{env}
```

## 🔗 Next Steps

After completing quickstart:

1. **Read full documentation:**
   - [DEPLOYMENT.md](DEPLOYMENT.md) - Complete deployment guide
   - [requirements/deployment.md](requirements/deployment.md) - Detailed requirements
   
2. **Environment-specific guides:**
   - [scripts/README.md](scripts/README.md) - Local development details
   - [deployments/STAGING_SETUP.md](deployments/STAGING_SETUP.md) - Staging details
   - [deployments/PRODUCTION_SETUP.md](deployments/PRODUCTION_SETUP.md) - Production details
   
3. **Advanced topics:**
   - [deployments/argocd/README.md](deployments/argocd/README.md) - ArgoCD guide
   - Monitoring and alerting
   - Backup and disaster recovery
   - Scaling and performance tuning

## 📚 Additional Resources

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Kustomize Documentation](https://kustomize.io/)
- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Discord Developer Portal](https://discord.com/developers/docs/)

---

**Need help?** Check the troubleshooting sections in:
- [scripts/README.md](scripts/README.md#troubleshooting) - Local issues
- [deployments/STAGING_SETUP.md](deployments/STAGING_SETUP.md#troubleshooting) - Staging issues
- [deployments/PRODUCTION_SETUP.md](deployments/PRODUCTION_SETUP.md#troubleshooting) - Production issues

