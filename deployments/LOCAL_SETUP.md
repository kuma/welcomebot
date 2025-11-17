# Local Development Setup Guide

Quick guide for running welcomebot bot locally on Docker Desktop Kubernetes.

## Prerequisites

- ✅ Docker Desktop with Kubernetes enabled
- ✅ kubectl configured
- ✅ Go 1.24+ (for development)

## Quick Start

### 1. Enable Docker Desktop Kubernetes

1. Open Docker Desktop
2. Settings → Kubernetes
3. ✅ Enable Kubernetes
4. Apply & Restart
5. Wait for green indicator

### 2. Deploy Locally

```bash
./scripts/dev-local.sh
```

**What it does:**
- Switches to `docker-desktop` context
- Builds `welcomebot-master:local` and `welcomebot-worker:local`
- Creates `secrets.env.example` if missing
- Applies manifests to `welcomebot-local` namespace
- Waits for pods to be ready

### 3. Configure Secrets (First Time)

If it's your first run, you'll see:

```
⚠ Warning: deployments/overlays/local/secrets.env not found
✓ Created deployments/overlays/local/secrets.env
Please edit and add your Discord bot token!
```

Edit the file:

```bash
vim deployments/overlays/local/secrets.env

# Add your dev bot token:
DISCORD_BOT_TOKEN=your_dev_bot_token_here
```

Then run again:

```bash
./scripts/dev-local.sh
```

### 4. Development Cycle

```bash
# Make code changes
vim internal/features/ping/feature.go

# Rebuild and restart (30-60 sec)
./scripts/dev-reload.sh

# Test in Discord
# /ping

# Check logs if needed
./scripts/dev-logs.sh
```

## Local Configuration

### What's Different from Production?

| Aspect | Production | Local |
|--------|-----------|-------|
| **Images** | Harbor registry | Local Docker daemon |
| **ImagePullPolicy** | Always | **Never** (local only) |
| **Redis** | Sentinel (6 pods) | Simple (1 pod, no persistence) |
| **PostgreSQL** | 200Gi PVC | **emptyDir** (ephemeral) |
| **Workers** | 5 replicas | 1 replica |
| **Resources** | 5Gi RAM | **500Mi RAM** |
| **Log Level** | info | **debug** |
| **Storage** | 40Gi | Ephemeral (deleted on cleanup) |

### Why Ephemeral Storage?

Local uses `emptyDir` for PostgreSQL:
- ✅ **Faster** - No NFS overhead
- ✅ **Simpler** - No PVC management
- ✅ **Fresh start** - Clean state each deployment
- ⚠️ **Data lost** on pod restart (OK for dev!)

**Perfect for development where you want a clean slate!**

## Available Scripts

All scripts are in `scripts/` directory:

### 🚀 dev-local.sh - Initial Deploy

```bash
./scripts/dev-local.sh
```

First time: Creates secrets template
Subsequent runs: Full deploy (builds + applies)

### 🔄 dev-reload.sh - Quick Rebuild

```bash
./scripts/dev-reload.sh
```

**Fastest iteration** (30-60 sec):
- Rebuilds Docker images
- Restarts deployments
- No full redeploy needed

### 📋 dev-logs.sh - View Logs

```bash
./scripts/dev-logs.sh
```

Interactive menu:
1. Master bot (live)
2. Worker bot (live)
3. All pods
4. Master (last 100 lines)
5. Worker (last 100 lines)
6. PostgreSQL
7. Redis

### 🐚 dev-shell.sh - Shell Access

```bash
./scripts/dev-shell.sh
```

Shell into:
1. Master bot
2. Worker bot
3. PostgreSQL
4. Redis

**Example:**
```bash
./scripts/dev-shell.sh
# Select 3 (PostgreSQL)
# Inside pod:
psql -U welcomebot -d welcomebot_dev
\dt
SELECT * FROM guilds;
```

### 🧹 dev-clean.sh - Cleanup

```bash
./scripts/dev-clean.sh
```

Deletes entire `welcomebot-local` namespace:
- All pods
- All services
- All data (fresh start!)

## Manual Commands

### View Resources

```bash
# All resources
kubectl get all -n welcomebot-local

# Pods only
kubectl get pods -n welcomebot-local

# Watch pods
kubectl get pods -n welcomebot-local -w
```

### View Logs

```bash
# Master logs
kubectl logs -f deployment/welcomebot-master -n welcomebot-local

# Worker logs
kubectl logs -f deployment/welcomebot-worker -n welcomebot-local

# All welcomebot pods
kubectl logs -f -l app=welcomebot -n welcomebot-local
```

### Debug

```bash
# Describe pod
kubectl describe pod <pod-name> -n welcomebot-local

# Events
kubectl get events -n welcomebot-local --sort-by='.lastTimestamp'

# Shell into master
kubectl exec -it deployment/welcomebot-master -n welcomebot-local -- sh
```

### Manual Deploy/Update

```bash
# Build images
docker build -t welcomebot-master:local .
docker build -t welcomebot-worker:local .

# Apply manifests
kubectl apply -k deployments/overlays/local

# Restart deployments
kubectl rollout restart deployment/welcomebot-master -n welcomebot-local
kubectl rollout restart deployment/welcomebot-worker -n welcomebot-local
```

## Troubleshooting

### Pods CrashLoopBackOff

**Check logs:**
```bash
./scripts/dev-logs.sh
# Select: 4 (Master last 100 lines)
```

**Common causes:**
- Missing Discord token in secrets.env
- Database not ready
- Redis not accessible

### Images Not Found

```bash
# Verify images exist
docker images | grep welcomebot

# Should see:
# welcomebot-master   local
# welcomebot-worker   local
```

**Fix:**
```bash
./scripts/dev-reload.sh  # Rebuilds images
```

### PostgreSQL Won't Start

```bash
# Check logs
kubectl logs -n welcomebot-local -l app=postgres

# Delete and recreate
kubectl delete pod -l app=postgres -n welcomebot-local
```

### Clean Slate

```bash
# Nuclear option - delete everything and start fresh
./scripts/dev-clean.sh
./scripts/dev-local.sh
```

## Resource Usage

Local deployment is lightweight:

```
Component          RAM      CPU
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Master             128Mi    100m
Worker             128Mi    100m
PostgreSQL         128Mi    100m
Redis              64Mi     50m
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total              ~450Mi   ~350m
```

**Your Mac will barely notice it!** 💻

## Data Persistence

⚠️ **Local data is ephemeral:**

- PostgreSQL uses `emptyDir` (deleted when pod restarts)
- Redis has no persistence
- All data lost on `dev-clean.sh`

**This is intentional!** Local dev benefits from fresh state.

**If you need persistent data:**
```bash
# Don't use dev-clean.sh
# Just use dev-reload.sh for updates
# Pods will keep their data
```

## Best Practices

### 1. Test Locally First

```bash
# Before pushing to staging
./scripts/dev-reload.sh
# Test thoroughly locally
# Only then: git push origin main
```

### 2. Use Different Discord Bot

Don't use production/staging bot token locally:
- Create a separate test bot
- Invite to a test Discord server
- Use that token in local `secrets.env`

### 3. Fast Iteration

```bash
# Don't commit for every test
vim internal/features/ping/feature.go
./scripts/dev-reload.sh  # Test immediately
vim internal/features/ping/feature.go  # Adjust
./scripts/dev-reload.sh  # Test again
# Once working: git commit
```

### 4. Clean Up When Done

```bash
# Done for the day?
./scripts/dev-clean.sh

# Docker Desktop Kubernetes keeps running
# Low resource usage when idle
```

## Comparison: Local vs Remote

### When to Use Local

✅ Quick feature development
✅ Bug fixing
✅ Testing before push
✅ Database schema changes
✅ Learning the codebase

### When to Use Staging

✅ Team testing
✅ Integration testing
✅ Performance testing
✅ Testing with real-like data
✅ QA before production

## Tips

- 💡 Keep local running while developing (no need to clean up constantly)
- 💡 Use `dev-reload.sh` instead of `dev-local.sh` after first setup
- 💡 Check logs with `dev-logs.sh` after each reload
- 💡 Use `dev-shell.sh` to inspect database/Redis state
- 💡 `dev-clean.sh` when switching branches or want fresh start

## Next Steps

After local testing works:

1. ✅ Test feature locally
2. ✅ Commit and push to main
3. ✅ Auto-deploys to staging
4. ✅ Test in staging
5. ✅ Create release tag
6. ✅ Manual sync to production

---

**Local development is fast, isolated, and perfect for iteration!** 🚀

