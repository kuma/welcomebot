# Development and Production Scripts

Scripts for Discord Bot deployment on Kubernetes.

This directory contains two sets of scripts:
- **Local Development:** `dev-*.sh` - For local development on Docker Desktop
- **Lightweight Production:** `prod-*.sh` - For production deployment with Harbor registry

## Prerequisites

- Docker Desktop with Kubernetes enabled
- kubectl configured
- Go 1.24+ (for development)

## Setup Local Cluster

### Docker Desktop Kubernetes

1. Open Docker Desktop
2. Go to Settings → Kubernetes
3. Check "Enable Kubernetes"
4. Click "Apply & Restart"

The scripts will automatically:
- Detect Docker Desktop Kubernetes
- Switch to `docker-desktop` context
- Use local Docker images (no image loading needed)

## Scripts

### 🚀 dev-local.sh

**Purpose:** Initial deployment to local cluster

**What it does:**
1. Checks if Docker Desktop Kubernetes is running
2. Switches to `docker-desktop` kubectl context
3. Builds Docker images (`welcomebot-master:local`, `welcomebot-worker:local`)
4. Images are automatically available (Docker Desktop uses local daemon)
5. Creates `secrets.env` if it doesn't exist
6. Applies Kubernetes manifests via kustomize
7. Waits for pods to be ready
8. Shows useful commands and status

**Usage:**
```bash
./scripts/dev-local.sh
```

**First run:** Will create `deployments/overlays/local/secrets.env` - you need to edit it with your Discord bot token.

---

### 🔄 dev-reload.sh

**Purpose:** Quick rebuild and redeploy after code changes

**What it does:**
1. Rebuilds Docker images
2. Loads images into cluster
3. Restarts deployments (kubectl rollout restart)
4. Waits for pods to be ready

**Usage:**
```bash
# After making code changes
./scripts/dev-reload.sh
```

**Typical workflow:**
```bash
# Edit code
vim internal/features/ping/feature.go

# Reload
./scripts/dev-reload.sh

# Check logs
./scripts/dev-logs.sh
```

---

### 📋 dev-logs.sh

**Purpose:** View logs from various pods

**What it does:**
- Interactive menu to select which logs to view
- Options for master, worker, PostgreSQL, Redis
- Live following or last N lines

**Usage:**
```bash
./scripts/dev-logs.sh

# Then select:
# 1) Master bot (live)
# 2) Worker bot (live)
# 3) All pods (live)
# 4) Master bot (last 100 lines)
# 5) Worker bot (last 100 lines)
# 6) PostgreSQL
# 7) Redis
```

---

### 🐚 dev-shell.sh

**Purpose:** Open an interactive shell in a pod

**What it does:**
- Interactive menu to select which pod to shell into
- Opens shell for debugging, inspection, manual testing

**Usage:**
```bash
./scripts/dev-shell.sh

# Then select:
# 1) Master bot
# 2) Worker bot
# 3) PostgreSQL
# 4) Redis
```

**Example use cases:**
```bash
# In master pod:
./scripts/dev-shell.sh
# Select 1
# Inside pod:
/app/master --version
ls -la /app/internal/core/i18n/translations/

# In PostgreSQL:
./scripts/dev-shell.sh
# Select 3
# Inside pod:
psql -U discord_bot_dev -d discord_bot_dev
\dt  # List tables
SELECT * FROM guild_languages LIMIT 5;
```

---

### 🧹 dev-clean.sh

**Purpose:** Clean up local development environment

**What it does:**
1. Shows current resources in `welcomebot-local` namespace
2. Asks for confirmation
3. Deletes the entire namespace

**Usage:**
```bash
./scripts/dev-clean.sh
```

**⚠️ Warning:** This deletes everything! Use when:
- Switching branches
- Testing clean deployment
- Freeing up resources

---

## Typical Development Workflow

### Day 1: Initial Setup

```bash
# 1. Enable Kubernetes in Docker Desktop
# (Docker Desktop → Settings → Kubernetes → Enable)

# 2. Deploy
./scripts/dev-local.sh

# 3. Edit secrets file (first time only)
vim deployments/overlays/local/secrets.env
# Add your Discord bot token

# 4. Redeploy with secrets
./scripts/dev-reload.sh

# 5. Watch logs
./scripts/dev-logs.sh
# Select: 1 (Master bot live)
```

### Daily Development

```bash
# 1. Make code changes
vim internal/features/ping/feature.go

# 2. Reload
./scripts/dev-reload.sh

# 3. Test in Discord
# /ping in your test Discord server

# 4. Check logs if needed
./scripts/dev-logs.sh
```

### Debugging

```bash
# View recent logs
./scripts/dev-logs.sh
# Select: 4 (Master last 100 lines)

# Shell into pod
./scripts/dev-shell.sh
# Select: 1 (Master bot)
# Inside:
/app/master --help
env | grep DISCORD

# Check database
./scripts/dev-shell.sh
# Select: 3 (PostgreSQL)
# Inside:
psql -U discord_bot_dev -d discord_bot_dev
\dt
SELECT * FROM guild_languages;
```

### Cleanup

```bash
# Clean up everything
./scripts/dev-clean.sh

# Docker Desktop Kubernetes stays running (no need to stop)
# If you want to disable: Docker Desktop → Settings → Kubernetes → Disable
```

---

## Troubleshooting

### Script fails: "No Kubernetes cluster detected"

**Solution:**
1. Open Docker Desktop
2. Go to Settings → Kubernetes
3. Enable "Enable Kubernetes"
4. Wait for it to start (green indicator)
5. Verify:
```bash
kubectl config use-context docker-desktop
kubectl cluster-info
```

### Images not available

**Docker Desktop uses local Docker daemon, so images should be automatically available.**

Check if images exist:
```bash
docker images | grep welcomebot
```

If missing, rebuild:
```bash
docker build -t welcomebot-master:local .
docker build -t welcomebot-worker:local .
```

### Pods stuck in ImagePullBackOff

**Check:**
```bash
kubectl describe pod -n welcomebot-local <pod-name>
```

**Solution:**
```bash
# Rebuild and reload
./scripts/dev-reload.sh
```

### Master pod crashes immediately

**Check logs:**
```bash
./scripts/dev-logs.sh
# Select: 4 (Master last 100 lines)
```

**Common issues:**
1. Missing or invalid Discord bot token
2. Database not ready
3. Redis not accessible

**Fix secrets:**
```bash
vim deployments/overlays/local/secrets.env
# Update DISCORD_BOT_TOKEN

# Recreate secret
kubectl delete secret welcomebot-secrets -n welcomebot-local
kubectl create secret generic welcomebot-secrets \
  --from-env-file=deployments/overlays/local/secrets.env \
  -n welcomebot-local

# Restart
./scripts/dev-reload.sh
```

### PostgreSQL issues

**Check if running:**
```bash
kubectl get pods -n welcomebot-local -l app=postgres
```

**Access database:**
```bash
./scripts/dev-shell.sh
# Select: 3 (PostgreSQL)
psql -U discord_bot_dev -d discord_bot_dev
```

**Reset database:**
```bash
# Delete and recreate
kubectl delete pod -n welcomebot-local -l app=postgres
# Wait for new pod to start
kubectl wait --for=condition=ready pod -l app=postgres -n welcomebot-local --timeout=60s
```

---

## Advanced Usage

### Custom kubectl commands

```bash
# View all resources
kubectl get all -n welcomebot-local

# Describe deployment
kubectl describe deployment welcomebot-master -n welcomebot-local

# View events
kubectl get events -n welcomebot-local --sort-by='.lastTimestamp'

# Scale worker
kubectl scale deployment welcomebot-worker -n welcomebot-local --replicas=2

# Port forward (if you add HTTP endpoints)
kubectl port-forward -n welcomebot-local deployment/welcomebot-master 8080:8080
```

### Direct log access

```bash
# Master logs (last 100 lines)
kubectl logs deployment/welcomebot-master -n welcomebot-local --tail=100

# Follow master logs
kubectl logs -f deployment/welcomebot-master -n welcomebot-local

# Worker logs
kubectl logs -f deployment/welcomebot-worker -n welcomebot-local

# All containers
kubectl logs -f -n welcomebot-local --all-containers=true -l app=welcomebot
```

### Manual image management

```bash
# Build without script
docker build -t welcomebot-master:local .
docker build -t welcomebot-worker:local .

# Verify images
docker images | grep welcomebot

# Docker Desktop automatically makes local images available to Kubernetes
# No need to manually load images
```

---

## Environment Variables

All environment variables are set in `deployments/overlays/local/secrets.env`:

```bash
# Required
DISCORD_BOT_TOKEN=your_dev_bot_token_here

# Database (defaults work for local)
POSTGRES_USER=discord_bot_dev
POSTGRES_PASSWORD=dev_password
POSTGRES_DB=discord_bot_dev

# Redis (optional)
REDIS_PASSWORD=
```

---

## Performance Tips

### Faster image builds

```bash
# Use Docker build cache
docker build -t welcomebot-master:local .

# Clear cache if needed
docker builder prune
```

### Faster reload

The `dev-reload.sh` script only rebuilds images and restarts pods (no full redeploy).

Typical reload time: **30-60 seconds**

### Resource usage

Local deployment uses minimal resources:
- Master: 256Mi RAM, 250m CPU
- Worker: 256Mi RAM, 250m CPU
- PostgreSQL: 256Mi RAM
- Redis: 128Mi RAM

**Total: ~900Mi RAM**

---

---

# Production Scripts (Lightweight)

Production deployment scripts using Harbor registry. These repurpose the local dev workflow for production use.

## Prerequisites

- Kubernetes cluster access (kubectl configured)
- Harbor registry access
- Docker installed locally
- NFS CSI storage class available

**See:** `../deployments/LIGHTPROD_SETUP.md` for complete setup guide.

## Production Scripts

### 🚀 prod-deploy.sh

**Purpose:** Initial deployment to production cluster

**What it does:**
1. Logs in to Harbor registry
2. Builds Docker images locally
3. Pushes images to Harbor with `lightprod` tag
4. Creates Harbor registry secret in Kubernetes
5. Applies Kubernetes manifests (namespace: `welcomebot-lightprod`)
6. Waits for pods to be ready

**Usage:**
```bash
# Deploy (Harbor credentials pre-configured in script)
./scripts/prod-deploy.sh
```

**First run:** Takes 3-5 minutes (builds images, pushes to Harbor, deploys)

**Configuration needed:**
1. `deployments/overlays/lightprod/secrets.env` - Discord tokens, DB credentials
2. That's it! Harbor credentials are already configured in the script

---

### 🔄 prod-reload.sh

**Purpose:** Quick rebuild and redeploy after code changes

**What it does:**
1. Rebuilds Docker images
2. Pushes to Harbor
3. Restarts deployments (kubectl rollout restart)
4. Waits for rollouts to complete

**Usage:**
```bash
# After making code changes
./scripts/prod-reload.sh
```

**Typical workflow:**
```bash
# Edit code
vim internal/features/ping/feature.go

# Reload production
./scripts/prod-reload.sh

# Check logs
./scripts/prod-logs.sh
```

**Reload time:** 2-4 minutes (build + push + restart)

---

### 📋 prod-logs.sh

**Purpose:** View logs from production pods

**What it does:**
- Interactive menu to select which logs to view
- Options for master, workers (3 slaves), PostgreSQL, Redis
- Live following or last N lines

**Usage:**
```bash
./scripts/prod-logs.sh

# Then select:
#  1) Master bot (live - follow)
#  2) Worker slave 1 (live - follow)
#  3) Worker slave 2 (live - follow)
#  4) Worker slave 3 (live - follow)
#  5) All bot pods (live - follow)
#  6-9) Last 100 lines of each
# 10-13) PostgreSQL/Redis logs
# 14) All pods status
```

---

### 🐚 prod-shell.sh

**Purpose:** Open an interactive shell in a production pod

**What it does:**
- Interactive menu to select which pod to shell into
- Opens shell for debugging, inspection, manual operations

**Usage:**
```bash
./scripts/prod-shell.sh

# Then select:
# 1) Master bot
# 2) Worker slave 1
# 3) Worker slave 2
# 4) Worker slave 3
# 5) PostgreSQL
# 6) Redis
```

**Example use cases:**
```bash
# In master pod:
./scripts/prod-shell.sh
# Select 1
# Inside pod:
/app/master --version

# In PostgreSQL:
./scripts/prod-shell.sh
# Select 5
# Inside pod:
psql -U welcomebot_prod -d welcomebot_prod
\dt
SELECT * FROM guilds;
```

---

### 🧹 prod-clean.sh

**Purpose:** Clean up production environment

**What it does:**
1. Shows current resources in `welcomebot-lightprod` namespace
2. Asks for confirmation (requires typing "yes")
3. Deletes the entire namespace

**Usage:**
```bash
./scripts/prod-clean.sh
```

**⚠️ WARNING:** This deletes EVERYTHING including:
- All bot pods
- PostgreSQL database and ALL DATA
- Redis and ALL DATA
- All persistent volumes

Only use when:
- Completely removing the deployment
- Starting fresh
- Switching to different setup

---

## Production Workflow

### Initial Production Setup

```bash
# 1. Configure Harbor credentials
export HARBOR_REGISTRY="harbor.example.com"
export HARBOR_PROJECT="welcomebot"
export HARBOR_USERNAME="your_username"
export HARBOR_PASSWORD="your_password"

# Add to ~/.bashrc or ~/.zshrc for persistence

# 2. Create secrets file
cd deployments/overlays/lightprod
cp secrets.env.example secrets.env
vim secrets.env
# Fill in all values

# 3. Update registry URLs
vim kustomization.yaml
# Update image names to match your Harbor

# 4. Deploy to production
cd ../../..
./scripts/prod-deploy.sh

# 5. Verify deployment
kubectl get pods -n welcomebot-lightprod
./scripts/prod-logs.sh
```

### Daily Production Operations

```bash
# Make code changes
vim internal/features/welcome/feature.go

# Reload production
./scripts/prod-reload.sh

# Monitor logs
./scripts/prod-logs.sh

# Test in Discord
# Commands should work in your production server
```

### Production Troubleshooting

```bash
# Check pod status
kubectl get pods -n welcomebot-lightprod

# View logs
./scripts/prod-logs.sh

# Shell into pod for debugging
./scripts/prod-shell.sh

# Check recent events
kubectl get events -n welcomebot-lightprod --sort-by='.lastTimestamp'

# Describe pod
kubectl describe pod <pod-name> -n welcomebot-lightprod
```

---

## Comparison: Local vs Production Scripts

| Feature | Local (`dev-*.sh`) | Production (`prod-*.sh`) |
|---------|-------------------|-------------------------|
| **Cluster** | Docker Desktop K8s | Production K8s cluster |
| **Namespace** | `welcomebot-local` | `welcomebot-lightprod` |
| **Images** | Local Docker | Harbor registry |
| **Image Tag** | `local` | `lightprod` |
| **Storage** | emptyDir (ephemeral) | NFS CSI (persistent) |
| **Deploy Time** | 30-60 sec | 2-4 min |
| **Registry Secret** | Not needed | Required |
| **Use Case** | Development, testing | Production deployment |

---

## Environment Setup

### Local Development

```bash
# No exports needed - uses local Docker
./scripts/dev-local.sh
```

### Production

```bash
# No exports needed - Harbor credentials pre-configured in scripts
./scripts/prod-deploy.sh
```

**Note:** If you need to override the default Harbor settings, you can still use environment variables:
```bash
export HARBOR_REGISTRY="custom-harbor.com"
export HARBOR_USERNAME="custom-user"
export HARBOR_PASSWORD="custom-password"
./scripts/prod-deploy.sh
```

---

## Common Issues

### Local Development Issues

See troubleshooting section above for local dev issues.

### Production Issues

#### Cannot push to Harbor

```bash
# Test login
docker login $HARBOR_REGISTRY -u $HARBOR_USERNAME

# Check credentials
echo $HARBOR_USERNAME
echo $HARBOR_PASSWORD  # Should be set
```

#### Pods stuck in ImagePullBackOff

```bash
# Check registry secret
kubectl get secret harbor-registry-secret -n welcomebot-lightprod

# Recreate secret
kubectl delete secret harbor-registry-secret -n welcomebot-lightprod
./scripts/prod-deploy.sh  # Recreates it
```

#### Storage class not found

```bash
# Check storage class
kubectl get storageclass nfs-csi

# If missing, update patches to use available storage class
vim deployments/overlays/lightprod/patches/postgres-patch.yaml
vim deployments/overlays/lightprod/redis.yaml
```

---

## See Also

- **Local Setup:** `../deployments/LOCAL_SETUP.md`
- **Production Setup:** `../deployments/LIGHTPROD_SETUP.md`
- **Main Deployment Guide:** `../DEPLOYMENT.md`
- **Deployment Requirements:** `../requirements/deployment.md`
- **Architecture:** `../docs/BOT_ARCHITECTURE.md`

