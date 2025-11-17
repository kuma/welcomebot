# Redis Sentinel Setup

This deployment uses **Redis Sentinel** for high availability.

## Architecture

```
┌─────────────────────────────────────────────────┐
│           Redis Sentinel Cluster                │
│                                                  │
│  ┌──────────────┐    ┌──────────────┐          │
│  │ Sentinel 1   │    │ Sentinel 2   │          │
│  │ :26379       │    │ :26379       │          │
│  └──────┬───────┘    └──────┬───────┘          │
│         │                   │                   │
│         └────────┬──────────┘                   │
│                  │                               │
│         ┌────────▼────────┐                     │
│         │   Sentinel 3    │                     │
│         │   :26379        │                     │
│         └────────┬────────┘                     │
│                  │                               │
│    ┌─────────────┼─────────────┐               │
│    │             │             │                │
│    ▼             ▼             ▼                │
│ ┌────────┐  ┌────────┐  ┌────────┐            │
│ │ Master │  │Replica1│  │Replica2│            │
│ │ :6379  │  │ :6379  │  │ :6379  │            │
│ └────────┘  └────────┘  └────────┘            │
│                                                  │
└─────────────────────────────────────────────────┘
          ▲                    ▲
          │                    │
     Master Bot            Worker Bots
```

## Components

### Redis Master
- **StatefulSet**: 1 replica
- **Storage**: 5Gi PVC
- **Port**: 6379
- **Role**: Read/Write operations

### Redis Replicas  
- **StatefulSet**: 2 replicas
- **Storage**: 5Gi PVC each
- **Port**: 6379
- **Role**: Read operations, failover candidates

### Sentinels
- **Deployment**: 3 replicas
- **Port**: 26379
- **Quorum**: 2 (majority of 3)
- **Role**: Monitor master, automatic failover

## Configuration

### Environment Variables

**Both Master and Worker Bots need:**

```yaml
# Sentinel Configuration (High Availability)
REDIS_SENTINEL_ADDRS: "redis-sentinel:26379"
REDIS_MASTER_NAME: "welcomebot-master"
REDIS_PASSWORD: ""  # Optional

# OR for single Redis (no HA)
REDIS_ADDR: "redis-service:6379"
REDIS_PASSWORD: ""
```

### How It Works

1. **Normal Operation**:
   - Sentinels monitor the master
   - Bots connect to master via Sentinel
   - Writes go to master
   - Reads can go to replicas

2. **Master Failure**:
   - Sentinels detect master is down
   - Quorum (2 out of 3) agrees
   - One replica is promoted to master
   - Bots automatically reconnect to new master

3. **Recovery**:
   - Old master comes back as replica
   - System continues with new master
   - No data loss (with proper configuration)

## Deployment

### Deploy Redis Sentinel

```bash
# Deploy the entire Sentinel cluster
kubectl apply -f deployments/shared/redis-sentinel.yaml

# Verify deployment
kubectl get pods -n welcomebot -l app=redis
kubectl get pods -n welcomebot -l app=redis-sentinel

# Check Sentinel status
kubectl exec -it deployment/redis-sentinel -n welcomebot -- \
  redis-cli -p 26379 sentinel masters

# Check replication
kubectl exec -it redis-master-0 -n welcomebot -- \
  redis-cli info replication
```

### Verify High Availability

```bash
# 1. Check Sentinel sees the master
kubectl exec -it deployment/redis-sentinel -n welcomebot -- \
  redis-cli -p 26379 sentinel get-master-addr-by-name welcomebot-master

# 2. Simulate master failure
kubectl delete pod redis-master-0 -n welcomebot

# 3. Watch automatic failover
kubectl get pods -n welcomebot -l app=redis -w

# 4. Verify new master elected
kubectl exec -it deployment/redis-sentinel -n welcomebot -- \
  redis-cli -p 26379 sentinel get-master-addr-by-name welcomebot-master
```

## Monitoring

### Check Sentinel Status

```bash
# Get master info
kubectl exec -it deployment/redis-sentinel -n welcomebot -- \
  redis-cli -p 26379 sentinel masters

# Get replica info  
kubectl exec -it deployment/redis-sentinel -n welcomebot -- \
  redis-cli -p 26379 sentinel replicas welcomebot-master

# Get sentinel info
kubectl exec -it deployment/redis-sentinel -n welcomebot -- \
  redis-cli -p 26379 sentinel sentinels welcomebot-master
```

### Check Redis Replication

```bash
# On master
kubectl exec -it redis-master-0 -n welcomebot -- \
  redis-cli info replication

# On replica
kubectl exec -it redis-replica-0 -n welcomebot -- \
  redis-cli info replication
```

### Monitor Logs

```bash
# Sentinel logs
kubectl logs -f deployment/redis-sentinel -n welcomebot

# Master logs
kubectl logs -f redis-master-0 -n welcomebot

# Replica logs
kubectl logs -f redis-replica-0 -n welcomebot
```

## Failover Testing

### Manual Failover

```bash
# Force failover to test
kubectl exec -it deployment/redis-sentinel -n welcomebot -- \
  redis-cli -p 26379 sentinel failover welcomebot-master

# Watch it happen
kubectl get pods -n welcomebot -l app=redis -w
```

### Chaos Testing

```bash
# Kill master
kubectl delete pod redis-master-0 -n welcomebot

# Kill a sentinel
kubectl delete pod $(kubectl get pods -n welcomebot -l app=redis-sentinel -o name | head -1 | cut -d'/' -f2) -n welcomebot

# Kill a replica
kubectl delete pod redis-replica-0 -n welcomebot
```

## Scaling

### Scale Replicas

```bash
# Add more replicas
kubectl scale statefulset redis-replica -n welcomebot --replicas=3

# Verify replication
kubectl exec -it redis-master-0 -n welcomebot -- \
  redis-cli info replication
```

### Scale Sentinels

```bash
# Add more sentinels (always use odd numbers: 3, 5, 7)
kubectl scale deployment redis-sentinel -n welcomebot --replicas=5

# Verify
kubectl get pods -n welcomebot -l app=redis-sentinel
```

## Troubleshooting

### Sentinel Not Finding Master

```bash
# Check Sentinel config
kubectl exec -it deployment/redis-sentinel -n welcomebot -- \
  cat /etc/redis/sentinel.conf

# Check Sentinel logs
kubectl logs deployment/redis-sentinel -n welcomebot

# Verify master service
kubectl get svc redis-master -n welcomebot
```

### Replication Broken

```bash
# Check replication on master
kubectl exec -it redis-master-0 -n welcomebot -- \
  redis-cli info replication

# Check on replica
kubectl exec -it redis-replica-0 -n welcomebot -- \
  redis-cli info replication

# Force resync
kubectl delete pod redis-replica-0 -n welcomebot
```

### Bots Can't Connect

```bash
# Verify Sentinel service
kubectl get svc redis-sentinel -n welcomebot

# Test from master bot
kubectl exec -it deployment/welcomebot-master -n welcomebot -- \
  nc -zv redis-sentinel 26379

# Check environment variables
kubectl exec -it deployment/welcomebot-master -n welcomebot -- env | grep REDIS
```

## Configuration Tuning

### Sentinel Timing

Edit `redis-sentinel.yaml` ConfigMap:

```yaml
# How long master must be unreachable before failover (ms)
sentinel down-after-milliseconds welcomebot-master 5000

# Maximum time for failover (ms)
sentinel failover-timeout welcomebot-master 10000

# How many replicas can sync with new master simultaneously
sentinel parallel-syncs welcomebot-master 1
```

### Persistence

Redis is configured with both RDB and AOF:
- **RDB**: Snapshots every 15min/5min/1min based on write volume
- **AOF**: Append-only file for durability
- **fsync**: Every second (balance between performance and durability)

## Migration from Single Redis

If you're switching from single Redis to Sentinel:

1. **Deploy Sentinel cluster**:
```bash
kubectl apply -f deployments/shared/redis-sentinel.yaml
```

2. **Update bot deployments**:
```yaml
# Change from:
- name: REDIS_ADDR
  value: "redis-service:6379"

# To:
- name: REDIS_SENTINEL_ADDRS
  value: "redis-sentinel:26379"
- name: REDIS_MASTER_NAME
  value: "welcomebot-master"
```

3. **Rolling update**:
```bash
kubectl rollout restart deployment/welcomebot-master -n welcomebot
kubectl rollout restart deployment/welcomebot-worker -n welcomebot
```

4. **Verify**:
```bash
kubectl logs -f deployment/welcomebot-master -n welcomebot | grep -i redis
```

5. **Remove old Redis** (after verification):
```bash
kubectl delete -f deployments/shared/redis.yaml
```

## Production Checklist

- [ ] 3+ Sentinels running
- [ ] Quorum properly configured (n/2 + 1)
- [ ] Master + 2 replicas running
- [ ] Persistent volumes attached
- [ ] Monitoring configured
- [ ] Alert rules for failover events
- [ ] Backup strategy in place
- [ ] Tested failover procedure
- [ ] Resource limits set
- [ ] Network policies configured

## Benefits of Sentinel

✅ **Automatic Failover** - No manual intervention  
✅ **High Availability** - Master failure doesn't stop the system  
✅ **Zero Downtime** - Bots reconnect automatically  
✅ **Data Durability** - AOF + RDB persistence  
✅ **Monitoring** - Sentinels monitor health  
✅ **Scalable** - Add more replicas for read scaling  

## Support

For issues:
1. Check Sentinel logs
2. Verify replication status
3. Test Sentinel connectivity
4. Review bot logs for Redis errors

