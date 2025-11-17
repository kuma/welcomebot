# Lightweight Production Setup Guide

Simple, script-based production deployment for the welcomebot Discord bot.

## Overview

This is a **lightweight production setup** that repurposes the local development workflow for production use. Unlike the full CI/CD pipeline with ArgoCD and GitHub Actions, this approach uses simple scripts that you run from your local machine.

**Key Differences from Local Dev:**
- Uses **Harbor registry** instead of local Docker images
- Deploys to **production Kubernetes cluster** (not Docker Desktop)
- Uses **NFS CSI** for persistent storage
- Namespace: `welcomebot-lightprod`

**Key Differences from Full Production:**
- No ArgoCD (manual deployment via scripts)
- No GitHub Actions (manual build and push)
- Simple Redis (no Sentinel HA)
- Lightweight but production-ready

## Prerequisites

### 1. Kubernetes Cluster Access
- Access to your production Kubernetes cluster
- `kubectl` configured to access the cluster
- Verify: `kubectl cluster-info`

### 2. Harbor Registry Access
- Harbor registry URL (e.g., `harbor.example.com`)
- Harbor username and password
- Project created in Harbor (e.g., `welcomebot`)

### 3. Storage Class
- NFS CSI storage class available: `nfs-csi`
- Verify: `kubectl get storageclass nfs-csi`

### 4. Docker
- Docker installed locally for building images
- Verify: `docker version`

### 5. Discord Bot Tokens
- 11 Discord bot tokens (1 master + 10 slaves)
- See `TOKEN_SETUP_GUIDE.md` for creating bot accounts

## Quick Start

### Step 1: Configure Secrets

**Note:** Harbor credentials are pre-configured in the deployment scripts. No environment variables needed!

Create secrets file from example:

```bash
cd deployments/overlays/lightprod
cp secrets.env.example secrets.env
vim secrets.env
```

Fill in your values:

```bash
# Master Bot Token
DISCORD_BOT_TOKEN=your_master_bot_token_here

# Slave Bot Tokens (10 separate bot accounts)
SLAVE_1_TOKEN=your_slave1_bot_token_here
SLAVE_2_TOKEN=your_slave2_bot_token_here
SLAVE_3_TOKEN=your_slave3_bot_token_here
SLAVE_4_TOKEN=your_slave4_bot_token_here
SLAVE_5_TOKEN=your_slave5_bot_token_here
SLAVE_6_TOKEN=your_slave6_bot_token_here
SLAVE_7_TOKEN=your_slave7_bot_token_here
SLAVE_8_TOKEN=your_slave8_bot_token_here
SLAVE_9_TOKEN=your_slave9_bot_token_here
SLAVE_10_TOKEN=your_slave10_bot_token_here

# PostgreSQL Configuration
POSTGRES_USER=welcomebot_prod
POSTGRES_PASSWORD=your_strong_password_here
POSTGRES_DB=welcomebot_prod

# Redis Password
REDIS_PASSWORD=your_redis_password_here
```

### Step 3: Update Registry Configuration

Edit the kustomization file to match your Harbor setup:

```bash
vim deployments/overlays/lightprod/kustomization.yaml
```

Update the image names:

```yaml
images:
  - name: harbor.example.com/welcomebot/welcomebot-master
    newName: harbor.example.com/welcomebot/welcomebot-master  # Update this
    newTag: lightprod
  - name: harbor.example.com/welcomebot/welcomebot-worker
    newName: harbor.example.com/welcomebot/welcomebot-worker  # Update this
    newTag: lightprod
```

### Step 4: Deploy to Production

From project root, run:

```bash
./scripts/prod-deploy.sh
```

**What it does:**
1. Logs in to Harbor registry
2. Builds Docker images locally
3. Pushes images to Harbor with `lightprod` tag
4. Creates Harbor registry secret in Kubernetes
5. Applies Kubernetes manifests via Kustomize
6. Waits for pods to be ready

**First deployment takes 3-5 minutes.**

### Step 5: Verify Deployment

Check pod status:

```bash
kubectl get pods -n welcomebot-lightprod
```

Expected output:

```
NAME                                         READY   STATUS    RESTARTS   AGE
postgres-0                                   1/1     Running   0          2m
redis-xxxxxxxxxx-xxxxx                       1/1     Running   0          2m
welcomebot-master-xxxxxxxxxx-xxxxx           1/1     Running   0          2m
welcomebot-worker-slave1-xxxxxxxxxx-xxxxx    1/1     Running   0          2m
welcomebot-worker-slave2-xxxxxxxxxx-xxxxx    1/1     Running   0          2m
welcomebot-worker-slave3-xxxxxxxxxx-xxxxx    1/1     Running   0          2m
welcomebot-worker-slave4-xxxxxxxxxx-xxxxx    1/1     Running   0          2m
welcomebot-worker-slave5-xxxxxxxxxx-xxxxx    1/1     Running   0          2m
welcomebot-worker-slave6-xxxxxxxxxx-xxxxx    1/1     Running   0          2m
welcomebot-worker-slave7-xxxxxxxxxx-xxxxx    1/1     Running   0          2m
welcomebot-worker-slave8-xxxxxxxxxx-xxxxx    1/1     Running   0          2m
welcomebot-worker-slave9-xxxxxxxxxx-xxxxx    1/1     Running   0          2m
welcomebot-worker-slave10-xxxxxxxxxx-xxxxx   1/1     Running   0          2m
```

## Daily Operations

### Making Code Changes

After editing code:

```bash
# 1. Make your changes
vim internal/features/ping/feature.go

# 2. Reload production (builds, pushes, restarts)
./scripts/prod-reload.sh

# 3. Verify deployment
kubectl get pods -n welcomebot-lightprod
```

**Reload time: 2-4 minutes** (build + push + restart)

### Viewing Logs

Interactive log viewer:

```bash
./scripts/prod-logs.sh
```

Menu options:
1. Master bot (live)
2-11. Worker slave 1-10 (live)
12. All bot pods (live)
13. Master (last 100 lines)
14. All workers (last 100 lines each)
15-18. PostgreSQL/Redis logs
19. Pod status and events

**Direct log access:**

```bash
# Master logs
kubectl logs -f deployment/welcomebot-master -n welcomebot-lightprod

# Worker logs
kubectl logs -f deployment/welcomebot-worker-slave1 -n welcomebot-lightprod

# All bot logs
kubectl logs -f -l app=welcomebot -n welcomebot-lightprod --prefix=true
```

### Shell Access

Interactive shell access:

```bash
./scripts/prod-shell.sh
```

Menu options:
1. Master bot
2-11. Worker slave 1-10
12. PostgreSQL
13. Redis

**Direct shell access:**

```bash
# Master bot
kubectl exec -it deployment/welcomebot-master -n welcomebot-lightprod -- sh

# PostgreSQL
kubectl exec -it statefulset/postgres -n welcomebot-lightprod -- sh
# Then: psql -U welcomebot_prod -d welcomebot_prod

# Redis
kubectl exec -it deployment/redis -n welcomebot-lightprod -- sh
# Then: redis-cli
```

### Cleanup

⚠️ **WARNING: This deletes EVERYTHING including data!**

```bash
./scripts/prod-clean.sh
```

Requires typing "yes" to confirm.

## Architecture

### Components

```
welcomebot-lightprod namespace
├── Master Bot (1 pod)
│   └── Handles Discord gateway, slash commands
├── Worker Bots (10 pods - slave1 through slave10)
│   └── Handle voice channel connections
├── PostgreSQL (1 pod)
│   └── Persistent storage (20Gi NFS)
└── Redis (1 pod)
    └── Caching and queues (5Gi NFS)
```

### Resource Allocation

| Component | Memory | CPU | Storage |
|-----------|--------|-----|---------|
| Master | 512Mi-1Gi | 250m-500m | - |
| Worker (each x10) | 512Mi-1Gi | 250m-500m | - |
| PostgreSQL | 512Mi-1Gi | 250m-500m | 20Gi NFS |
| Redis | 256Mi-512Mi | 100m-200m | 5Gi NFS |

**Total:** ~6-11Gi RAM, ~3-6 CPU cores, 25Gi storage

### Persistent Storage

Both PostgreSQL and Redis use NFS CSI for persistent storage:

- **Storage Class:** `nfs-csi`
- **PostgreSQL:** 20Gi PVC via StatefulSet
- **Redis:** 5Gi PVC

**Data persists across pod restarts.**

### Networking

All services use ClusterIP (internal only):

- `welcomebot-master-service:8080` - Master bot
- `welcomebot-worker-service:8080` - Worker bots
- `postgres-service:5432` - PostgreSQL
- `redis-service:6379` - Redis

## Configuration

### Environment Variables

Configure via `deployments/overlays/lightprod/secrets.env`:

```bash
# Discord Tokens
DISCORD_BOT_TOKEN=master_token
SLAVE_1_TOKEN=worker1_token
SLAVE_2_TOKEN=worker2_token
SLAVE_3_TOKEN=worker3_token

# PostgreSQL
POSTGRES_HOST=postgres-service  # Auto-configured
POSTGRES_PORT=5432
POSTGRES_USER=welcomebot_prod
POSTGRES_PASSWORD=strong_password
POSTGRES_DB=welcomebot_prod
POSTGRES_SSLMODE=disable

# Redis
REDIS_ADDR=redis-service:6379  # Auto-configured
REDIS_PASSWORD=redis_password

# Logging
LOG_LEVEL=info  # Auto-configured in patches
LOG_FORMAT=json
```

### Scaling Workers

To change number of worker replicas:

```bash
vim deployments/overlays/lightprod/patches/worker-slave1-patch.yaml
# Change replicas: 1 to desired number

./scripts/prod-reload.sh
```

### Changing Resources

To adjust memory/CPU limits:

```bash
vim deployments/overlays/lightprod/patches/master-patch.yaml
# Adjust resources section

./scripts/prod-reload.sh
```

## Troubleshooting

### Pods Not Starting

Check pod status:

```bash
kubectl get pods -n welcomebot-lightprod
kubectl describe pod <pod-name> -n welcomebot-lightprod
```

Common issues:

1. **ImagePullBackOff**
   - Harbor credentials incorrect
   - Registry secret not created
   - Image doesn't exist in Harbor

   Fix:
   ```bash
   kubectl delete secret harbor-registry-secret -n welcomebot-lightprod
   ./scripts/prod-deploy.sh  # Recreates secret
   ```

2. **CrashLoopBackOff**
   - Check logs: `./scripts/prod-logs.sh`
   - Common causes:
     - Missing Discord tokens
     - Database connection failed
     - Redis connection failed

3. **Pending (Storage)**
   - NFS CSI not available
   - Storage class not found

   Check:
   ```bash
   kubectl get storageclass nfs-csi
   kubectl get pvc -n welcomebot-lightprod
   ```

### Cannot Push to Harbor

1. **Login failed**
   ```bash
   docker login harbor.example.com -u your_username
   ```

2. **Project doesn't exist**
   - Create project in Harbor UI
   - Ensure username has push permissions

3. **Image name incorrect**
   - Verify: `harbor.example.com/project/image:tag`
   - Must match Harbor project name

### Database Issues

Connect to PostgreSQL:

```bash
./scripts/prod-shell.sh
# Select: 5 (PostgreSQL)
psql -U welcomebot_prod -d welcomebot_prod
```

Check tables:
```sql
\dt
SELECT * FROM guilds LIMIT 10;
```

Reset database (⚠️ deletes all data):
```bash
kubectl delete statefulset postgres -n welcomebot-lightprod
kubectl delete pvc postgres-data-postgres-0 -n welcomebot-lightprod
./scripts/prod-deploy.sh
```

### Redis Issues

Connect to Redis:

```bash
./scripts/prod-shell.sh
# Select: 6 (Redis)
redis-cli
```

Check Redis:
```redis
INFO
KEYS *
```

## Comparison: Lightprod vs Full Production

| Feature | Lightprod | Full Production |
|---------|-----------|----------------|
| **Deployment** | Manual scripts | ArgoCD GitOps |
| **CI/CD** | Local build/push | GitHub Actions |
| **Registry** | Harbor (manual) | Harbor (automated) |
| **Redis** | Simple (1 pod) | Sentinel HA (6 pods) |
| **SSL** | No (internal) | Yes (production) |
| **Monitoring** | kubectl/logs | ArgoCD UI |
| **Rollback** | Manual | ArgoCD history |
| **Complexity** | ⭐ Low | ⭐⭐⭐⭐ High |
| **Setup Time** | 15 minutes | 2-3 hours |
| **Suitable For** | Small/medium | Enterprise |

## Security Best Practices

1. **Use Strong Passwords**
   - PostgreSQL password: 20+ characters
   - Redis password: 20+ characters
   - Never commit `secrets.env` to Git

2. **Rotate Discord Tokens**
   - Update tokens periodically
   - Use separate tokens for production
   - Never share tokens

3. **Harbor Access**
   - Use Harbor robot accounts (not personal)
   - Limit permissions to push/pull only
   - Rotate credentials regularly

4. **Network Security**
   - All services use ClusterIP (internal only)
   - No external exposure by default
   - Use network policies if needed

5. **Secrets Management**
   - `secrets.env` is git-ignored
   - Kubernetes secrets are base64 encoded
   - Consider using sealed-secrets for extra security

## Backup and Recovery

### Backup PostgreSQL

```bash
kubectl exec -it statefulset/postgres -n welcomebot-lightprod -- \
  pg_dump -U welcomebot_prod welcomebot_prod > backup-$(date +%Y%m%d).sql
```

### Restore PostgreSQL

```bash
cat backup-20241114.sql | kubectl exec -i statefulset/postgres -n welcomebot-lightprod -- \
  psql -U welcomebot_prod -d welcomebot_prod
```

### Backup Redis

```bash
kubectl exec -it deployment/redis -n welcomebot-lightprod -- \
  redis-cli SAVE

kubectl cp welcomebot-lightprod/redis-xxxxx:/data/dump.rdb \
  ./redis-backup-$(date +%Y%m%d).rdb
```

## Upgrading

### Updating Go Version

1. Edit `Dockerfile`:
   ```dockerfile
   FROM golang:1.24-alpine AS builder  # Update version
   ```

2. Rebuild and deploy:
   ```bash
   ./scripts/prod-reload.sh
   ```

### Updating Dependencies

1. Update `go.mod`:
   ```bash
   go get -u ./...
   go mod tidy
   ```

2. Rebuild and deploy:
   ```bash
   ./scripts/prod-reload.sh
   ```

### Database Migrations

1. Add migration files to `internal/core/database/migrations/`
2. Migrations run automatically on bot startup
3. Deploy with `./scripts/prod-reload.sh`

## Monitoring

### Pod Health

```bash
# Watch pod status
kubectl get pods -n welcomebot-lightprod -w

# Check events
kubectl get events -n welcomebot-lightprod --sort-by='.lastTimestamp'

# Resource usage
kubectl top pods -n welcomebot-lightprod
```

### Application Logs

```bash
# Real-time logs
./scripts/prod-logs.sh

# Search logs
kubectl logs deployment/welcomebot-master -n welcomebot-lightprod | grep ERROR

# Export logs
kubectl logs deployment/welcomebot-master -n welcomebot-lightprod > master.log
```

### Database Health

```bash
# PostgreSQL stats
kubectl exec -it statefulset/postgres -n welcomebot-lightprod -- \
  psql -U welcomebot_prod -d welcomebot_prod -c "SELECT * FROM pg_stat_activity;"

# Database size
kubectl exec -it statefulset/postgres -n welcomebot-lightprod -- \
  psql -U welcomebot_prod -d welcomebot_prod -c "SELECT pg_size_pretty(pg_database_size('welcomebot_prod'));"
```

## FAQ

### Q: Can I use this for a large Discord server?

Yes, but consider scaling workers if you have many voice channels. Simple Redis is usually sufficient for most use cases.

### Q: How do I add more worker bots?

The current setup supports 3 worker bots (slaves). To add more, you would need to:
1. Create additional worker deployment files in `base/`
2. Add patches in `lightprod/patches/`
3. Update kustomization.yaml

### Q: Can I use a different storage class?

Yes, edit the `storageClassName` in:
- `deployments/overlays/lightprod/redis.yaml`
- `deployments/overlays/lightprod/patches/postgres-patch.yaml`

### Q: How do I migrate from local dev to lightprod?

Lightprod is a separate environment. To migrate data:
1. Backup local PostgreSQL: See "Backup and Recovery"
2. Deploy lightprod: `./scripts/prod-deploy.sh`
3. Restore to lightprod: See "Backup and Recovery"

### Q: Can I use this with multiple Kubernetes clusters?

Yes, use `kubectl` contexts to switch between clusters:

```bash
kubectl config use-context prod-cluster
./scripts/prod-deploy.sh
```

## Next Steps

After deployment:

1. ✅ Test all Discord commands in your production server
2. ✅ Set up regular backups (PostgreSQL + Redis)
3. ✅ Monitor logs for errors
4. ✅ Consider setting up alerts (e.g., via kubectl + cron)
5. ✅ Document your Harbor credentials securely
6. ✅ Create runbook for common operations

---

**Happy deploying! 🚀**

For questions or issues, check the main project README or deployment documentation.

