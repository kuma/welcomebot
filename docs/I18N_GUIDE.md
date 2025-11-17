# Internationalization (i18n) Guide

## Overview

welcomebot bot supports **per-guild multilingual** capabilities. Each guild can set its preferred language.

## Supported Languages

- **English** (`en`) - Default
- **Japanese** (`ja`)

## Architecture

### Language Scope
**Per-Guild**: Each Discord server (guild) has its own language preference.

### Fallback Chain
1. Guild's configured language (from database)
2. English (default)

If a translation key is missing in Japanese → fallback to English

### Storage
```sql
-- Database
guild_languages table (guild_id → language_code)

-- Cache
Redis key: welcomebot:i18n:guild:{guild_id}
TTL: Indefinite (cached until changed)
```

---

## Usage in Code

### Basic Translation

```go
// In your feature
func (f *Feature) sendSuccessMessage(ctx context.Context, guildID, channelID string) error {
    title := f.i18n.T(ctx, guildID, "commands.room.created_title")
    desc := f.i18n.T(ctx, guildID, "commands.room.created_description")
    
    embed := &discordgo.MessageEmbed{
        Title:       title,
        Description: desc,
        Color:       int(shared.ColorSuccess),
    }
    
    return f.discord.SendEmbed(ctx, channelID, embed)
}
```

### Translation with Variables

```go
// Template with placeholders
message := f.i18n.TWithArgs(ctx, guildID, "commands.room.user_joined",
    map[string]string{
        "user": userName,
        "room": roomName,
    })

// en.json: "{user} joined {room}"
// ja.json: "{user}が{room}に参加しました"
// Result: "Alice joined Room 1" or "Aliceがルーム1に参加しました"
```

### Error Messages

```go
// Translate error messages too
errorMsg := f.i18n.T(ctx, guildID, "errors.permission_denied")
return fmt.Errorf("%s: %w", errorMsg, err)

// Or for user display
f.discord.SendMessage(ctx, channelID, discord.Message{
    Content: f.i18n.T(ctx, guildID, "errors.not_found"),
})
```

---

## Adding Translations

### Step 1: Add to English (en.json)

```json
{
    "commands": {
        "myfeature": {
            "title": "My Feature",
            "description": "This is my feature description",
            "success": "Operation completed successfully",
            "limit_reached": "Limit reached: {current}/{max}"
        }
    }
}
```

### Step 2: Add to Japanese (ja.json)

```json
{
    "commands": {
        "myfeature": {
            "title": "私の機能",
            "description": "これは私の機能の説明です",
            "success": "操作が正常に完了しました",
            "limit_reached": "上限に達しました: {current}/{max}"
        }
    }
}
```

### Step 3: Use in Code

```go
title := f.i18n.T(ctx, guildID, "commands.myfeature.title")
desc := f.i18n.T(ctx, guildID, "commands.myfeature.description")

// With variables
msg := f.i18n.TWithArgs(ctx, guildID, "commands.myfeature.limit_reached",
    map[string]string{
        "current": "5",
        "max": "10",
    })
```

---

## Translation Key Organization

### Naming Convention

Use dot notation with hierarchy:

```
{category}.{feature}.{message_type}
```

### Categories

| Category | Purpose | Example |
|----------|---------|---------|
| `commands.*` | Command responses | `commands.room.created` |
| `errors.*` | Error messages | `errors.permission_denied` |
| `common.*` | Common/shared text | `common.success` |
| `embeds.*` | Embed content | `embeds.help.title` |

### Examples

```
✅ commands.room.created_title
✅ commands.room.deleted_success
✅ errors.invalid_channel
✅ common.processing
✅ embeds.welcome.description

❌ room_created (no category)
❌ msg1 (not descriptive)
❌ CreateRoomSuccess (not snake_case)
```

---

## Checklist for Features

When implementing a feature with user-facing text:

- [ ] Add translations to `en.json`
- [ ] Add translations to `ja.json`
- [ ] Use `i18n.T()` for all user-facing strings
- [ ] Use `i18n.TWithArgs()` for dynamic content
- [ ] Test in both languages (if possible)
- [ ] No hardcoded strings in embeds/messages
- [ ] Error messages are translated
- [ ] Button labels are translated (if applicable)

---

## Managing Guild Language

### Set Language (Admin Only)

```
/set-language language:Japanese
```

Implementation:
```go
func (f *Feature) setLanguage(ctx context.Context, guildID, langCode string) error {
    // Permission check required
    if !f.checkAdminPermission(ctx, s, guildID, userID) {
        return fmt.Errorf("permission denied")
    }
    
    // Set in database
    return f.i18n.SetGuildLanguage(ctx, guildID, langCode)
}
```

### Get Available Languages

```go
langs := f.i18n.AvailableLanguages()
// Returns: ["en", "ja"]
```

---

## Translation File Structure

```
internal/core/i18n/translations/
├── en.json    # English (default, always complete)
├── ja.json    # Japanese
└── ...        # Future languages
```

### Full Structure Example

```json
{
    "commands": {
        "ping": { ... },
        "botinfo": { ... },
        "room": { ... },
        "admin": { ... }
    },
    "errors": {
        "permission_denied": "...",
        "not_found": "...",
        "invalid_config": "..."
    },
    "common": {
        "success": "Success",
        "error": "Error",
        "processing": "Processing..."
    },
    "embeds": {
        "help": {
            "title": "...",
            "description": "..."
        }
    }
}
```

---

## Best Practices

### 1. Always Provide Context
```go
// Good
"Room {name} created by {user}"

// Better  
"Room {name} has been created successfully by {user}"
```

### 2. Use Placeholders Consistently
```go
// Use {key} format
"Welcome {user} to {server}"

// Not mixed formats
"Welcome $user to %s"  // ❌
```

### 3. Keep Keys Organized
Group related translations:
```json
{
    "commands": {
        "room": {
            "created": "...",
            "deleted": "...",
            "updated": "..."
        }
    }
}
```

### 4. Complete Translations
If you add a key to `en.json`, add it to `ja.json` too (even if just copying English temporarily).

### 5. Test Both Languages
When possible, test your feature in both English and Japanese guilds.

---

## Common Patterns

### Success Message
```go
title := f.i18n.T(ctx, guildID, "common.success")
desc := f.i18n.T(ctx, guildID, "commands.room.created")
```

### Error Message
```go
errorTitle := f.i18n.T(ctx, guildID, "common.error")
errorDesc := f.i18n.T(ctx, guildID, "errors.permission_denied")
```

### Confirmation Message
```go
msg := f.i18n.TWithArgs(ctx, guildID, "commands.room.confirm_delete",
    map[string]string{
        "room": roomName,
    })
// "Are you sure you want to delete {room}?"
```

### List/Enumeration
```go
msg := f.i18n.TWithArgs(ctx, guildID, "commands.room.count",
    map[string]string{
        "count": strconv.Itoa(len(rooms)),
    })
// "Found {count} rooms"
```

---

## Adding a New Language

To add support for a new language (e.g., Korean):

1. **Create translation file**:
```bash
cp internal/core/i18n/translations/en.json \
   internal/core/i18n/translations/ko.json
```

2. **Translate content**:
Edit `ko.json` with Korean translations

3. **Update constants** (optional):
```go
// internal/shared/constants.go
const (
    LangEnglish  = "en"
    LangJapanese = "ja"
    LangKorean   = "ko"  // Add new language
)
```

4. **Restart bot**:
Translations are loaded at startup

5. **Test**:
```
/set-language language:Korean
```

---

## FAQ

**Q: What if translation key is missing?**  
A: Returns the key itself as fallback (e.g., "commands.room.missing_key")

**Q: Can users set their own language?**  
A: No, language is per-guild only (set by guild admins)

**Q: How often is guild language cached?**  
A: Indefinitely - only refreshes when changed via `/set-language`

**Q: What about command names/descriptions?**  
A: Discord doesn't support i18n for slash command names yet. Keep them in English.

**Q: Should log messages be translated?**  
A: No - logs are for developers, keep them in English

**Q: What about DM messages to users?**  
A: Use English default (no guild context in DMs)

---

## Migration for Existing Features

If you have features with hardcoded strings:

1. **Identify user-facing strings**
2. **Create translation keys**
3. **Add to en.json and ja.json**
4. **Replace hardcoded strings with i18n.T()**
5. **Test in both languages**

---

## Summary

✅ **All user-facing text → i18n**  
✅ **Per-guild language**  
✅ **Fallback to English**  
✅ **Cache indefinitely**  
✅ **Easy to add new languages**  

**Remember**: If a user sees it, it must be translated!

