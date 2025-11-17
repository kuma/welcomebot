# welcomebot Bot - Architecture & Core Principles

**Last Updated**: 2025-10-28  
**Status**: MANDATORY - All features MUST follow these principles

---

## 🎯 Bot Purpose

welcomebot is a **multi-purpose Discord administrative bot** designed to work across multiple Discord servers (guilds) simultaneously.

---

## ⚠️ CRITICAL ARCHITECTURE RULES

These rules are **ABSOLUTE and MANDATORY**. Violating them will cause data corruption and cross-guild contamination.

### Rule 1: Multi-Guild Awareness (CRITICAL)

**Every feature MUST be guild-aware.**

### Rule 2: Multi-Lingual Support (CRITICAL)

**Every user-facing string MUST be internationalized.**

**Every feature MUST be guild-aware.**

#### What This Means:

✅ **DO:**
- ALL database queries MUST filter by `guild_id`
- ALL cache keys MUST include `guild_id`
- ALL configurations are per-guild (with rare exceptions)
- ALL data is isolated per guild

❌ **DON'T:**
- NEVER mix data between guilds
- NEVER assume single-guild operation
- NEVER use global state for guild-specific data
- NEVER cache without guild separation

#### Database Schema Pattern:

```sql
-- CORRECT: guild_id as indexed column
CREATE TABLE room_configs (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(20) NOT NULL,  -- Discord guild ID
    channel_id VARCHAR(20) NOT NULL,
    config JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    INDEX idx_guild (guild_id)      -- REQUIRED index
);

-- Always query with guild_id
SELECT * FROM room_configs WHERE guild_id = $1 AND channel_id = $2;
```

#### Cache Key Pattern:

```go
// CORRECT: Include guild_id in every cache key
const cacheKeyPattern = "welcomebot:feature:{guild_id}:{resource_id}"

// Example
cacheKey := fmt.Sprintf("welcomebot:rooms:%s:%s", guildID, channelID)
```

#### Code Pattern:

```go
// WRONG: Missing guild_id
func GetRoomConfig(channelID string) (*Config, error) {
    // This will mix guilds! ❌
}

// CORRECT: Always require guild_id
func GetRoomConfig(ctx context.Context, guildID, channelID string) (*Config, error) {
    // Query with both guild_id and channel_id ✅
    query := "SELECT * FROM configs WHERE guild_id = $1 AND channel_id = $2"
    // ...
}
```

---

### Rule 2: Multi-Lingual Support (CRITICAL)

**All user-facing strings MUST be translatable.**

#### What This Means:

✅ **DO:**
- ALL user-facing text goes through i18n
- ALL embed titles/descriptions are translated
- ALL error messages are translated
- ALL button labels are translated
- Store translation keys in code, not hardcoded strings

❌ **DON'T:**
- NEVER hardcode user-facing strings
- NEVER mix languages (use consistent language per guild)
- NEVER skip translation for "simple" messages
- NEVER assume English-only usage

#### Code Pattern:

```go
// WRONG: Hardcoded English
embed := &discordgo.MessageEmbed{
    Title: "Room Created",
    Description: "Your room has been created successfully",
}

// CORRECT: Use i18n
embed := &discordgo.MessageEmbed{
    Title: f.i18n.T(ctx, guildID, "commands.room.created_title"),
    Description: f.i18n.T(ctx, guildID, "commands.room.created_description"),
}

// CORRECT: With variables
message := f.i18n.TWithArgs(ctx, guildID, "commands.room.limit_reached", 
    map[string]string{
        "current": "5",
        "max": "10",
    })
// Returns: "Room limit reached (5/10)" or "ルーム上限に達しました (5/10)"
```

#### Translation File Structure:

```json
// internal/core/i18n/translations/en.json
{
    "commands": {
        "room": {
            "created_title": "Room Created",
            "created_description": "Your room has been created successfully",
            "limit_reached": "Room limit reached ({current}/{max})"
        }
    },
    "errors": {
        "permission_denied": "You don't have permission"
    }
}

// internal/core/i18n/translations/ja.json
{
    "commands": {
        "room": {
            "created_title": "ルーム作成完了",
            "created_description": "ルームが正常に作成されました",
            "limit_reached": "ルーム上限に達しました ({current}/{max})"
        }
    },
    "errors": {
        "permission_denied": "権限がありません"
    }
}
```

#### Language Configuration:

**Scope**: Per-guild (each guild can set its language)

**Command**: `/set-language language:Japanese`

**Storage**:
```sql
CREATE TABLE guild_languages (
    guild_id VARCHAR(20) PRIMARY KEY,
    language_code VARCHAR(5) NOT NULL DEFAULT 'en',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

**Fallback Chain**:
1. Guild's configured language (from database)
2. English (default)

**Supported Languages**:
- `en` - English (default)
- `ja` - Japanese

**Cache Strategy**:
- Guild language cached indefinitely (only changes when admin updates)
- Cache key: `welcomebot:i18n:guild:{guild_id}`

#### Usage in Code:

```go
// In feature struct
type Feature struct {
    db     database.Client
    cache  cache.Client
    i18n   i18n.I18n        // Inject i18n
    logger logger.Logger
}

// In feature methods
func (f *Feature) CreateRoom(ctx context.Context, guildID string, name string) error {
    // ... create room logic ...
    
    // Respond with translated message
    title := f.i18n.T(ctx, guildID, "commands.room.created_title")
    desc := f.i18n.T(ctx, guildID, "commands.room.created_description")
    
    embed := &discordgo.MessageEmbed{
        Title: title,
        Description: desc,
        Color: int(shared.ColorSuccess),
    }
    
    // Send embed
    return f.discord.SendEmbed(ctx, channelID, embed)
}
```

#### Adding New Translations:

When adding a new feature:

1. Add English translations to `en.json`
2. Add Japanese translations to `ja.json`
3. Use translation keys in code
4. Test in both languages

**Translation Key Naming**:
- Use dot notation: `category.subcategory.key`
- Group by feature: `commands.room.*`, `errors.*`, `common.*`
- Be descriptive: `room.created_success` not `room.msg1`

---

### Rule 3: Admin Permission Model

**Who can configure bot features in a guild:**

#### Default Permission Check:

A user can configure bot features if they meet **ANY** of these conditions:

1. ✅ User has Discord `Administrator` permission in that guild
2. ✅ User has a role named `"welcomebotbotadmin"` (hardcoded default)
3. ✅ User has the custom admin role (if configured for that guild)

#### Permission Check Flow:

```go
func HasAdminPermission(ctx context.Context, guildID, userID string) (bool, error) {
    // 1. Check Discord Administrator permission
    if hasDiscordAdminPerm(guildID, userID) {
        return true, nil
    }
    
    // 2. Check for "welcomebotbotadmin" role (hardcoded default)
    if hasRole(guildID, userID, "welcomebotbotadmin") {
        return true, nil
    }
    
    // 3. Check for custom admin role (from database)
    customRole, err := getGuildAdminRole(ctx, guildID)
    if err == nil && customRole != "" {
        if hasRole(guildID, userID, customRole) {
            return true, nil
        }
    }
    
    return false, nil
}
```

#### Admin Role Configuration:

Guilds can customize their admin role:

```
/set-admin-role role:@ServerMods
  → Stores in database: guild_id → "ServerMods"
  → Now users with "ServerMods" role can configure bot

/delete-admin-role
  → Removes custom role
  → Falls back to "welcomebotbotadmin" + Discord Administrator
```

**Database Schema:**

```sql
CREATE TABLE guild_admin_roles (
    guild_id VARCHAR(20) PRIMARY KEY,
    role_name VARCHAR(100) NOT NULL,
    created_by VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

---

### Rule 3: Configuration Hierarchy

Most bot configurations are **per-guild**, with rare global exceptions.

#### Per-Guild Configurations (99% of features):

✅ Feature settings
✅ Channel configurations  
✅ Role assignments
✅ Custom messages
✅ Schedules and timers
✅ Feature enable/disable

**Pattern:**
```sql
CREATE TABLE feature_configs (
    guild_id VARCHAR(20) NOT NULL,  -- Per-guild
    config_key VARCHAR(100) NOT NULL,
    config_value JSONB,
    PRIMARY KEY (guild_id, config_key)
);
```

#### Global Configurations (Bot Owner Only):

⚠️ Very rare, only for bot-level settings:
- Bot status message
- Global rate limits
- Feature flags (enable/disable features globally)
- Bot owner whitelist

**Pattern:**
```sql
CREATE TABLE global_configs (
    config_key VARCHAR(100) PRIMARY KEY,  -- No guild_id
    config_value JSONB,
    owner_only BOOLEAN DEFAULT true
);
```

---

## 📋 Implementation Checklist

When implementing ANY feature, verify:

### Guild Awareness Checklist:

- [ ] All database tables have `guild_id` column
- [ ] All database queries filter by `guild_id`
- [ ] `guild_id` column is indexed
- [ ] All cache keys include `guild_id`
- [ ] Feature config is stored per-guild
- [ ] No global state for guild-specific data

### Permission Checklist:

- [ ] Admin commands check permissions
- [ ] Permission check includes Discord Administrator
- [ ] Permission check includes "welcomebotbotadmin" role
- [ ] Permission check includes custom admin role (from DB)
- [ ] Permission errors are user-friendly

### Code Review Checklist:

- [ ] All functions that query data accept `guildID` parameter
- [ ] `guildID` is always the first or second parameter (after `ctx`)
- [ ] Cache keys follow pattern: `"welcomebot:{feature}:{guild_id}:{resource_id}"`
- [ ] Logs include `guild_id` for debugging
- [ ] Tests verify guild isolation

---

## 🚫 Common Anti-Patterns (DON'T DO THIS)

### ❌ Anti-Pattern 1: Missing Guild Filter

```go
// WRONG: Will return data from ALL guilds
func GetAllRooms(ctx context.Context) ([]Room, error) {
    query := "SELECT * FROM rooms"  // Missing WHERE guild_id = $1
    // ...
}

// CORRECT: Always filter by guild
func GetAllRooms(ctx context.Context, guildID string) ([]Room, error) {
    query := "SELECT * FROM rooms WHERE guild_id = $1"
    // ...
}
```

### ❌ Anti-Pattern 2: Guild-less Cache Key

```go
// WRONG: Will mix guilds in cache
cacheKey := fmt.Sprintf("room:%s", channelID)

// CORRECT: Include guild_id
cacheKey := fmt.Sprintf("room:%s:%s", guildID, channelID)
```

### ❌ Anti-Pattern 3: Global Configuration for Guild Data

```go
// WRONG: Single config for all guilds
var globalRoomLimit = 10

// CORRECT: Per-guild configuration
func GetRoomLimit(ctx context.Context, guildID string) (int, error) {
    // Query from guild_configs table
}
```

### ❌ Anti-Pattern 4: Assuming Single Guild

```go
// WRONG: Assumes bot is only in one guild
func CleanupOldRooms(ctx context.Context) error {
    query := "DELETE FROM rooms WHERE created_at < NOW() - INTERVAL '7 days'"
    // This deletes across ALL guilds without checking! ❌
}

// CORRECT: Clean per guild
func CleanupOldRooms(ctx context.Context, guildID string) error {
    query := "DELETE FROM rooms WHERE guild_id = $1 AND created_at < NOW() - INTERVAL '7 days'"
    // ...
}
```

---

## 📝 Code Examples

### Example 1: Guild-Aware Feature

```go
package rooms

import "context"

type Feature struct {
    db     database.Client
    cache  cache.Client
    logger logger.Logger
}

// CreateRoom is guild-aware
func (f *Feature) CreateRoom(ctx context.Context, guildID, categoryID, name string) (*Room, error) {
    // 1. Validate guild-specific limits
    limit, err := f.getRoomLimit(ctx, guildID)
    if err != nil {
        return nil, err
    }
    
    currentCount, err := f.countRooms(ctx, guildID)
    if err != nil {
        return nil, err
    }
    
    if currentCount >= limit {
        return nil, ErrRoomLimitReached
    }
    
    // 2. Create room
    room := &Room{
        GuildID:    guildID,  // Always include guild_id
        CategoryID: categoryID,
        Name:       name,
        CreatedAt:  time.Now(),
    }
    
    // 3. Save with guild_id
    query := "INSERT INTO rooms (guild_id, category_id, name, created_at) VALUES ($1, $2, $3, $4) RETURNING id"
    err = f.db.QueryRow(ctx, query, guildID, categoryID, name, room.CreatedAt).Scan(&room.ID)
    if err != nil {
        return nil, fmt.Errorf("create room: %w", err)
    }
    
    // 4. Cache with guild_id in key
    cacheKey := fmt.Sprintf("welcomebot:rooms:%s:%s", guildID, room.ID)
    f.cache.SetJSON(ctx, cacheKey, room, 30*time.Minute)
    
    f.logger.Info("room created",
        "guild_id", guildID,  // Always log guild_id
        "room_id", room.ID,
    )
    
    return room, nil
}

// getRoomLimit gets per-guild configuration
func (f *Feature) getRoomLimit(ctx context.Context, guildID string) (int, error) {
    query := "SELECT room_limit FROM guild_configs WHERE guild_id = $1"
    var limit int
    err := f.db.QueryRow(ctx, query, guildID).Scan(&limit)
    if err != nil {
        return 10, nil // Default if not configured
    }
    return limit, nil
}

// countRooms counts rooms in a specific guild
func (f *Feature) countRooms(ctx context.Context, guildID string) (int, error) {
    query := "SELECT COUNT(*) FROM rooms WHERE guild_id = $1"  // Always filter by guild
    var count int
    err := f.db.QueryRow(ctx, query, guildID).Scan(&count)
    return count, err
}
```

### Example 2: Permission Check

```go
package admin

// CheckAdminPermission verifies if user can configure bot
func (f *Feature) CheckAdminPermission(ctx context.Context, s *discordgo.Session, guildID, userID string) (bool, error) {
    // 1. Check Discord Administrator permission
    member, err := s.GuildMember(guildID, userID)
    if err != nil {
        return false, fmt.Errorf("get guild member: %w", err)
    }
    
    // Get guild roles
    guild, err := s.Guild(guildID)
    if err != nil {
        return false, fmt.Errorf("get guild: %w", err)
    }
    
    // Check if user has Administrator permission
    for _, roleID := range member.Roles {
        for _, guildRole := range guild.Roles {
            if guildRole.ID == roleID {
                if guildRole.Permissions&discordgo.PermissionAdministrator != 0 {
                    return true, nil // Has Discord admin permission
                }
            }
        }
    }
    
    // 2. Check for hardcoded "welcomebotbotadmin" role
    if f.hasRole(member, guild, "welcomebotbotadmin") {
        return true, nil
    }
    
    // 3. Check for custom admin role (from database)
    customRole, err := f.getCustomAdminRole(ctx, guildID)
    if err == nil && customRole != "" {
        if f.hasRole(member, guild, customRole) {
            return true, nil
        }
    }
    
    return false, nil
}

func (f *Feature) hasRole(member *discordgo.Member, guild *discordgo.Guild, roleName string) bool {
    for _, roleID := range member.Roles {
        for _, guildRole := range guild.Roles {
            if guildRole.ID == roleID && guildRole.Name == roleName {
                return true
            }
        }
    }
    return false
}

func (f *Feature) getCustomAdminRole(ctx context.Context, guildID string) (string, error) {
    query := "SELECT role_name FROM guild_admin_roles WHERE guild_id = $1"
    var roleName string
    err := f.db.QueryRow(ctx, query, guildID).Scan(&roleName)
    if err != nil {
        return "", err
    }
    return roleName, nil
}
```

---

## 🧪 Testing Guild Isolation

Every feature MUST have tests that verify guild isolation:

```go
func TestGuildIsolation(t *testing.T) {
    ctx := context.Background()
    
    // Create rooms in two different guilds
    room1, _ := feature.CreateRoom(ctx, "guild-1", "cat-1", "Room A")
    room2, _ := feature.CreateRoom(ctx, "guild-2", "cat-2", "Room B")
    
    // Verify guild-1 can only see its own rooms
    rooms1, _ := feature.GetAllRooms(ctx, "guild-1")
    if len(rooms1) != 1 || rooms1[0].ID != room1.ID {
        t.Error("Guild-1 should only see its own rooms")
    }
    
    // Verify guild-2 can only see its own rooms
    rooms2, _ := feature.GetAllRooms(ctx, "guild-2")
    if len(rooms2) != 1 || rooms2[0].ID != room2.ID {
        t.Error("Guild-2 should only see its own rooms")
    }
    
    // Verify cross-guild access is denied
    room, _ := feature.GetRoom(ctx, "guild-1", room2.ID)
    if room != nil {
        t.Error("Guild-1 should NOT see guild-2's rooms")
    }
}
```

---

## 📚 Summary

### The Six Commandments:

1. **THOU SHALL BE GUILD-AWARE** - Every feature, every query, every cache key
2. **THOU SHALL INTERNATIONALIZE** - Every user-facing string through i18n
3. **THOU SHALL CHECK PERMISSIONS** - Discord admin OR "welcomebotbotadmin" OR custom role
4. **THOU SHALL NOT MIX GUILDS** - Data isolation is sacred
5. **THOU SHALL USE MENU SYSTEM** - Register features in central menu for discoverability
6. **THOU SHALL DETERMINE EVENT FREQUENCY** - Before implementing event features, ask: high or low frequency?

### Quick Reference:

| Aspect | Rule |
|--------|------|
| Database queries | Always filter by `guild_id` |
| Cache keys | Always include `guild_id` |
| Function signatures | Accept `guildID` parameter |
| User-facing text | Use `i18n.T(ctx, guildID, key)` |
| Configurations | Per-guild (99% of the time) |
| Admin permission | Discord admin OR role-based |
| Testing | Verify guild isolation + i18n |

---

### Rule 4: Menu System & UX Pattern

**All features should be discoverable through `/menu` command.**

#### Menu System Design:

**User Flow:**
```
/menu → Shows categorized buttons → Click button → Start feature wizard
```

**Benefits:**
- ✅ Feature discoverability (users find features easily)
- ✅ Clean UX (one command for everything)
- ✅ Permission-aware (admin features hidden from regular users)
- ✅ Organized (features grouped by category)
- ✅ Ephemeral (no channel spam)

#### Feature Menu Registration:

```go
// Each feature optionally provides a menu button
type Feature interface {
    Name() string
    HandleInteraction(...)
    RegisterCommands()
    GetMenuButton() *MenuButton  // Optional, return nil if no menu entry
}

type MenuButton struct {
    Label     string  // "🏠 Setup Room Creation"
    CustomID  string  // "menu:rooms:setup"
    Category  string  // "management", "configuration", "information", "interactive"
    AdminOnly bool    // Only show to admins
}

// Example implementation
func (f *RoomFeature) GetMenuButton() *bot.MenuButton {
    return &bot.MenuButton{
        Label:     "🏠 Setup Room Creation",
        CustomID:  "menu:rooms:setup",
        Category:  "management",
        AdminOnly: true,
    }
}
```

#### Categories:

| Category | Purpose | Examples |
|----------|---------|----------|
| `configuration` | Bot setup | Language, admin roles |
| `management` | Channel/role management | Rooms, roles, welcome |
| `information` | Info/stats | Bot info, ping, help |
| `interactive` | Games/fun | Games, polls |
| `moderation` | Mod tools | Cleanup, moderation |

#### Concurrent Access:

**Menu system is stateless and concurrent-safe:**
- Multiple users can run `/menu` simultaneously
- Each sees their own ephemeral menu
- Button clicks are isolated per user
- CustomID carries state (no shared storage)

#### Stateless Wizard Pattern:

```go
// Step 1: User clicks menu button
CustomID: "menu:rooms:setup"

// Step 2: Feature shows first step
CustomID: "rooms:setup:step1"

// Step 3: User makes selection (e.g., channel_123)
CustomID: "rooms:setup:step2:channel_123"
                               ^^^^^^^^^^
                               State encoded!

// Step 4: User submits modal
CustomID: "rooms:setup:step3:channel_123:category_456"
                               ^^^^^^^^^^^^^^^^^^^^^^^
                               Full state in CustomID!
```

**See `MENU_SYSTEM.md` for complete implementation details.**

---

**Remember**: This bot serves MULTIPLE guilds. Data leakage between guilds is a **critical bug**. When in doubt, always filter by `guild_id`.

