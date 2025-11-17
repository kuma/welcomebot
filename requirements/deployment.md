# Infrastructure: CI/CD Pipeline and Deployment

## Overview
Automated deployment pipeline for Discord bot template using GitOps (ArgoCD), Kustomize, and GitHub Actions across multiple environments (local, staging, production).

## Purpose
- **Local Development**: Fast iteration with Docker Desktop Kubernetes
- **Staging**: Automated deployment for testing (push to `main` branch)
- **Production**: Manual deployment with safety checks (version tags)

## Environments

### Local (Docker Desktop)
- **Cluster**: Docker Desktop Kubernetes
- **Namespace**: `welcomebot-local`
- **Images**: `bot-master:local`, `bot-worker:local` (no registry)
- **Deployment**: Manual via scripts
- **Purpose**: Development and testing
- **Resources**: Minimal (256Mi RAM per service)

### Staging
- **Cluster**: Production Kubernetes cluster (isolated namespace)
- **Namespace**: `welcomebot-staging`
- **Images**: Harbor registry with commit SHA tags
- **Deployment**: Automated via ArgoCD (GitOps)
- **Trigger**: Push to `main` branch
- **Purpose**: Pre-production testing
- **Resources**: Medium (512Mi RAM per service)

### Production
- **Cluster**: Production Kubernetes cluster
- **Namespace**: `welcomebot-prod`
- **Images**: Harbor registry with version tags (e.g., `v1.0.0`)
- **Deployment**: ArgoCD with manual sync (safety)
- **Trigger**: Git tag (`v*`)
- **Purpose**: Live bot
- **Resources**: High (1Gi RAM per service)

## Architecture Components

### 1. Kustomize Structure
Base + overlay pattern for environment-specific configuration.

```
deployments/
├── base/                        # Shared manifests
│   ├── kustomization.yaml
│   ├── master-deployment.yaml
│   ├── worker-deployment.yaml
│   ├── postgres.yaml
│   └── redis-sentinel.yaml
│
├── overlays/
│   ├── local/                   # Docker Desktop
│   │   ├── kustomization.yaml
│   │   ├── namespace.yaml
│   │   ├── secrets.env.example
│   │   └── patches/
│   │       ├── master-patch.yaml
│   │       └── worker-patch.yaml
│   │
│   ├── staging/                 # Staging environment
│   │   ├── kustomization.yaml
│   │   ├── namespace.yaml
│   │   ├── secrets.env.example
│   │   └── patches/
│   │       ├── master-patch.yaml
│   │       ├── worker-patch.yaml
│   │       └── redis-patch.yaml
│   │
│   └── production/              # Production environment
│       ├── kustomization.yaml
│       ├── namespace.yaml
│       ├── secrets.env.example
│       └── patches/
│           ├── master-patch.yaml
│           ├── worker-patch.yaml
│           └── redis-sentinel-patch.yaml
│
└── argocd/                      # GitOps applications
    ├── application-staging.yaml
    └── application-production.yaml
```

### 2. GitHub Actions Workflows

#### Staging Workflow (`.github/workflows/deploy-staging.yml`)
- **Trigger**: Push to `main` branch
- **Process**:
  1. Build multi-arch Docker images (amd64, arm64)
  2. Push to Harbor registry with tags: `staging`, `{commit-sha}`
  3. Update `deployments/overlays/staging/kustomization.yaml` with commit SHA
  4. Commit manifest change back to Git
  5. ArgoCD detects change and auto-deploys
- **Duration**: ~5-10 minutes (build + deploy)

#### Production Workflow (`.github/workflows/deploy-production.yml`)
- **Trigger**: Push version tag (`v*`)
- **Process**:
  1. Build Docker images (amd64 only for production)
  2. Push to Harbor registry with tags: `{version}`, `latest`
  3. Update `deployments/overlays/production/kustomization.yaml` with version
  4. Generate changelog from Git commits
  5. Create GitHub Release with deployment instructions
  6. ArgoCD detects change (requires manual sync)
- **Duration**: ~5-7 minutes (build) + manual sync

### 3. ArgoCD (GitOps)

#### Staging Application
- **Auto-sync**: Enabled
- **Self-heal**: Enabled (auto-corrects manual changes)
- **Prune**: Enabled (removes resources deleted from Git)
- **Sync Interval**: 3 minutes
- **Source**: `main` branch, `deployments/overlays/staging`

#### Production Application
- **Auto-sync**: Disabled (manual sync required)
- **Self-heal**: Disabled
- **Prune**: Enabled
- **Source**: `main` branch, `deployments/overlays/production`
- **Manual approval**: Required via UI or CLI

### 4. Local Development Scripts

Scripts for fast development iteration on Docker Desktop.

#### `scripts/dev-local.sh`
- Initial deployment to local cluster
- Builds images, creates secrets, applies manifests
- Waits for pods to be ready
- Shows helpful commands

#### `scripts/dev-reload.sh`
- Fast rebuild and redeploy after code changes
- Rebuilds images, restarts deployments
- Typical reload time: 30-60 seconds

#### `scripts/dev-logs.sh`
- Interactive log viewer with menu
- Options: Master/Worker (live or last N lines), All pods, PostgreSQL, Redis

#### `scripts/dev-shell.sh`
- Interactive shell access to pods
- Menu to select pod (Master, Worker, PostgreSQL, Redis)

#### `scripts/dev-clean.sh`
- Clean up local environment
- Deletes entire namespace with confirmation

## Container Images

### Image Strategy
- **Single Dockerfile**: Builds both master and worker binaries
- **Multi-stage build**: Minimizes image size
- **Base image**: Alpine Linux (small, secure)
- **Final size**: ~50-80MB per image

### Image Tags
- **Local**: `bot-master:local`, `bot-worker:local`
- **Staging**: `harbor.example.com/bot/bot-master:23e81a6`
- **Production**: `harbor.example.com/bot/bot-master:v1.0.0`

### Registry
- **Harbor**: Private container registry
- **Authentication**: Via Kubernetes secrets
- **Cache**: Build cache stored in registry for faster builds

## Deployment Flow

### Flow 1: Local Development
```
1. Developer runs: ./scripts/dev-local.sh
2. Script checks Docker Desktop Kubernetes is running
3. Script builds images: bot-master:local, bot-worker:local
4. Script creates secrets.env if missing (requires manual edit)
5. Script applies Kustomize manifests to welcomebot-local namespace
6. Script waits for pods to be ready
7. Developer makes code changes
8. Developer runs: ./scripts/dev-reload.sh
9. Script rebuilds images and restarts deployments
10. Developer tests changes in Discord
11. Developer views logs: ./scripts/dev-logs.sh
```

### Flow 2: Staging Deployment (Automated)
```
1. Developer commits code and pushes to main
2. GitHub Actions workflow triggers
3. Workflow builds multi-arch Docker images
4. Workflow pushes to Harbor: bot-master:23e81a6
5. Workflow updates kustomization.yaml with commit SHA
6. Workflow commits manifest change to Git
7. ArgoCD polls Git repo (every 3 minutes)
8. ArgoCD detects manifest change
9. ArgoCD syncs changes to staging namespace
10. ArgoCD restarts pods with new images
11. Staging bot is now running new code
```

### Flow 3: Production Deployment (Manual)
```
1. Developer creates version tag: git tag v1.0.0
2. Developer pushes tag: git push --tags
3. GitHub Actions workflow triggers
4. Workflow builds Docker images
5. Workflow pushes to Harbor: bot-master:v1.0.0
6. Workflow updates kustomization.yaml with version
7. Workflow generates changelog from Git history
8. Workflow creates GitHub Release with:
   - Version tag
   - Changelog
   - Deployment instructions
   - Rollback instructions
9. ArgoCD detects manifest change (shows "Out of Sync")
10. Operator reviews change in ArgoCD UI
11. Operator clicks "Sync" button (or runs: argocd app sync welcomebot-prod)
12. ArgoCD deploys to production namespace
13. Operator monitors pod status
14. Production bot is now running new version
```

## Data Models

### Kustomization Config
```yaml
# deployments/overlays/staging/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: welcomebot-staging
resources:
  - ../../base
  - namespace.yaml
images:
  - name: harbor.example.com/bot/bot-master
    newTag: 23e81a6  # Updated by CI/CD
  - name: harbor.example.com/bot/bot-worker
    newTag: 23e81a6  # Updated by CI/CD
patches:
  - path: patches/master-patch.yaml
  - path: patches/worker-patch.yaml
```

### ArgoCD Application
```yaml
# deployments/argocd/application-staging.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: welcomebot-staging
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/user/welcomebot-template2
    targetRevision: main
    path: deployments/overlays/staging
  destination:
    server: https://kubernetes.default.svc
    namespace: welcomebot-staging
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

### Secrets Structure
```bash
# deployments/overlays/local/secrets.env
DISCORD_BOT_TOKEN=your_dev_bot_token_here
POSTGRES_USER=bot_dev
POSTGRES_PASSWORD=dev_password
POSTGRES_DB=bot_dev
REDIS_PASSWORD=
```

## Business Logic

### CI/CD Rules
1. **Staging auto-deploys** from `main` branch
2. **Production requires manual sync** (safety)
3. **All secrets are environment-specific** (not in Git)
4. **Image tags are immutable** (no `:latest` in staging/prod)
5. **Local uses local images** (no registry push)

### Deployment Safety
1. **Staging first**: All changes tested in staging
2. **Manual production**: Requires explicit approval
3. **Rollback ready**: ArgoCD keeps deployment history
4. **Health checks**: Pods must be ready before considered deployed
5. **Resource limits**: Prevent resource exhaustion

### Secrets Management
1. **Never commit secrets to Git**
2. **Environment-specific**: Each environment has own secrets
3. **Kubernetes secrets**: Created manually via `kubectl create secret`
4. **Example files**: `.env.example` files show required variables
5. **Local secrets**: Generated automatically with placeholders

## Examples

### Example 1: First Time Local Setup
```bash
# 1. Enable Kubernetes in Docker Desktop
# (Docker Desktop → Settings → Kubernetes → Enable)

# 2. Deploy locally
./scripts/dev-local.sh

# Output:
# 🚀 Starting Local Development Deployment
# ✓ Using Docker Desktop Kubernetes
# 📦 Building Docker images...
#   ✓ bot-master:local built
#   ✓ bot-worker:local built
# ⚠ Warning: secrets.env not found
# Created: deployments/overlays/local/secrets.env
# Please edit secrets.env and add your Discord bot token!

# 3. Edit secrets file
vim deployments/overlays/local/secrets.env
# Add: DISCORD_BOT_TOKEN=your_actual_token

# 4. Deploy again
./scripts/dev-local.sh

# Output:
# ☸️ Applying Kubernetes manifests...
# ⏳ Waiting for pods to be ready...
# ✅ Deployment complete!
```

### Example 2: Daily Development Workflow
```bash
# 1. Make code changes
vim internal/features/ping/feature.go

# 2. Reload (30-60 seconds)
./scripts/dev-reload.sh

# Output:
# 🔄 Reloading Local Development
# 📦 Rebuilding Docker images...
#   ✓ Images rebuilt
# 🔄 Restarting deployments...
# ✅ Reload complete!

# 3. View logs
./scripts/dev-logs.sh
# Select: 1 (Master bot live)

# 4. Test in Discord
# Use test Discord server: /ping

# 5. Shell into pod (if needed)
./scripts/dev-shell.sh
# Select: 1 (Master bot)
# Inside pod: /app/master --version
```

### Example 3: Staging Deployment
```bash
# 1. Commit and push code
git add .
git commit -m "feat(ping): add latency display"
git push origin main

# 2. GitHub Actions runs automatically
# (Check: https://github.com/user/repo/actions)

# 3. After 5-10 minutes, staging is updated
# Verify: kubectl get pods -n welcomebot-staging

# 4. Test in staging Discord server
# Use staging bot token in staging server
```

### Example 4: Production Deployment
```bash
# 1. Create version tag
git tag v1.0.0
git push --tags

# 2. GitHub Actions builds and creates Release
# (Check: https://github.com/user/repo/releases)

# 3. Review ArgoCD UI
# Visit: https://argocd.example.com/applications/welcomebot-prod
# Status: "Out of Sync" (new version available)

# 4. Review diff in ArgoCD
# Click "App Diff" to see changes
# Verify image tag: v1.0.0

# 5. Sync to production
# Via UI: Click "Sync" button
# Via CLI: argocd app sync welcomebot-prod

# 6. Monitor deployment
kubectl get pods -n welcomebot-prod -w

# 7. Verify bot is running
# Test in production Discord server
```

### Example 5: Rollback Production
```bash
# 1. View deployment history
argocd app history welcomebot-prod

# Output:
# ID  DATE                  REVISION
# 10  2024-01-15 10:30:00   v1.0.0 (current)
# 9   2024-01-10 14:20:00   v0.9.5
# 8   2024-01-05 09:15:00   v0.9.4

# 2. Rollback to previous version
argocd app rollback welcomebot-prod 9

# 3. Monitor rollback
kubectl get pods -n welcomebot-prod -w

# 4. Verify bot is running v0.9.5
# Test in production Discord server
```

### Example 6: Cleanup Local Environment
```bash
# Clean up everything
./scripts/dev-clean.sh

# Output:
# 🧹 Cleaning Up Local Development
# Current resources in welcomebot-local:
# [Shows all pods, services, etc.]
# ⚠️ This will DELETE all resources!
# Are you sure? (y/N): y
# 🗑️ Deleting namespace...
# ✅ Cleanup complete!

# To redeploy:
# ./scripts/dev-local.sh
```

## Technical Requirements

### Prerequisites
- **Docker Desktop**: With Kubernetes enabled (local only)
- **kubectl**: Kubernetes CLI (`brew install kubectl`)
- **ArgoCD**: Installed on production cluster (staging/prod only)
- **Harbor Registry**: Access credentials configured (staging/prod only)
- **GitHub Secrets**: Harbor credentials configured in repo (CI/CD only)
- **Go 1.24+**: For local development

### Kubernetes Cluster Requirements
- **Local**: Docker Desktop Kubernetes (1 node)
- **Staging/Production**: Kubernetes 1.28+ cluster with ArgoCD

### Repository Structure
- **Single repository**: Code + manifests in same repo (GitOps)
- **Protected branches**: `main` branch requires PR approval
- **Tag protection**: Version tags immutable

### Security Requirements
- **Secrets never in Git**: All secrets created manually
- **Registry authentication**: Via Kubernetes image pull secrets
- **ArgoCD RBAC**: Separate access for staging/production
- **GitHub Actions secrets**: Harbor credentials stored securely

### Performance Requirements
- **Local reload**: < 60 seconds
- **Staging deployment**: < 10 minutes (build + deploy)
- **Production deployment**: < 5 minutes (manual sync only)
- **Image build cache**: Reduces build time by 50%

### Monitoring Requirements
- **ArgoCD UI**: Visual deployment status
- **Kubernetes events**: Pod status, errors, restarts
- **Pod logs**: Accessible via kubectl or scripts
- **GitHub Actions logs**: Build and deployment logs

## Configuration

### Environment Variables (Per Environment)

#### Local
```bash
DISCORD_BOT_TOKEN=dev_bot_token
POSTGRES_HOST=postgres-service
POSTGRES_PORT=5432
POSTGRES_USER=bot_dev
POSTGRES_PASSWORD=dev_password
POSTGRES_DB=bot_dev
POSTGRES_SSLMODE=disable
REDIS_ADDR=redis-service:6379
REDIS_PASSWORD=
LOG_LEVEL=debug
LOG_FORMAT=text
```

#### Staging
```bash
DISCORD_BOT_TOKEN=staging_bot_token
POSTGRES_HOST=postgres-service
POSTGRES_PORT=5432
POSTGRES_USER=bot_staging
POSTGRES_PASSWORD=strong_password
POSTGRES_DB=bot_staging
POSTGRES_SSLMODE=disable
REDIS_SENTINEL_ADDRS=redis-sentinel:26379
REDIS_MASTER_NAME=bot-master
REDIS_PASSWORD=redis_password
LOG_LEVEL=debug
LOG_FORMAT=json
```

#### Production
```bash
DISCORD_BOT_TOKEN=prod_bot_token
POSTGRES_HOST=postgres-service
POSTGRES_PORT=5432
POSTGRES_USER=bot_prod
POSTGRES_PASSWORD=very_strong_password
POSTGRES_DB=bot_prod
POSTGRES_SSLMODE=require
REDIS_SENTINEL_ADDRS=redis-sentinel:26379
REDIS_MASTER_NAME=bot-master
REDIS_PASSWORD=strong_redis_password
LOG_LEVEL=info
LOG_FORMAT=json
```

### Resource Limits

#### Local (Docker Desktop)
```yaml
master:
  requests: { memory: 256Mi, cpu: 250m }
  limits: { memory: 512Mi, cpu: 500m }
worker:
  requests: { memory: 256Mi, cpu: 250m }
  limits: { memory: 512Mi, cpu: 500m }
postgres:
  requests: { memory: 256Mi, cpu: 200m }
redis:
  requests: { memory: 128Mi, cpu: 100m }
```

#### Staging
```yaml
master:
  requests: { memory: 512Mi, cpu: 500m }
  limits: { memory: 1Gi, cpu: 1000m }
worker:
  requests: { memory: 256Mi, cpu: 250m }
  limits: { memory: 512Mi, cpu: 500m }
postgres:
  requests: { memory: 512Mi, cpu: 500m }
redis:
  requests: { memory: 256Mi, cpu: 250m }
```

#### Production
```yaml
master:
  requests: { memory: 1Gi, cpu: 1000m }
  limits: { memory: 2Gi, cpu: 2000m }
worker:
  requests: { memory: 512Mi, cpu: 500m }
  limits: { memory: 1Gi, cpu: 1000m }
postgres:
  requests: { memory: 1Gi, cpu: 1000m }
redis-sentinel:
  requests: { memory: 512Mi, cpu: 500m }
```

## Documentation to Create

### Top-Level Documentation
- `DEPLOYMENT.md`: Overview of deployment architecture
- `DEPLOYMENT_QUICKSTART.md`: Quick start guide for all environments

### Deployment Documentation
- `deployments/README.md`: Overview of deployment structure
- `deployments/argocd/README.md`: ArgoCD setup and usage
- `deployments/overlays/local/README.md`: Local development guide
- `deployments/STAGING_SETUP.md`: Staging environment setup
- `deployments/PRODUCTION_SETUP.md`: Production environment setup

### Scripts Documentation
- `scripts/README.md`: Development scripts overview

## Success Criteria

### Local Development
- [ ] First-time setup works without issues
- [ ] Reload time < 60 seconds
- [ ] Clear error messages when something fails
- [ ] Easy to view logs and debug
- [ ] Clean up removes all resources

### Staging Deployment
- [ ] Auto-deploys within 10 minutes of push to main
- [ ] ArgoCD shows healthy status
- [ ] Pods restart with new images
- [ ] Staging bot responds to commands
- [ ] Build cache reduces subsequent build times

### Production Deployment
- [ ] Manual sync works reliably
- [ ] Rollback works in < 2 minutes
- [ ] Clear deployment instructions in GitHub Release
- [ ] ArgoCD UI shows deployment progress
- [ ] Production bot has zero downtime

### Documentation
- [ ] New developer can set up local environment in < 15 minutes
- [ ] All commands are documented with examples
- [ ] Troubleshooting guides cover common issues
- [ ] Architecture diagrams show deployment flow

## Future Enhancements
- **Multi-region production**: Deploy to multiple regions for redundancy
- **Canary deployments**: Gradual rollout in production
- **Automated testing**: Run tests before deployment
- **Monitoring integration**: Prometheus + Grafana
- **Slack notifications**: Deployment success/failure alerts
- **Database migrations**: Automated schema migrations in CI/CD
- **Helm charts**: Alternative to Kustomize for more complex scenarios

## References
- **Kustomize Documentation**: https://kustomize.io/
- **ArgoCD Documentation**: https://argo-cd.readthedocs.io/
- **GitHub Actions Documentation**: https://docs.github.com/en/actions
- **Harbor Documentation**: https://goharbor.io/docs/
- **Kubernetes Documentation**: https://kubernetes.io/docs/

