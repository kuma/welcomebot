# Discord Bot Token Setup Guide

You need **4 Discord bot accounts** for the welcome bot system:
- 1 Master bot
- 3 Slave bots

## Where to Configure Tokens

### Option 1: Local Development with Shell Scripts (Recommended for Testing)

Create individual shell scripts for each bot:

#### Master Bot (`run-master.sh`)
```bash
#!/bin/bash
export DISCORD_BOT_TOKEN="YOUR_MASTER_TOKEN_HERE"
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5432"
export POSTGRES_USER="welcomebot"
export POSTGRES_PASSWORD="your_password"
export POSTGRES_DB="welcomebot"
export REDIS_ADDR="localhost:6379"
export LOG_LEVEL="info"
./master
```

#### Slave 1 (`run-worker-1.sh`)
```bash
#!/bin/bash
export SLAVE_ID="slave-1"
export DISCORD_BOT_TOKEN="YOUR_SLAVE1_TOKEN_HERE"
export POSTGRES_HOST="localhost"
export POSTGRES_PASSWORD="your_password"
export REDIS_ADDR="localhost:6379"
./worker
```

#### Slave 2 (`run-worker-2.sh`)
```bash
#!/bin/bash
export SLAVE_ID="slave-2"
export DISCORD_BOT_TOKEN="YOUR_SLAVE2_TOKEN_HERE"
export POSTGRES_HOST="localhost"
export POSTGRES_PASSWORD="your_password"
export REDIS_ADDR="localhost:6379"
./worker
```

#### Slave 3 (`run-worker-3.sh`)
```bash
#!/bin/bash
export SLAVE_ID="slave-3"
export DISCORD_BOT_TOKEN="YOUR_SLAVE3_TOKEN_HERE"
export POSTGRES_HOST="localhost"
export POSTGRES_PASSWORD="your_password"
export REDIS_ADDR="localhost:6379"
./worker
```

**Make executable:**
```bash
chmod +x run-master.sh run-worker-*.sh
```

**Run each in separate terminal:**
```bash
# Terminal 1
./run-master.sh

# Terminal 2
./run-worker-1.sh

# Terminal 3
./run-worker-2.sh

# Terminal 4
./run-worker-3.sh
```

### Option 2: Kubernetes with secrets.env

1. **Copy and edit secrets.env:**
```bash
cd deployments/overlays/local
cp secrets.env.example secrets.env
nano secrets.env  # or your editor
```

2. **Fill in all 4 tokens:**
```bash
DISCORD_BOT_TOKEN=your_master_token_here
SLAVE_1_TOKEN=your_slave1_token_here
SLAVE_2_TOKEN=your_slave2_token_here
SLAVE_3_TOKEN=your_slave3_token_here
```

3. **Deploy:**
```bash
./scripts/dev-local.sh
```

### Option 3: Direct Environment Variables

Set directly in your shell:

```bash
# Master
export DISCORD_BOT_TOKEN="master_token"
./master

# Slaves (in separate terminals)
export SLAVE_ID="slave-1"
export DISCORD_BOT_TOKEN="slave1_token"
./worker

export SLAVE_ID="slave-2"
export DISCORD_BOT_TOKEN="slave2_token"
./worker

export SLAVE_ID="slave-3"
export DISCORD_BOT_TOKEN="slave3_token"
./worker
```

## How to Get Discord Bot Tokens

### Step 1: Go to Discord Developer Portal
https://discord.com/developers/applications

### Step 2: Create 4 Applications

1. Click "New Application"
2. Name: "WelcomeBot Master" (or similar)
3. Go to "Bot" tab
4. Click "Add Bot"
5. **Copy the token** (keep it secret!)
6. Repeat for 3 slave bots:
   - "WelcomeBot Slave 1"
   - "WelcomeBot Slave 2"
   - "WelcomeBot Slave 3"

### Step 3: Enable Required Intents

For ALL 4 bots, enable these intents:
- ✅ Presence Intent
- ✅ Server Members Intent
- ✅ Message Content Intent

### Step 4: Invite Bots to Your Server

For each bot, generate invite URL:

1. Go to "OAuth2" → "URL Generator"
2. Select scopes:
   - ✅ `bot`
   - ✅ `applications.commands`
3. Select bot permissions:
   - ✅ Read Messages/View Channels
   - ✅ Send Messages
   - ✅ Manage Messages
   - ✅ Embed Links
   - ✅ Create Voice Channels
   - ✅ Connect
   - ✅ Speak
   - ✅ Manage Channels
   - ✅ Manage Roles (if using role features)
4. Copy generated URL
5. Open in browser and invite to your test server

**Repeat for all 4 bots!**

## Token Security

⚠️ **NEVER commit tokens to git!**

The following files are in `.gitignore`:
- `run-master.sh`
- `run-worker.sh`
- `secrets.env`

Always use:
- Environment variables
- Shell scripts (not committed)
- Kubernetes secrets
- Secret management tools (production)

## Verification

Check that all 4 bots are in your server:
```
Server Settings → Members → Search "welcomebot"
```

You should see:
- WelcomeBot Master (or your name)
- WelcomeBot Slave 1
- WelcomeBot Slave 2
- WelcomeBot Slave 3

## Quick Test

After starting all 4 bots:

```bash
# Check logs show all connected
# Master:
# "bot connected" user="WelcomeBot Master#1234"

# Each slave:
# "Discord connected" user="WelcomeBot Slave 1#5678"
# "Discord connected" user="WelcomeBot Slave 2#9012"
# "Discord connected" user="WelcomeBot Slave 3#3456"
```

## Troubleshooting

### "Invalid Token"
- Double-check you copied the full token
- Regenerate token in Developer Portal if needed
- Make sure no extra spaces/newlines

### "Missing Permissions"
- Check bot has required permissions in server
- Verify intents are enabled in Developer Portal

### Slave Not Connecting
- Verify `SLAVE_ID` is set correctly (slave-1, slave-2, or slave-3)
- Check token is for the correct bot
- Verify all environment variables are set

## Production Setup

For production, use proper secret management:
- Kubernetes Secrets
- AWS Secrets Manager
- HashiCorp Vault
- Azure Key Vault

Never use shell scripts with hardcoded tokens in production!

