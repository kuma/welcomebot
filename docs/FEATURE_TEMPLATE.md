# Feature Implementation Template

Use this template when creating a new feature.

## AI Prompt Template

⚠️ **IMPORTANT**: If feature uses Discord events (messages, voice, members), first determine:
**Is this event HIGH-FREQUENCY (10+ per second) or LOW-FREQUENCY (< 1 per minute)?**

This determines routing strategy (indexed vs filtered).

```
I need to implement a Discord bot feature following strict coding guidelines.

REQUIREMENTS:
[Paste from requirements/FEATURE_NAME.md]

EVENT FREQUENCY (if applicable):
[ ] High-frequency (10+ per second) → Use indexed routing
[ ] Low-frequency (< 1 per minute) → Use filtered routing
[ ] Not event-based → Skip

CRITICAL CODING GUIDELINES:
⚠️ GUILD-AWARE: All functions must accept guildID parameter (after ctx)
⚠️ GUILD-AWARE: All database queries MUST filter by guild_id
⚠️ GUILD-AWARE: All cache keys MUST include guild_id
⚠️ I18N: ALL user-facing text must use i18n.T(ctx, guildID, "key")
⚠️ I18N: Add translations to both en.json and ja.json
⚠️ NATIVE PICKERS: Use ChannelSelectMenu/RoleSelectMenu/UserSelectMenu (NOT manual options)
⚠️ ADMIN CHECK: Discord Administrator OR "welcomebotbotadmin" role OR custom role

- No interface{} types
- Functions ≤ 50 lines
- Files ≤ 300 lines
- Explicit error handling (fmt.Errorf with %w)
- Use context.Context as first param, guildID as second
- Constructor pattern: func New(deps Dependencies) (*Feature, error)
- Structured logging: logger.Info("msg", "key", val)
- Return bot.ErrNotHandled if interaction not handled

📖 See BOT_ARCHITECTURE.md for complete multi-guild requirements

AVAILABLE DEPENDENCIES (inject via Dependencies struct):
- database.Client: PostgreSQL operations
- cache.Client: Redis operations
- discord.Helper: Discord API wrapper
- logger.Logger: Structured logging
- i18n.I18n: Internationalization (REQUIRED for user-facing text)

REQUIRED FILES:
1. doc.go: Package documentation
2. dependencies.go: Dependencies struct with Validate()
3. feature.go: Main implementation
4. types.go: Domain types (if needed)
5. feature_test.go: Unit tests

INTERFACE TO IMPLEMENT:
```go
type Feature interface {
    Name() string
    HandleInteraction(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error
    RegisterCommands() []*discordgo.ApplicationCommand
    GetMenuButton() *MenuButton  // Optional: return nil if no menu entry
}

type MenuButton struct {
    Label     string  // "🏠 Setup Room Creation"
    CustomID  string  // "menu:rooms:setup" 
    Category  string  // "management", "configuration", "information", "interactive"
    AdminOnly bool    // Only admins see this button
}
```

OPTIONAL INTERFACES (implement if needed):
```go
// For message handling
type MessageFeature interface {
    Feature
    HandleMessage(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) error
}

// For reactions
type ReactionFeature interface {
    Feature
    HandleReactionAdd(ctx context.Context, s *discordgo.Session, r *discordgo.MessageReactionAdd) error
    HandleReactionRemove(ctx context.Context, s *discordgo.Session, r *discordgo.MessageReactionRemove) error
}

// For voice events
type VoiceFeature interface {
    Feature
    HandleVoiceStateUpdate(ctx context.Context, s *discordgo.Session, v *discordgo.VoiceStateUpdate) error
}

// For member events
type MemberFeature interface {
    Feature
    HandleMemberJoin(ctx context.Context, s *discordgo.Session, m *discordgo.GuildMemberAdd) error
    HandleMemberLeave(ctx context.Context, s *discordgo.Session, m *discordgo.GuildMemberRemove) error
}
```

TEMPLATE FEATURES TO REFERENCE:
- Simple command: internal/features/ping
- With state tracking: internal/features/botinfo

Generate complete, production-ready code following these patterns.
```

## File Templates

### doc.go
```go
// Package FEATURE_NAME provides [description].
//
// [Additional details about what this feature does]
package FEATURE_NAME
```

### dependencies.go
```go
package FEATURE_NAME

import (
    "errors"
    
    "welcomebot/internal/core/cache"
    "welcomebot/internal/core/database"
    "welcomebot/internal/core/discord"
    "welcomebot/internal/core/logger"
)

// Dependencies contains all required dependencies for the FEATURE_NAME feature.
type Dependencies struct {
    DB      database.Client  // Required if using database
    Cache   cache.Client     // Required if using cache
    Discord discord.Helper   // Required if creating channels/messages
    I18n    i18n.I18n        // Required if showing user-facing text
    Logger  logger.Logger    // Always required
}

// Validate ensures all required dependencies are present.
func (d Dependencies) Validate() error {
    if d.Logger == nil {
        return errors.New("logger is required")
    }
    // Add checks for other required dependencies
    return nil
}
```

### feature.go
```go
package FEATURE_NAME

import (
    "context"
    "fmt"
    
    "welcomebot/internal/bot"
    "welcomebot/internal/core/logger"
    
    "github.com/bwmarrin/discordgo"
)

const featureName = "FEATURE_NAME"

// Feature implements the FEATURE_NAME feature.
type Feature struct {
    db      database.Client
    cache   cache.Client
    discord discord.Helper
    i18n    i18n.I18n
    logger  logger.Logger
}

// New creates a new FEATURE_NAME feature.
func New(deps Dependencies) (*Feature, error) {
    if err := deps.Validate(); err != nil {
        return nil, fmt.Errorf("validate dependencies: %w", err)
    }
    
    return &Feature{
        db:      deps.DB,
        cache:   deps.Cache,
        discord: deps.Discord,
        i18n:    deps.I18n,
        logger:  deps.Logger,
    }, nil
}

// Name returns the feature name.
func (f *Feature) Name() string {
    return featureName
}

// HandleInteraction handles interactions for this feature.
// IMPORTANT: All operations must be guild-aware!
func (f *Feature) HandleInteraction(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
    // Return ErrNotHandled if not our interaction
    if !f.shouldHandle(i) {
        return bot.ErrNotHandled
    }
    
    // Get guild ID - REQUIRED for all operations
    guildID := i.GuildID
    if guildID == "" {
        return fmt.Errorf("guild ID required")
    }
    
    // Check admin permission
    if !f.checkAdminPermission(ctx, s, guildID, i.Member.User.ID) {
        return fmt.Errorf("permission denied")
    }
    
    // Handle the interaction with guild awareness
    // ...
    
    return nil
}

// checkAdminPermission verifies user has admin rights in this guild.
func (f *Feature) checkAdminPermission(ctx context.Context, s *discordgo.Session, guildID, userID string) bool {
    // Check Discord Administrator permission or "welcomebotbotadmin" role or custom role
    // See BOT_ARCHITECTURE.md for full implementation
    return true // Implement actual check
}

// RegisterCommands returns slash commands to register.
func (f *Feature) RegisterCommands() []*discordgo.ApplicationCommand {
    return []*discordgo.ApplicationCommand{
        {
            Name:        "command-name",
            Description: "Command description",
            Options: []*discordgo.ApplicationCommandOption{
                // Add options if needed
            },
        },
    }
}

// GetMenuButton returns the menu button for this feature.
// Return nil if feature should not appear in /menu.
func (f *Feature) GetMenuButton() *bot.MenuButton {
    return &bot.MenuButton{
        Label:     "🎯 Feature Name",           // Shown in menu
        CustomID:  "menu:FEATURE_NAME:setup",  // Triggers wizard
        Category:  "management",               // Category for grouping
        AdminOnly: true,                       // Only admins see it
    }
}

// shouldHandle checks if this feature should handle the interaction.
func (f *Feature) shouldHandle(i *discordgo.InteractionCreate) bool {
    // Implement check logic
    return false
}
```

### feature_test.go
```go
package FEATURE_NAME_test

import (
    "testing"
    
    "welcomebot/internal/core/logger"
    "welcomebot/internal/features/FEATURE_NAME"
)

func TestNew(t *testing.T) {
    log, err := logger.New(logger.DefaultConfig())
    if err != nil {
        t.Fatalf("failed to create logger: %v", err)
    }
    
    deps := FEATURE_NAME.Dependencies{
        Logger: log,
    }
    
    feature, err := FEATURE_NAME.New(deps)
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
    
    if feature == nil {
        t.Error("expected feature, got nil")
    }
}

func TestNew_MissingDependency(t *testing.T) {
    deps := FEATURE_NAME.Dependencies{}
    
    _, err := FEATURE_NAME.New(deps)
    if err == nil {
        t.Error("expected error for missing dependencies, got nil")
    }
}

func TestName(t *testing.T) {
    log, _ := logger.New(logger.DefaultConfig())
    feature, _ := FEATURE_NAME.New(FEATURE_NAME.Dependencies{Logger: log})
    
    name := feature.Name()
    if name != "FEATURE_NAME" {
        t.Errorf("expected name 'FEATURE_NAME', got '%s'", name)
    }
}

func TestRegisterCommands(t *testing.T) {
    log, _ := logger.New(logger.DefaultConfig())
    feature, _ := FEATURE_NAME.New(FEATURE_NAME.Dependencies{Logger: log})
    
    commands := feature.RegisterCommands()
    if len(commands) < 1 {
        t.Error("expected at least 1 command")
    }
}
```

## Registration Template

Add to `cmd/master/main.go`:

```go
// Import
import "welcomebot/internal/features/FEATURE_NAME"

// In main(), after creating deps
FEATURE_NAMEFeature, err := FEATURE_NAME.New(FEATURE_NAME.Dependencies{
    DB:      deps.DB,      // If needed
    Cache:   deps.Cache,   // If needed
    Discord: deps.Discord, // If needed
    Logger:  deps.Logger,  // Always needed
})
if err != nil {
    log.Fatalf("Failed to create FEATURE_NAME feature: %v", err)
}
if err := bot.Registry().Register(FEATURE_NAMEFeature); err != nil {
    log.Fatalf("Failed to register FEATURE_NAME feature: %v", err)
}
```

## Checklist

Before submitting a feature:

- [ ] Requirements document created
- [ ] `doc.go` with package documentation
- [ ] `dependencies.go` with Validate()
- [ ] `feature.go` with all required methods
- [ ] `feature_test.go` with tests
- [ ] All functions ≤ 50 lines
- [ ] All files ≤ 300 lines
- [ ] No `interface{}` usage
- [ ] All errors handled with context
- [ ] Context used for I/O operations
- [ ] Tests passing: `go test ./internal/features/FEATURE_NAME/...`
- [ ] Linter passing: `golangci-lint run ./internal/features/FEATURE_NAME/...`
- [ ] Build passing: `go build ./cmd/master`
- [ ] Feature registered in `cmd/master/main.go`
- [ ] Documentation complete

## Example Workflow

1. **Create requirement**:
```bash
cat > requirements/myfeature.md << 'EOF'
# Feature: My Feature
...
EOF
```

2. **Create directory**:
```bash
mkdir -p internal/features/myfeature
```

3. **Use AI** with prompt template above

4. **Test**:
```bash
go test ./internal/features/myfeature/...
```

5. **Register** in `cmd/master/main.go`

6. **Build**:
```bash
go build ./cmd/master
```

7. **Run** (with Discord token):
```bash
export DISCORD_BOT_TOKEN="your-token"
./master
```

Done! 🎉

