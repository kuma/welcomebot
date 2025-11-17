# Welcome Bot Quick Start Guide

## Prerequisites

1. **4 Discord Bot Accounts**
   - 1 Master bot
   - 3 Slave bots (slave-1, slave-2, slave-3)
   - All need tokens from Discord Developer Portal

2. **Infrastructure**
   - PostgreSQL database
   - Redis instance
   - Go 1.24+

## Step 1: Database Setup

```bash
# Create database
createdb welcomebot

# Run migrations (automatic on first master bot start)
```

## Step 2: Build

```bash
go build ./cmd/master
go build ./cmd/worker
```

## Step 3: Configure Environment

### Master Bot
Copy `run-master.sh.example` to `run-master.sh`:
```bash
export DISCORD_BOT_TOKEN="your-master-token"
export POSTGRES_HOST="localhost"
export POSTGRES_PASSWORD="password"
export REDIS_ADDR="localhost:6379"
```

### Slave Bots (create 3 files)

**run-worker-1.sh:**
```bash
export SLAVE_ID="slave-1"
export DISCORD_BOT_TOKEN="your-slave1-token"
# ... same database/redis config
```

**run-worker-2.sh:**
```bash
export SLAVE_ID="slave-2"
export DISCORD_BOT_TOKEN="your-slave2-token"
# ... same database/redis config
```

**run-worker-3.sh:**
```bash
export SLAVE_ID="slave-3"
export DISCORD_BOT_TOKEN="your-slave3-token"
# ... same database/redis config
```

## Step 4: Audio Files

Place audio files in `./audio/` directory:
```bash
./audio/welcome.mp3
./audio/step1.mp3
./audio/step2.mp3
```

See `audio/README.md` for creating audio files.

## Step 5: Start Bots

Open 4 terminal windows:

**Terminal 1 - Master:**
```bash
./run-master.sh
```

**Terminal 2 - Slave 1:**
```bash
./run-worker-1.sh
```

**Terminal 3 - Slave 2:**
```bash
./run-worker-2.sh
```

**Terminal 4 - Slave 3:**
```bash
./run-worker-3.sh
```

## Step 6: Configure in Discord

1. Run `/welcomebot` in your Discord server
2. Click "👑 Admin" → "⚙️ Configuration"
3. Click "👋 Setup Welcome Onboarding"
4. Select welcome text channel (where button will appear)
5. Select VC category (where temporary VCs will be created)

## Step 7: Test

1. A button should appear in your configured welcome channel
2. Click "Start Onboarding"
3. A private voice channel will be created
4. Join the VC to test the onboarding flow

## Verification

### Check Master Bot
```bash
# Should see:
# - Bot connected
# - Database migrations completed
# - Welcome feature registered
```

### Check Slave Bots
```bash
# Each slave should show:
# - Discord connected
# - Initial slave status set to available
# - Worker started, waiting for tasks
```

### Check Redis
```bash
redis-cli
> KEYS welcomebot:slaves:status:*
# Should return 3 keys (one per slave)

> GET welcomebot:slaves:status:slave-1
# Should return "available"
```

## Common Issues

### "All bots busy"
- Check that all 3 slave bots are running
- Check Redis connection
- Verify slave status: `redis-cli GET welcomebot:slaves:status:slave-1`

### Button doesn't appear
- Check master bot logs
- Verify configuration was saved
- Check bot permissions in Discord

### VC not created
- Check slave bot permissions
- Verify category ID is correct
- Check slave bot logs

### Voice connection fails
- Ensure slave bot has Voice permissions
- Check voice intents are enabled
- Review worker logs

## Logs

Monitor logs for debugging:

```bash
# Master
tail -f master.log

# Slaves
tail -f worker-1.log
tail -f worker-2.log
tail -f worker-3.log
```

## Stopping

```bash
# Each terminal: Ctrl+C
# Bots will gracefully shutdown
```

## Next Steps

1. **Add Audio**: Place real audio files in `./audio/`
2. **Configure Roles**: Add in-progress and completed role IDs in database
3. **Customize Flow**: Edit `internal/worker/onboarding_session.go` for custom steps
4. **Deploy**: See `DEPLOYMENT_QUICKSTART.md` for Kubernetes deployment

## Architecture Reminder

```
User clicks button
    ↓
Master checks slave availability
    ↓
Master enqueues task for available slave
    ↓
Slave receives task
    ↓
Slave creates VC + joins voice
    ↓
Interactive onboarding flow
    ↓
Slave adds completion role
    ↓
Slave deletes VC + cleanup
    ↓
Slave marks self as available again
```

## Support

- Full details: `requirements/welcome.md`
- Audio setup: `audio/README.md`
- Implementation: `IMPLEMENTATION_SUMMARY.md`
- Architecture: `docs/BOT_ARCHITECTURE.md`

