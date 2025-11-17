# Deployment Guide

Complete deployment guide for Discord Bot across local, staging, and production environments.

## Quick Links

- 🚀 **Quick Start:** [DEPLOYMENT_QUICKSTART.md](DEPLOYMENT_QUICKSTART.md)
- 📋 **Requirements:** [requirements/deployment.md](requirements/deployment.md)
- 💻 **Local Development:** [scripts/README.md](scripts/README.md)
- 📦 **Deployments:** [deployments/README.md](deployments/README.md)
- 🔧 **Staging Setup:** [deployments/STAGING_SETUP.md](deployments/STAGING_SETUP.md)
- 🚢 **Production Setup:** [deployments/PRODUCTION_SETUP.md](deployments/PRODUCTION_SETUP.md)

## Overview

This project uses a **GitOps** approach with:
- **Kustomize** for environment-specific configuration
- **ArgoCD** for automated deployments
- **GitHub Actions** for CI/CD pipelines
- **Docker Desktop** for local development

## Environments

| Environment | Namespace | Deployment | Auto-Deploy | Resources |
|------------|-----------|------------|-------------|-----------|
| Local | `welcomebot-local` | Scripts | Manual | Low (256Mi) |
| Staging | `welcomebot-staging` | ArgoCD | Yes | Medium (512Mi) |
| Production | `welcomebot-prod` | ArgoCD | No (Manual) | High (1Gi) |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    DEPLOYMENT ARCHITECTURE                   │
└─────────────────────────────────────────────────────────────┘

LOCAL (Docker Desktop)
├─ Docker Desktop Kubernetes
├─ Kustomize (overlays/local)
├─ Scripts (dev-*.sh)
└─ Images: welcomebot-master:local, welcomebot-worker:local

STAGING (Auto-deploy)
├─ Push to main → GitHub Actions
├─ Build images → Harbor registry
├─ Update kustomization.yaml → Git commit
├─ ArgoCD detects change → Auto-sync
└─ Pods restart with new images

PRODUCTION (Manual)
├─ Create tag → GitHub Actions
├─ Build images → Harbor registry
├─ Update kustomization.yaml → Git commit
├─ Create GitHub Release
├─ ArgoCD detects change → Shows "Out of Sync"
├─ Operator reviews → Manual sync
└─ Pods restart with new images
```

## Components

### 1. Kustomize

**Structure:**
- `base/` - Shared manifests (master, worker, postgres, redis)
- `overlays/local/` - Local development (Docker Desktop)
- `overlays/staging/` - Staging environment
- `overlays/production/` - Production environment

**Purpose:** Manages environment-specific configuration without duplication.

### 2. ArgoCD

**Purpose:** GitOps continuous deployment

**Features:**
- Automatic sync (staging)
- Manual sync (production)
- Self-healing
- Rollback support
- Drift detection

### 3. GitHub Actions

**Workflows:**
- `deploy-staging.yml` - Triggers on push to `main`
- `deploy-production.yml` - Triggers on version tags

**Process:**
1. Build Docker images
2. Push to registry
3. Update manifests
4. Commit changes
5. ArgoCD syncs

### 4. Local Scripts

**Scripts:**
- `dev-local.sh` - Initial deployment
- `dev-reload.sh` - Quick rebuild
- `dev-logs.sh` - View logs
- `dev-shell.sh` - Shell access
- `dev-clean.sh` - Cleanup

## Getting Started

### Local Development (10 minutes)

```bash
# 1. Enable Kubernetes in Docker Desktop
# (Docker Desktop → Settings → Kubernetes → Enable)

# 2. Deploy locally
./scripts/dev-local.sh

# 3. Edit secrets file
vim deployments/overlays/local/secrets.env
# Add your Discord bot token

# 4. Reload
./scripts/dev-reload.sh

# 5. View logs
./scripts/dev-logs.sh
```

See: [scripts/README.md](scripts/README.md)

### Staging Deployment (30 minutes)

```bash
# 1. Configure ArgoCD
# Edit deployments/argocd/application-staging.yaml
# Update repoURL to your repository

# 2. Create secrets
cp deployments/overlays/staging/secrets.env.example deployments/overlays/staging/secrets.env
vim deployments/overlays/staging/secrets.env

# 3. Apply secrets to cluster
kubectl create secret generic welcomebot-secrets \
  --from-env-file=deployments/overlays/staging/secrets.env \
  -n welcomebot-staging

# 4. Deploy ArgoCD application
kubectl apply -f deployments/argocd/application-staging.yaml

# 5. Push code to main branch
git push origin main

# 6. Wait for ArgoCD to sync (~5-10 minutes)
```

See: [deployments/STAGING_SETUP.md](deployments/STAGING_SETUP.md)

### Production Deployment (45 minutes)

```bash
# 1. Configure ArgoCD
# Edit deployments/argocd/application-production.yaml
# Update repoURL to your repository

# 2. Create secrets
cp deployments/overlays/production/secrets.env.example deployments/overlays/production/secrets.env
vim deployments/overlays/production/secrets.env

# 3. Apply secrets to cluster
kubectl create secret generic welcomebot-secrets \
  --from-env-file=deployments/overlays/production/secrets.env \
  -n welcomebot-prod

# 4. Deploy ArgoCD application
kubectl apply -f deployments/argocd/application-production.yaml

# 5. Create version tag
git tag v1.0.0
git push --tags

# 6. Review GitHub Release (created automatically)

# 7. Manually sync in ArgoCD UI
```

See: [deployments/PRODUCTION_SETUP.md](deployments/PRODUCTION_SETUP.md)

## Workflow

### Daily Development

```bash
# 1. Make code changes
vim internal/features/ping/feature.go

# 2. Test locally
./scripts/dev-reload.sh

# 3. View logs
./scripts/dev-logs.sh

# 4. Test in Discord
# /ping in your test server

# 5. Commit and push
git add .
git commit -m "feat: improve ping command"
git push origin main

# 6. Staging auto-deploys (~10 minutes)
# 7. Test in staging Discord server
```

### Release to Production

```bash
# 1. Verify staging is stable
# Test all features in staging

# 2. Create version tag
git tag v1.2.3
git push --tags

# 3. GitHub Actions builds images (~5 minutes)
# Creates GitHub Release automatically

# 4. Review changes in ArgoCD UI
kubectl port-forward svc/argocd-server -n argocd 8080:443
# Visit: https://localhost:8080

# 5. Manually sync production
# Click "Sync" button in ArgoCD UI

# 6. Monitor deployment
kubectl get pods -n welcomebot-prod -w

# 7. Verify in production Discord server
```

## Secrets Management

### Structure

Each environment has its own secrets file:
- `deployments/overlays/local/secrets.env`
- `deployments/overlays/staging/secrets.env`
- `deployments/overlays/production/secrets.env`

**⚠️ These files are gitignored and must be created manually.**

### Required Secrets

```bash
# Discord Bot Token
DISCORD_BOT_TOKEN=your_bot_token_here

# PostgreSQL Configuration
POSTGRES_USER=discord_bot_{env}
POSTGRES_PASSWORD=strong_password
POSTGRES_DB=discord_bot_{env}

# Redis Password
REDIS_PASSWORD=redis_password
```

### Creating Secrets

```bash
# Local (handled by script)
./scripts/dev-local.sh  # Creates secrets.env automatically

# Staging/Production
kubectl create secret generic welcomebot-secrets \
  --from-env-file=deployments/overlays/{env}/secrets.env \
  -n welcomebot-{env}
```

## Monitoring

### ArgoCD Status

```bash
# Via UI
kubectl port-forward svc/argocd-server -n argocd 8080:443
# Visit: https://localhost:8080

# Via CLI
argocd app get welcomebot-staging
argocd app get welcomebot-production
```

### Pod Logs

```bash
# Local
./scripts/dev-logs.sh

# Staging
kubectl logs -f deployment/welcomebot-master -n welcomebot-staging

# Production
kubectl logs -f deployment/welcomebot-master -n welcomebot-prod
```

### Pod Status

```bash
# Staging
kubectl get pods -n welcomebot-staging

# Production
kubectl get pods -n welcomebot-prod
```

## Troubleshooting

### Local Issues

See: [scripts/README.md#troubleshooting](scripts/README.md#troubleshooting)

### Staging Issues

See: [deployments/STAGING_SETUP.md#troubleshooting](deployments/STAGING_SETUP.md#troubleshooting)

### Production Issues

See: [deployments/PRODUCTION_SETUP.md#troubleshooting](deployments/PRODUCTION_SETUP.md#troubleshooting)

### Common Issues

#### Pods Not Starting

```bash
# Check pod status
kubectl get pods -n welcomebot-{env}

# Describe pod
kubectl describe pod <pod-name> -n welcomebot-{env}

# View logs
kubectl logs <pod-name> -n welcomebot-{env}
```

#### Image Pull Errors

```bash
# Check if registry secret exists
kubectl get secret harbor-registry -n welcomebot-{env}

# Recreate secret
kubectl create secret docker-registry harbor-registry \
  --docker-server=harbor.example.com \
  --docker-username=USER \
  --docker-password=PASS \
  -n welcomebot-{env}
```

#### Database Connection Errors

```bash
# Check PostgreSQL pod
kubectl get pods -l app=postgres -n welcomebot-{env}

# View PostgreSQL logs
kubectl logs -l app=postgres -n welcomebot-{env}

# Test connection
kubectl exec -it -l app=postgres -n welcomebot-{env} -- \
  psql -U discord_bot_{env} -d discord_bot_{env}
```

## Rollback

### Staging Rollback

```bash
# Via ArgoCD UI
# History → Select previous version → Rollback

# Via CLI
argocd app history welcomebot-staging
argocd app rollback welcomebot-staging <revision-id>
```

### Production Rollback

```bash
# Via ArgoCD UI
# History → Select previous version → Rollback

# Via CLI
argocd app history welcomebot-production
argocd app rollback welcomebot-production <revision-id>

# Monitor rollback
kubectl get pods -n welcomebot-prod -w
```

## Maintenance

### Update Secrets

```bash
# Edit secrets file
vim deployments/overlays/{env}/secrets.env

# Delete old secret
kubectl delete secret welcomebot-secrets -n welcomebot-{env}

# Create new secret
kubectl create secret generic welcomebot-secrets \
  --from-env-file=deployments/overlays/{env}/secrets.env \
  -n welcomebot-{env}

# Restart deployments
kubectl rollout restart deployment/welcomebot-master -n welcomebot-{env}
kubectl rollout restart deployment/welcomebot-worker -n welcomebot-{env}
```

### Scale Workers

```bash
# Edit patch file
vim deployments/overlays/{env}/patches/worker-patch.yaml
# Change: replicas: 5

# Commit and push
git add deployments/overlays/{env}/patches/worker-patch.yaml
git commit -m "feat({env}): scale workers to 5"
git push

# ArgoCD will sync automatically (staging)
# Manual sync required (production)
```

## Best Practices

1. **Always test in local first**
2. **Deploy to staging before production**
3. **Monitor staging for 24 hours before releasing to production**
4. **Use semantic versioning for releases** (v1.2.3)
5. **Review ArgoCD diff before syncing production**
6. **Keep secrets secure** (never commit to Git)
7. **Backup production database regularly**
8. **Document all manual changes**

## Resources

### Documentation
- [DEPLOYMENT_QUICKSTART.md](DEPLOYMENT_QUICKSTART.md) - Quick start guide
- [requirements/deployment.md](requirements/deployment.md) - Requirements
- [scripts/README.md](scripts/README.md) - Local development
- [deployments/README.md](deployments/README.md) - Deployment overview
- [deployments/STAGING_SETUP.md](deployments/STAGING_SETUP.md) - Staging setup
- [deployments/PRODUCTION_SETUP.md](deployments/PRODUCTION_SETUP.md) - Production setup
- [deployments/argocd/README.md](deployments/argocd/README.md) - ArgoCD guide

### External Resources
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Kustomize Documentation](https://kustomize.io/)
- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)

## Support

For issues or questions:
1. Check troubleshooting sections in relevant guides
2. Review ArgoCD application status
3. Check pod logs and events
4. Review GitHub Actions workflow logs

## License

See [LICENSE](LICENSE) for details.
