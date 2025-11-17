# Deployments

Kubernetes deployment manifests for Discord Bot using Kustomize and ArgoCD.

## Directory Structure

```
deployments/
├── base/                      # Base manifests (shared by all environments)
│   ├── kustomization.yaml     # Base kustomization config
│   ├── master-deployment.yaml # Master bot deployment
│   ├── worker-deployment.yaml # Worker bot deployment
│   ├── postgres.yaml          # PostgreSQL database
│   └── redis-sentinel.yaml    # Redis Sentinel (HA)
│
├── overlays/                  # Environment-specific overlays
│   ├── local/                 # Local development (Docker Desktop)
│   │   ├── kustomization.yaml # Local kustomization
│   │   ├── namespace.yaml     # Local namespace
│   │   ├── redis.yaml         # Simple Redis (no Sentinel)
│   │   ├── secrets.env.example# Example secrets
│   │   └── patches/           # Local-specific patches
│   │       ├── master-patch.yaml
│   │       ├── worker-patch.yaml
│   │       ├── postgres-patch.yaml
│   │       └── redis-patch.yaml
│   │
│   ├── staging/               # Staging environment
│   │   ├── kustomization.yaml # Staging kustomization
│   │   ├── namespace.yaml     # Staging namespace
│   │   ├── redis.yaml         # Redis with persistence
│   │   ├── secrets.env.example# Example secrets
│   │   └── patches/           # Staging-specific patches
│   │       ├── master-patch.yaml
│   │       ├── worker-patch.yaml
│   │       ├── postgres-patch.yaml
│   │       └── redis-patch.yaml
│   │
│   └── production/            # Production environment
│       ├── kustomization.yaml # Production kustomization
│       ├── namespace.yaml     # Production namespace
│       ├── secrets.env.example# Example secrets
│       └── patches/           # Production-specific patches
│           ├── master-patch.yaml
│           ├── worker-patch.yaml
│           ├── postgres-patch.yaml
│           └── redis-sentinel-patch.yaml
│
├── argocd/                    # ArgoCD application manifests
│   ├── application-staging.yaml
│   └── application-production.yaml
│
├── README.md                  # This file
├── STAGING_SETUP.md           # Staging setup guide
└── PRODUCTION_SETUP.md        # Production setup guide
```

## Environments

### Local (Docker Desktop)
- **Namespace:** `welcomebot-local`
- **Cluster:** Docker Desktop
- **Deployment:** Scripts (`scripts/dev-*.sh`)
- **Trigger:** Manual
- **Resources:** Low (256Mi RAM per service)
- **Location:** See `../scripts/README.md`

### Staging
- **Namespace:** `welcomebot-staging`
- **Cluster:** Production K8s cluster (isolated namespace)
- **Deployment:** ArgoCD (GitOps)
- **Trigger:** Push to `main` branch
- **Resources:** Medium (512Mi RAM)
- **Auto-deploy:** Yes

### Production
- **Namespace:** `welcomebot-prod`
- **Cluster:** Production K8s cluster
- **Deployment:** ArgoCD (GitOps)
- **Trigger:** Git tag (`v*`)
- **Resources:** High (1Gi RAM)
- **Auto-deploy:** No (manual sync)

## Quick Start

### Local Development

See `../scripts/README.md` for local development guide.

```bash
# 1. Deploy locally
./scripts/dev-local.sh

# 2. Edit secrets
vim deployments/overlays/local/secrets.env

# 3. Reload
./scripts/dev-reload.sh
```

### Staging Setup

See [STAGING_SETUP.md](STAGING_SETUP.md) for detailed instructions.

```bash
# 1. Update repo URL in ArgoCD manifest
vim deployments/argocd/application-staging.yaml

# 2. Create secrets
cp deployments/overlays/staging/secrets.env.example deployments/overlays/staging/secrets.env
vim deployments/overlays/staging/secrets.env

# 3. Apply to cluster (requires kubectl access)
kubectl create secret generic welcomebot-secrets \
  --from-env-file=deployments/overlays/staging/secrets.env \
  -n welcomebot-staging

kubectl apply -f deployments/argocd/application-staging.yaml
```

### Production Setup

See [PRODUCTION_SETUP.md](PRODUCTION_SETUP.md) for detailed instructions.

## Kustomize

### Build Manifests Locally

```bash
# Build local manifests
kubectl kustomize deployments/overlays/local

# Build staging manifests
kubectl kustomize deployments/overlays/staging

# Build production manifests
kubectl kustomize deployments/overlays/production

# Validate without applying
kubectl kustomize deployments/overlays/staging | kubectl apply --dry-run=client -f -
```

### Apply Manually (Without ArgoCD)

```bash
# Apply local
kubectl apply -k deployments/overlays/local

# Apply staging
kubectl apply -k deployments/overlays/staging

# Apply production
kubectl apply -k deployments/overlays/production
```

## ArgoCD

### Install ArgoCD

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

See [argocd/README.md](argocd/README.md) for detailed setup.

### Deploy Application

```bash
# Apply staging application
kubectl apply -f deployments/argocd/application-staging.yaml

# Apply production application
kubectl apply -f deployments/argocd/application-production.yaml
```

### Check Status

```bash
# Via kubectl
kubectl get application welcomebot-staging -n argocd

# Via ArgoCD CLI
argocd app get welcomebot-staging

# Via UI
kubectl port-forward svc/argocd-server -n argocd 8080:443
# Visit: https://localhost:8080
```

## CI/CD

### GitHub Actions

Workflow files: `.github/workflows/`

**Triggers:**
- Push to `main` branch → Staging deployment
- Push version tag (`v*`) → Production deployment

**Process:**
1. Build multi-arch Docker images
2. Push to Harbor registry with tag
3. Update `kustomization.yaml` with commit SHA/version
4. Commit manifest change
5. ArgoCD detects and syncs (auto for staging, manual for prod)

### Required Secrets

Add to GitHub → Settings → Secrets:
- `HARBOR_USERNAME`
- `HARBOR_PASSWORD`

## Base Manifests

### Components

**master-deployment.yaml**
- Discord bot instance
- Handles Discord events
- Replica: 1 (Discord limitation)

**worker-deployment.yaml**
- Background task processor
- Scales independently
- Default: 2 replicas

**postgres.yaml**
- PostgreSQL 16 database
- StatefulSet with PVC
- Single instance

**redis-sentinel.yaml**
- Redis 7 with Sentinel
- High availability setup
- 3 replicas (production)

## Overlays

### Local Patches

**master-patch.yaml**
- Log level: `debug`
- Log format: `text`
- Resources: 256Mi/250m CPU
- Simple Redis (no Sentinel)

**worker-patch.yaml**
- Log level: `debug`
- Replicas: 1
- Resources: 256Mi/250m CPU

**postgres-patch.yaml**
- Storage: 2Gi
- Resources: 256Mi/200m CPU

**redis-patch.yaml**
- Simple Redis deployment
- No persistence
- Resources: 128Mi/100m CPU

### Staging Patches

**master-patch.yaml**
- Log level: `debug`
- Resources: 512Mi/500m CPU

**worker-patch.yaml**
- Log level: `debug`
- Replicas: 2
- Resources: 256Mi/250m CPU

**postgres-patch.yaml**
- Storage: 5Gi
- Resources: 512Mi/500m CPU

**redis-patch.yaml**
- StatefulSet with persistence
- Storage: 2Gi
- Resources: 256Mi/250m CPU

### Production Patches

**master-patch.yaml**
- Log level: `info`
- Resources: 1Gi/1000m CPU
- SSL for PostgreSQL

**worker-patch.yaml**
- Log level: `info`
- Replicas: 3
- Resources: 512Mi/500m CPU

**postgres-patch.yaml**
- Storage: 20Gi
- Resources: 1Gi/1000m CPU

**redis-sentinel-patch.yaml**
- Redis Sentinel (HA)
- Storage: 10Gi
- Resources: 512Mi/500m CPU

## Secrets Management

### Local Secrets

File: `deployments/overlays/local/secrets.env` (gitignored)

```bash
DISCORD_BOT_TOKEN=dev_bot_token
POSTGRES_USER=discord_bot_dev
POSTGRES_PASSWORD=dev_password
POSTGRES_DB=discord_bot_dev
REDIS_PASSWORD=
```

### Staging/Production Secrets

File: `deployments/overlays/{env}/secrets.env` (gitignored)

```bash
DISCORD_BOT_TOKEN=staging_or_prod_bot_token
POSTGRES_USER=discord_bot_{env}
POSTGRES_PASSWORD=strong_password
POSTGRES_DB=discord_bot_{env}
REDIS_PASSWORD=redis_password
```

### Creating Secrets

```bash
# From env file
kubectl create secret generic welcomebot-secrets \
  --from-env-file=deployments/overlays/staging/secrets.env \
  -n welcomebot-staging

# Registry credentials
kubectl create secret docker-registry harbor-registry \
  --docker-server=harbor.example.com \
  --docker-username=USER \
  --docker-password=PASS \
  -n welcomebot-staging
```

## Maintenance

### Update Image Tags

```bash
cd deployments/overlays/staging

# Edit kustomization.yaml
vim kustomization.yaml
# Change: newTag: new-commit-sha

# Commit and push (ArgoCD will sync)
git add kustomization.yaml
git commit -m "chore(staging): update to new-commit-sha"
git push
```

### Scale Workers

```bash
# Edit worker-patch.yaml
vim deployments/overlays/staging/patches/worker-patch.yaml
# Change: replicas: 4

# Commit and push
git add deployments/overlays/staging/patches/worker-patch.yaml
git commit -m "feat(staging): scale workers to 4"
git push
```

### View Logs

```bash
# Local (via scripts)
./scripts/dev-logs.sh

# Staging/Production (via kubectl)
kubectl logs -f deployment/welcomebot-master -n welcomebot-staging
kubectl logs -f deployment/welcomebot-worker -n welcomebot-staging
```

## Monitoring

### Pod Status

```bash
kubectl get pods -n welcomebot-staging
kubectl describe pod <pod-name> -n welcomebot-staging
```

### Events

```bash
kubectl get events -n welcomebot-staging --sort-by='.lastTimestamp'
```

### Resource Usage

```bash
kubectl top pods -n welcomebot-staging
kubectl top nodes
```

## Troubleshooting

### Quick checks:
```bash
# Pod status
kubectl get pods -n welcomebot-staging

# Pod logs
kubectl logs <pod-name> -n welcomebot-staging

# Events
kubectl get events -n welcomebot-staging

# ArgoCD sync status
kubectl get application welcomebot-staging -n argocd
```

### Common Issues

See environment-specific guides:
- **Local:** `../scripts/README.md#troubleshooting`
- **Staging:** [STAGING_SETUP.md](STAGING_SETUP.md#troubleshooting)
- **Production:** [PRODUCTION_SETUP.md](PRODUCTION_SETUP.md#troubleshooting)

## Documentation

- **Staging Setup:** [STAGING_SETUP.md](STAGING_SETUP.md)
- **Production Setup:** [PRODUCTION_SETUP.md](PRODUCTION_SETUP.md)
- **ArgoCD Guide:** [argocd/README.md](argocd/README.md)
- **Local Development:** `../scripts/README.md`
- **Deployment Requirements:** `../requirements/deployment.md`
- **Quick Start:** `../DEPLOYMENT_QUICKSTART.md`

## Support

- Kubernetes Docs: https://kubernetes.io/docs/
- Kustomize Docs: https://kustomize.io/
- ArgoCD Docs: https://argo-cd.readthedocs.io/
