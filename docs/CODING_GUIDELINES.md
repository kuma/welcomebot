# welcomebot Bot - Coding Guidelines

**Version**: 1.0  
**Last Updated**: 2025-10-28

This document defines the **absolute rules** for AI-first development of the welcomebot Discord bot.

## Table of Contents

1. [Core Principles](#core-principles)
2. [Absolute Rules](#absolute-rules)
3. [Code Organization](#code-organization)
4. [Error Handling](#error-handling)
5. [Logging](#logging)
6. [Testing](#testing)
7. [Dependencies](#dependencies)
8. [Feature Development Workflow](#feature-development-workflow)

---

## Core Principles

### 1. No God Objects
Every feature is self-contained. No single struct should have more than 10 methods.

### 2. Explicit Dependencies
Never use `interface{}`. All dependencies must be explicitly typed.

### 3. Single Responsibility
Each file handles ONE concern. Each function does ONE thing.

### 4. Dependency Injection
All dependencies are passed via constructors, never globals.

### 5. Interface Segregation
Interfaces are small and focused (≤ 5 methods).

---

## Absolute Rules

### Rule 0: GUILD-AWARE - ALWAYS ⚠️ **CRITICAL**

**This bot serves MULTIPLE Discord guilds. ALL features MUST be guild-aware.**

❌ **WRONG:**
```go
func GetRoomConfig(channelID string) (*Config, error) {
    query := "SELECT * FROM configs WHERE channel_id = $1"
    // Missing guild_id - will mix guilds! ❌
}
```

✅ **CORRECT:**
```go
func GetRoomConfig(ctx context.Context, guildID, channelID string) (*Config, error) {
    query := "SELECT * FROM configs WHERE guild_id = $1 AND channel_id = $2"
    // Always filter by guild_id ✅
}
```

**Cache keys MUST include guild_id:**
```go
// WRONG
cacheKey := fmt.Sprintf("config:%s", channelID)

// CORRECT
cacheKey := fmt.Sprintf("welcomebot:config:%s:%s", guildID, channelID)
```

**📖 See `BOT_ARCHITECTURE.md` for complete guild-awareness requirements.**

### Rule 0.5: I18N - ALL USER-FACING TEXT ⚠️ **CRITICAL**

**All user-facing strings MUST be internationalized.**

❌ **WRONG:**
```go
embed := &discordgo.MessageEmbed{
    Title: "Room Created",
    Description: "Success!",
}
```

✅ **CORRECT:**
```go
title := f.i18n.T(ctx, guildID, "commands.room.created_title")
desc := f.i18n.T(ctx, guildID, "commands.room.created_description")
embed := &discordgo.MessageEmbed{
    Title: title,
    Description: desc,
}
```

**With variables:**
```go
msg := f.i18n.TWithArgs(ctx, guildID, "room.limit", map[string]string{
    "max": "10",
})
// English: "Room limit reached (max 10)"
// Japanese: "ルーム上限に達しました (最大 10)"
```

**📖 See `BOT_ARCHITECTURE.md` Rule 2 for complete i18n requirements.**

### Rule 1: NO `interface{}` - EVER ❌

**WRONG:**
```go
func DoSomething(data interface{}) error {
    if str, ok := data.(string); ok {
        return processString(str)
    }
    return errors.New("invalid type")
}
```

**CORRECT:**
```go
func DoSomething(data string) error {
    return processString(data)
}
```

**Exception**: Only allowed in generic data structures like `map[string]interface{}` for JSON unmarshaling, and must be immediately converted to concrete types.

### Rule 2: Functions ≤ 50 Lines

If a function exceeds 50 lines, split it into smaller functions.

**WRONG:**
```go
func ProcessRoom(config Config) error {
    // 100 lines of code...
}
```

**CORRECT:**
```go
func ProcessRoom(config Config) error {
    if err := validateConfig(config); err != nil {
        return err
    }
    room := buildRoom(config)
    return saveRoom(room)
}

func validateConfig(config Config) error { ... }
func buildRoom(config Config) Room { ... }
func saveRoom(room Room) error { ... }
```

### Rule 3: Files ≤ 300 Lines

Split large files by responsibility:
- `room_creation.go` - Creating rooms
- `room_deletion.go` - Deleting rooms
- `room_queries.go` - Querying rooms

### Rule 4: Explicit Error Handling

Never ignore errors.

**WRONG:**
```go
result, _ := db.Query(ctx, query)
```

**CORRECT:**
```go
result, err := db.Query(ctx, query)
if err != nil {
    return fmt.Errorf("query failed: %w", err)
}
```

### Rule 5: Always Use Context

All I/O operations must accept `context.Context` as the **first parameter**.

**WRONG:**
```go
func GetRoom(id string) (*Room, error)
```

**CORRECT:**
```go
// For guild-aware functions: ctx first, then guildID
func GetRoom(ctx context.Context, guildID, id string) (*Room, error)
```

**Note**: Guild-aware functions should have parameters in order: `ctx`, `guildID`, then other params.

### Rule 6: Constructor Pattern

All structs with dependencies must have a `New*` constructor.

**WRONG:**
```go
feature := &Feature{db: db, cache: cache}
```

**CORRECT:**
```go
func NewFeature(deps Dependencies) *Feature {
    return &Feature{
        db:    deps.DB,
        cache: deps.Cache,
    }
}

feature := NewFeature(deps)
```

### Rule 7: Struct Methods ≤ 10

If a struct has more than 10 methods, split it into multiple types.

**Example:**
```go
// WRONG: One struct with 20 methods
type RoomManager struct { ... }
func (r *RoomManager) CreateRoom() { ... }
func (r *RoomManager) DeleteRoom() { ... }
// ... 18 more methods

// CORRECT: Split into focused types
type RoomCreator struct { ... }
type RoomDeleter struct { ... }
type RoomQuery struct { ... }
```

### Rule 8: Package Documentation

Every package must have a `doc.go` file:

```go
// Package rooms provides Discord voice room management.
//
// It handles room creation, deletion, and lifecycle management
// for temporary and permanent voice channels.
package rooms
```

### Rule 9: Exported Symbol Documentation

All exported functions, types, and methods must have documentation.

```go
// CreateRoom creates a new voice room with the specified configuration.
// It validates the config, creates the Discord channel, and stores state.
//
// Returns an error if the category is invalid or creation fails.
func CreateRoom(ctx context.Context, config RoomConfig) (*Room, error) {
    // ...
}
```

### Rule 10: Test Files

For every `feature.go`, create `feature_test.go` in the same directory.

### Rule 11: Use Discord Native Pickers ⚠️

**When features need channel, role, or user input: ALWAYS use Discord's native pickers.**

```go
// CORRECT: Native pickers
discordgo.SelectMenu{
    MenuType: discordgo.ChannelSelectMenu,  // For channels
    // OR
    MenuType: discordgo.RoleSelectMenu,     // For roles
    // OR
    MenuType: discordgo.UserSelectMenu,     // For users
}

// WRONG: Manual options
discordgo.SelectMenu{
    Options: []SelectMenuOption{...}  ❌
}
```

**Benefits:** Built-in search, unlimited items, better UX.

**See UX_PATTERNS.md Pattern 5 for details.**

```
internal/features/rooms/
├── feature.go
├── feature_test.go
├── types.go
└── dependencies.go
```

---

## Code Organization

### Naming Conventions

#### Packages
- Lowercase, single word, no underscores
- ✅ `rooms`, `cache`, `discord`, `logger`
- ❌ `room_manager`, `discordHelpers`, `RoomMgr`

#### Files
- Lowercase with underscores for multi-word names
- ✅ `room_creation.go`, `slash_commands.go`, `doc.go`
- ❌ `roomCreation.go`, `SlashCommands.go`, `ROOM.go`

#### Types
- PascalCase (exported) or camelCase (unexported)
- ✅ `RoomConfig`, `UserProfile`, `internalState`
- ❌ `room_config`, `ROOM_CONFIG`

#### Functions/Methods
- PascalCase (exported) or camelCase (unexported)
- ✅ `CreateRoom`, `DeleteRoom`, `parseConfig`, `validateInput`
- ❌ `create_room`, `CREATE_ROOM`

#### Constants
- PascalCase or SCREAMING_SNAKE_CASE for groups
- ✅ `MaxRoomSize`, `DefaultTimeout`
- ✅ `REDIS_KEY_PREFIX` (for grouped constants)

### File Organization

Every Go file must follow this exact order:

```go
// 1. Package declaration with optional doc comment
// Package rooms handles voice room management.
package rooms

// 2. Imports (grouped: stdlib, external, internal)
import (
    "context"
    "errors"
    "fmt"
    
    "github.com/bwmarrin/discordgo"
    "github.com/sirupsen/logrus"
    
    "welcomebot/internal/core/database"
    "welcomebot/internal/core/logger"
)

// 3. Constants
const (
    MaxRoomSize = 100
    DefaultName = "Room"
)

// 4. Package-level variables (avoid if possible)
var (
    ErrRoomNotFound = errors.New("room not found")
)

// 5. Types
type Feature struct {
    db     database.Client
    logger logger.Logger
}

// 6. Constructor(s)
func New(deps Dependencies) *Feature {
    return &Feature{
        db:     deps.DB,
        logger: deps.Logger,
    }
}

// 7. Public methods (alphabetically sorted)
func (f *Feature) CreateRoom(ctx context.Context, cfg RoomConfig) error {
    // ...
}

func (f *Feature) DeleteRoom(ctx context.Context, id string) error {
    // ...
}

// 8. Private methods (alphabetically sorted)
func (f *Feature) parseConfig(cfg RoomConfig) (*parsedConfig, error) {
    // ...
}

func (f *Feature) validateConfig(cfg RoomConfig) error {
    // ...
}
```

### Dependency Injection Pattern

Each feature must have a `dependencies.go` file:

```go
package rooms

import (
    "errors"
    
    "welcomebot/internal/core/cache"
    "welcomebot/internal/core/database"
    "welcomebot/internal/core/discord"
    "welcomebot/internal/core/logger"
)

// Dependencies contains all required dependencies for the rooms feature.
type Dependencies struct {
    DB      database.Client
    Cache   cache.Client
    Discord discord.Helper
    Logger  logger.Logger
}

// Validate ensures all required dependencies are present.
func (d Dependencies) Validate() error {
    if d.DB == nil {
        return errors.New("database client is required")
    }
    if d.Cache == nil {
        return errors.New("cache client is required")
    }
    if d.Discord == nil {
        return errors.New("discord helper is required")
    }
    if d.Logger == nil {
        return errors.New("logger is required")
    }
    return nil
}
```

---

## Error Handling

### Error Wrapping

Always wrap errors with context using `fmt.Errorf` and `%w`:

```go
if err != nil {
    return fmt.Errorf("failed to create room: %w", err)
}
```

### Context in Errors

Include relevant identifiers in error messages:

```go
return fmt.Errorf("failed to delete room %s in guild %s: %w", roomID, guildID, err)
```

### Custom Errors

Define custom errors for domain logic:

```go
var (
    ErrRoomNotFound    = errors.New("room not found")
    ErrInvalidConfig   = errors.New("invalid configuration")
    ErrPermissionDenied = errors.New("permission denied")
)

// Use them
if room == nil {
    return ErrRoomNotFound
}
```

### Error Checking

Check errors immediately, no deferred checks:

**WRONG:**
```go
data, err := fetchData(ctx)
result := process(data)
if err != nil {
    return err
}
```

**CORRECT:**
```go
data, err := fetchData(ctx)
if err != nil {
    return fmt.Errorf("fetch data: %w", err)
}
result := process(data)
```

---

## Logging

### Structured Logging

Always use structured logging with key-value pairs:

**WRONG:**
```go
log.Printf("Room %s created by user %s", roomID, userID)
```

**CORRECT:**
```go
logger.Info("room created",
    "room_id", roomID,
    "user_id", userID,
    "guild_id", guildID,
)
```

### Log Levels

- **Debug**: Detailed diagnostic information
- **Info**: Important business events
- **Warn**: Recoverable issues
- **Error**: Errors requiring attention

```go
logger.Debug("processing configuration", "config", config)
logger.Info("room created successfully", "room_id", id)
logger.Warn("cache miss, fetching from database", "key", key)
logger.Error("failed to create room", "error", err, "room_id", id)
```

### Never Use Standard Library Log

❌ `log.Println`, `log.Printf`, `log.Fatal`  
✅ Use the injected `logger.Logger`

---

## Testing

### Test File Structure

```go
package rooms_test

import (
    "context"
    "testing"
    
    "welcomebot/internal/features/rooms"
)

func TestCreateRoom(t *testing.T) {
    // Arrange
    ctx := context.Background()
    deps := setupTestDeps(t)
    feature := rooms.New(deps)
    
    // Act
    err := feature.CreateRoom(ctx, testConfig)
    
    // Assert
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
}
```

### Table-Driven Tests

For multiple test cases:

```go
func TestValidateConfig(t *testing.T) {
    tests := []struct {
        name    string
        config  RoomConfig
        wantErr bool
    }{
        {"valid config", validConfig, false},
        {"empty name", emptyNameConfig, true},
        {"invalid size", invalidSizeConfig, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateConfig(tt.config)
            if (err != nil) != tt.wantErr {
                t.Errorf("want error: %v, got: %v", tt.wantErr, err)
            }
        })
    }
}
```

### Mocking

Create mock interfaces for testing:

```go
type MockDB struct {
    GetRoomFunc func(ctx context.Context, id string) (*Room, error)
}

func (m *MockDB) GetRoom(ctx context.Context, id string) (*Room, error) {
    if m.GetRoomFunc != nil {
        return m.GetRoomFunc(ctx, id)
    }
    return nil, nil
}
```

---

## Dependencies

### Core Services

All features have access to these core services:

#### database.Client
```go
type Client interface {
    Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
    Exec(ctx context.Context, query string, args ...interface{}) error
}
```

#### cache.Client
```go
type Client interface {
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key string, value string, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}
```

#### discord.Helper
```go
type Helper interface {
    CreateChannel(ctx context.Context, guildID string, cfg ChannelConfig) (string, error)
    DeleteChannel(ctx context.Context, channelID string) error
    SendMessage(ctx context.Context, channelID string, msg Message) error
}
```

#### logger.Logger
```go
type Logger interface {
    Debug(msg string, fields ...interface{})
    Info(msg string, fields ...interface{})
    Warn(msg string, fields ...interface{})
    Error(msg string, fields ...interface{})
}
```

---

## Feature Development Workflow

### Step 1: Create Requirements Document

Create `requirements/feature_name.md`:

```markdown
# Feature: Room Management

## User-Facing Description
Users can create temporary voice rooms by joining a trigger channel.

## Commands/Interactions
- Voice join on trigger channel → Creates new room
- Voice leave on empty room → Deletes room

## Data Models
- Room: id, guild_id, creator_id, created_at
- RoomConfig: trigger_channel_id, category_id, name_template

## Business Logic
- Rooms are deleted when empty for >5 minutes
- Max 10 rooms per guild
- Creator has admin permissions

## Examples
1. User joins "Create Room" → Bot creates "Room 1"
2. All users leave → Room deleted after 5 minutes
```

### Step 2: Create Feature Structure

```bash
mkdir -p internal/features/rooms
touch internal/features/rooms/{doc.go,dependencies.go,feature.go,types.go,feature_test.go}
```

### Step 3: Implement Following Guidelines

Use this prompt template:

```
I need to implement a Discord bot feature following strict guidelines.

Requirements:
[Paste requirements/feature_name.md here]

Coding Guidelines:
- No interface{} types
- Functions < 50 lines
- Explicit error handling with context
- Use context.Context as first parameter
- Constructor pattern for structs
- Structured logging

Available Dependencies:
- database.Client (PostgreSQL)
- cache.Client (Redis)
- discord.Helper (Discord API)
- logger.Logger (structured logging)

File Structure:
- doc.go: Package documentation
- dependencies.go: Dependency injection
- types.go: Domain types
- feature.go: Main implementation
- feature_test.go: Tests

Please implement this feature.
```

### Step 4: Test and Validate

```bash
# Run linter
golangci-lint run ./internal/features/rooms/...

# Run tests
go test ./internal/features/rooms/...

# Build to verify
go build ./cmd/master
```

---

## AI Prompt Templates

### For New Features

```
Context: Building a Discord bot feature in Go with clean architecture.

Feature Name: [Name]
Requirements: [Paste from requirements/*.md]

Constraints:
- No interface{} types
- Functions ≤ 50 lines, files ≤ 300 lines
- Explicit error handling with fmt.Errorf("context: %w", err)
- Use context.Context as first param for I/O
- Constructor pattern: func New(deps Dependencies) *Feature
- Structured logging: logger.Info("msg", "key", val)

Dependencies (injected):
- database.Client: PostgreSQL operations
- cache.Client: Redis operations
- discord.Helper: Discord API wrapper
- logger.Logger: Structured logging

Output Structure:
1. doc.go: Package documentation
2. dependencies.go: Dependencies struct with Validate()
3. types.go: Domain types and constants
4. feature.go: Main implementation
5. feature_test.go: Unit tests

Generate complete, production-ready code following these guidelines.
```

### For Bug Fixes

```
I need to fix a bug in the [feature] feature.

Issue: [Description]
File: [Path]
Current behavior: [What's wrong]
Expected behavior: [What should happen]

Guidelines:
- Maintain function length ≤ 50 lines
- Add proper error context
- Include test case for the fix
- Use structured logging

Provide the fix with explanation.
```

---

## Checklist for Code Review

Before committing, verify:

**Critical Rules:**
- [ ] All functions accept `guildID` parameter (guild-aware)
- [ ] All database queries filter by `guild_id`
- [ ] All cache keys include `guild_id`
- [ ] All user-facing text uses `i18n.T()`
- [ ] Translations added to both `en.json` and `ja.json`
- [ ] Channel/role/user inputs use native Discord pickers (MenuType)

**Code Quality:**
- [ ] No `interface{}` usage (except JSON unmarshaling)
- [ ] All functions ≤ 50 lines
- [ ] All files ≤ 300 lines
- [ ] All errors are checked and wrapped
- [ ] All I/O functions accept `context.Context`
- [ ] All structs have constructors
- [ ] Package has `doc.go`
- [ ] All exported symbols are documented
- [ ] Tests exist and pass
- [ ] `golangci-lint` passes with no warnings
- [ ] No global variables (except errors)
- [ ] Structured logging used (no `log.Println`)

---

## Questions?

When in doubt:
1. Check existing template features for patterns
2. Refer to this document
3. Keep it simple - simpler is better
4. Type-safe over flexible
5. Explicit over implicit

**Remember**: These guidelines exist to make AI-assisted development fast and error-free. Follow them strictly!

