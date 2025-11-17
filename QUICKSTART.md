# welcomebot Bot - Quick Start Guide

## What's Been Built

You now have a **production-ready Discord bot foundation** with:

✅ **Clean Architecture** - No god objects, clear separation of concerns  
✅ **Type-Safe** - No `interface{}` pollution, compile-time safety  
✅ **Well-Tested** - Full test coverage on all core services  
✅ **AI-Ready** - Comprehensive guidelines for AI-assisted development  
✅ **2 Working Features** - Ping and BotInfo commands ready to use

## Quick Start

### 1. Set Up Environment

```bash
# Set required environment variables
export DISCORD_BOT_TOKEN="your-bot-token-here"

# Optional: Database (default: localhost)
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5432"
export POSTGRES_USER="welcomebot"
export POSTGRES_PASSWORD="yourpassword"
export POSTGRES_DB="welcomebot"

# Optional: Redis (default: localhost)
export REDIS_ADDR="localhost:6379"
export REDIS_PASSWORD=""

# Optional: Logging
export LOG_LEVEL="info"  # debug, info, warn, error
export LOG_FORMAT="json" # json, text
```

### 2. Run the Bot

```bash
cd /Users/k/w/welcomebot

# Build
go build -o bin/master ./cmd/master

# Run
./bin/master
```

### 3. Test It

In Discord, try these commands:
- `/ping` - Check bot responsiveness
- `/botinfo` - See bot information

## Adding Your First Feature

### Option A: Create New Feature

1. **Write Requirements** (`requirements/welcome.md`):
```markdown
# Feature: Welcome Messages

## User-Facing Description
Send a welcome message when new members join the server.

## Commands/Interactions
- `/setup-welcome channel:#channel message:"Welcome {user}!"`
- Event: Send message when member joins

## Data Models
- guild_id, channel_id, message_template

## Business Logic
- Admin only can configure
- {user} placeholder replaced with mention
- Message sent to configured channel

## Examples
User joins → Bot: "Welcome @User to the server!"
```

2. **Use AI to Implement**:
```
[Copy paste from docs/FEATURE_TEMPLATE.md with your requirements]
```

3. **Register in `cmd/master/main.go`**:
```go
import "welcomebot/internal/features/welcome"

// In main()
welcomeFeature, err := welcome.New(welcome.Dependencies{
    DB: deps.DB,
    Discord: deps.Discord,
    Logger: deps.Logger,
})
...
bot.Registry().Register(welcomeFeature)
```

4. **Test & Deploy**:
```bash
go test ./internal/features/welcome/...
go build ./cmd/master
./bin/master
```

### Option B: Extract from Old Bot

1. **Run old bot**, document behavior (don't read code!)
2. **Write requirements** based on observed behavior
3. **Implement fresh** following guidelines
4. **Test side-by-side** with old bot
5. **Switch over** when confident

## Project Structure

```
/Users/k/w/welcomebot/
├── cmd/
│   └── master/          # ← Bot entry point
├── internal/
│   ├── bot/             # ← Core framework
│   ├── core/            # ← Services (DB, cache, etc)
│   ├── features/        # ← Add features here
│   │   ├── ping/
│   │   └── botinfo/
│   └── shared/          # ← Common types
├── docs/                # ← Documentation
│   ├── CODING_GUIDELINES.md  # ← Read this!
│   ├── FEATURE_TEMPLATE.md   # ← Use this!
│   └── PROGRESS.md          # ← Status
├── requirements/        # ← Feature specs
└── README.md
```

## Key Documents

| Document | Purpose |
|----------|---------|
| `docs/CODING_GUIDELINES.md` | **Must read** - Coding standards |
| `docs/FEATURE_TEMPLATE.md` | **Use this** - Create new features |
| `docs/PROGRESS.md` | What's done, what's next |
| `requirements/*.md` | Feature specifications |

## Development Workflow

```
┌─────────────────────────────────────┐
│ 1. Create requirements/FEATURE.md  │
└──────────┬──────────────────────────┘
           │
┌──────────▼──────────────────────────┐
│ 2. Give AI requirements + template  │
└──────────┬──────────────────────────┘
           │
┌──────────▼──────────────────────────┐
│ 3. AI generates clean code          │
└──────────┬──────────────────────────┘
           │
┌──────────▼──────────────────────────┐
│ 4. Test: go test ./internal/...     │
└──────────┬──────────────────────────┘
           │
┌──────────▼──────────────────────────┐
│ 5. Register in cmd/master/main.go   │
└──────────┬──────────────────────────┘
           │
┌──────────▼──────────────────────────┐
│ 6. Deploy and iterate               │
└─────────────────────────────────────┘
```

## Architecture Overview

### Core Services (Dependency Injection)

Every feature gets these injected:

```go
type Dependencies struct {
    DB      database.Client  // PostgreSQL
    Cache   cache.Client     // Redis
    Discord discord.Helper   // Discord API
    Logger  logger.Logger    // Structured logs
}
```

### Feature Interface

Every feature implements:

```go
type Feature interface {
    Name() string
    HandleInteraction(...)  // Slash commands, buttons, modals
    RegisterCommands()      // Slash command definitions
}
```

Optional: `MessageFeature`, `ReactionFeature`, `VoiceFeature`, `MemberFeature`

### Event Flow

```
Discord Event
    ↓
Bot Handler
    ↓
Feature Registry (routes to correct feature)
    ↓
Feature.HandleInteraction() or Feature.HandleMessage()
```

## Coding Rules (Quick Reference)

1. ❌ **No `interface{}`** - Use concrete types
2. ⚠️ **Functions ≤ 50 lines** - Split if longer
3. ⚠️ **Files ≤ 300 lines** - Split by responsibility
4. ✅ **Always handle errors** - `fmt.Errorf("context: %w", err)`
5. ✅ **Use context** - First param for I/O functions
6. ✅ **Constructor pattern** - `func New(deps Dependencies) (*Feature, error)`
7. ✅ **Document exports** - All public symbols
8. ✅ **Write tests** - `feature_test.go` for every feature
9. ✅ **Log structured** - `logger.Info("msg", "key", val)`
10. ✅ **Return ErrNotHandled** - If interaction not for you

## Common Tasks

### Run Tests
```bash
go test ./...
```

### Run Linter
```bash
golangci-lint run
```

### Build
```bash
go build ./cmd/master
```

### Add Dependency
```bash
go get github.com/package/name
go mod tidy
```

### Create Feature
```bash
mkdir -p internal/features/myfeature
# Use template from docs/FEATURE_TEMPLATE.md
```

## Troubleshooting

### Bot won't start
- Check `DISCORD_BOT_TOKEN` is set
- Verify token is valid in Discord Developer Portal

### Database connection failed
- Check PostgreSQL is running
- Verify `POSTGRES_*` environment variables
- Test connection: `psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB`

### Redis connection failed
- Check Redis is running: `redis-cli ping`
- Verify `REDIS_ADDR`

### Feature not working
- Check feature is registered in `cmd/master/main.go`
- Check logs for errors
- Verify slash commands registered: Bot → Discord Developer Portal → OAuth2

### Build errors
- Run `go mod tidy`
- Check imports are correct
- Verify all required methods implemented

## Next Steps

1. **Learn the patterns**: Study `internal/features/ping` and `internal/features/botinfo`
2. **Read guidelines**: `docs/CODING_GUIDELINES.md`
3. **Pick a feature**: Start with something simple
4. **Write requirements**: Document what it should do
5. **Use AI**: Give it requirements + template
6. **Test thoroughly**: `go test`, manual testing
7. **Deploy**: Register and run
8. **Iterate**: Add more features

## Tips for Success

✨ **Start Simple** - Ping and botinfo are intentionally simple examples  
✨ **Follow Patterns** - Copy structure from template features  
✨ **Use AI** - Give it requirements, not old code  
✨ **Test Early** - Write tests as you go  
✨ **Read Logs** - Structured logging helps debug  
✨ **Stay Clean** - Linter is your friend  

## Getting Help

- 📖 Check `docs/CODING_GUIDELINES.md` for standards
- 🔧 Check `docs/FEATURE_TEMPLATE.md` for patterns
- 📊 Check `docs/PROGRESS.md` for status
- 💡 Reference template features for examples

## You're Ready! 🎉

The foundation is solid. The patterns are established. The guidelines are clear.

**Start building your features and enjoy the clean code!**

```bash
# Let's go!
export DISCORD_BOT_TOKEN="your-token"
./bin/master
```

