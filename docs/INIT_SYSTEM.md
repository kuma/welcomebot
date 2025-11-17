# Guild Initialization System

## Overview

The initialization system ensures guilds complete required setup before using the bot. It uses a **validation-based approach** (no flags) and delegates to individual setup features.

---

## Architecture

### No Init Flags - Just Validation ✅

```go
// NOT THIS (flag-based, can get out of sync):
is_initialized BOOLEAN  ❌

// THIS (validation-based, always accurate):
func isGuildReady(guildID) bool {
    return hasLanguage(guildID)  // Check actual data
}
```

### Three-Feature Design

```
┌─────────────────────────────────────────┐
│   1. Menu Feature (Entry Point)         │
│                                          │
│   /menu → Check if ready?                │
│      ├─ YES → Show menu                 │
│      └─ NO  → Delegate to init          │
└───────────────┬─────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────┐
│   2. Init Feature (Orchestrator)         │
│                                          │
│   Check what's missing                   │
│   If language missing → Delegate         │
│   (Future: other settings)               │
└───────────────┬─────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────┐
│   3. Language Feature (Implementation)   │
│                                          │
│   Show buttons: [English] [日本語]      │
│   Save selection                         │
│   Return to menu                         │
└─────────────────────────────────────────┘
```

---

## Complete User Flow

### First Time Setup

```
Admin: /menu
   ↓
Menu: Checks if guild has language
      SELECT language_code FROM guild_languages WHERE guild_id = $1
      → Row not found
   ↓
Menu: Delegates to Init
   ↓
Init: Checks missing settings
      missing = ["language"]
   ↓
Init: Delegates to Language Feature
   ↓
Language: Shows bilingual welcome
   ┌────────────────────────────────────┐
   │ 👋 Welcome to welcomebot Bot!            │
   │ 👋 welcomebot Botへようこそ！            │
   │                                    │
   │ Choose your language               │
   │ 言語を選択してください              │
   │                                    │
   │  [🇺🇸 English]  [🇯🇵 日本語]     │
   └────────────────────────────────────┘
   ↓
Admin: Clicks [🇯🇵 日本語]
   ↓
Language: Saves to database
   INSERT INTO guild_languages (guild_id, language_code)
   VALUES ($1, 'ja')
   ON CONFLICT (guild_id) DO UPDATE SET language_code = 'ja'
   ↓
Language: Updates cache (indefinite TTL)
   SET welcomebot:i18n:guild:{guild_id} = "ja"
   ↓
Language: Responds in Japanese
   "✅ 言語を日本語に設定しました"
   ↓
(Future: Init checks other settings)
   ↓
Menu: All required settings present → Show menu
```

### Subsequent Usage

```
User: /menu
   ↓
Menu: Checks if guild has language
      → Found in cache: "ja"
   ↓
Menu: Shows menu in Japanese
   ┌────────────────────────────────────┐
   │ 🤖 welcomebot Bot - 機能メニュー         │
   │                                    │
   │ 🔧 設定                            │
   │  [🌐 言語設定]                     │
   │                                    │
   │ 📊 情報                            │
   │  [🏓 Ping] [ℹ️ Bot情報]          │
   └────────────────────────────────────┘
```

### Changing Language Later

```
Admin: /menu → Clicks [🌐 言語設定]
   ↓
Language: Shows same picker
   [🇺🇸 English]  [🇯🇵 日本語]
   ↓
Admin: Switches to English
   ↓
Language: Updates database and cache
   ↓
Language: "✅ Language set to English"
```

---

## Required Settings

### Current (v1)
1. **Language** (REQUIRED)
   - Check: Row exists in `guild_languages`
   - Feature: Language feature
   - Validation: `SELECT language_code FROM guild_languages WHERE guild_id = $1`

### Future Examples

2. **Timezone** (Optional)
   - Check: Row exists in `guild_timezones`
   - Feature: Timezone feature
   - Only shown if language complete

3. **Admin Role** (Optional)
   - Check: Row exists in `guild_admin_roles`
   - Feature: Admin feature
   - Only shown if language complete

---

## Concurrent Access Protection

### Problem
Two admins run `/menu` simultaneously on uninitialized guild.

### Solution: Database Atomicity

```go
// Both admins click language at same time
// Database uses ON CONFLICT:

INSERT INTO guild_languages (guild_id, language_code)
VALUES ($1, $2)
ON CONFLICT (guild_id) DO UPDATE SET language_code = $2
RETURNING guild_id;

// First admin's transaction commits → Wins
// Second admin's transaction sees row exists → Gets updated value
// Result: Both see success, last one wins (acceptable)
```

**Alternative: If strict "first wins" needed:**
```go
INSERT INTO guild_languages (guild_id, language_code)
VALUES ($1, $2)
ON CONFLICT (guild_id) DO NOTHING  -- Don't update
RETURNING guild_id;

// Returns row: First admin (wins)
// Returns nothing: Second admin (lost, show "already configured")
```

**Current implementation**: Uses UPDATE (last wins) for simplicity.

---

## Extensibility

### Adding a New Required Setting

**Example: Add timezone as required**

1. **Create timezone feature** (following template)

2. **Update init validation**:
```go
// internal/features/initialization/feature.go
func (f *Feature) CheckRequired(ctx, guildID) (bool, []string) {
    missing := []string{}
    
    if !f.hasLanguage(ctx, guildID) {
        missing = append(missing, "language")
    }
    
    // ADD THIS:
    if !f.hasTimezone(ctx, guildID) {
        missing = append(missing, "timezone")
    }
    
    return len(missing) == 0, missing
}

func (f *Feature) hasTimezone(ctx, guildID) bool {
    _, err := f.getTimezone(ctx, guildID)
    return err == nil
}
```

3. **Update init wizard**:
```go
func (f *Feature) StartInitWizard(ctx, s, i, missing) error {
    if contains(missing, "language") {
        return f.delegateToLanguage(ctx, s, i)
    }
    
    // ADD THIS:
    if contains(missing, "timezone") {
        return f.delegateToTimezone(ctx, s, i)
    }
    
    return nil
}
```

4. **Done!** No schema changes to init tables needed.

---

## Code Architecture

### Language Feature (Standalone)

```go
package language

// Can be called from:
// - Menu (user clicks "Set Language")
// - Init (during first-time setup)
// - Any other feature (if needed)

func (f *Feature) ShowLanguagePicker(ctx, s, i) error {
    // Shows [English] [日本語] buttons
    // Handles selection
    // Saves to database
    // Updates cache
    // Responds with success
}

func (f *Feature) GetMenuButton() *bot.MenuButton {
    return &bot.MenuButton{
        Label:     "🌐 Language Settings",
        CustomID:  "menu:language:setup",
        Category:  "configuration",
        AdminOnly: true,
    }
}
```

### Init Feature (Orchestrator)

```go
package initialization

// Doesn't implement UI!
// Just checks and delegates

func (f *Feature) CheckRequired(ctx, guildID) (bool, []string) {
    // Returns: (isReady, missingSettings)
    // Checks if language exists (and future settings)
}

func (f *Feature) StartInitWizard(ctx, s, i, missing) error {
    // Delegates to appropriate feature based on what's missing
    if contains(missing, "language") {
        return f.languageFeature.ShowLanguagePicker(ctx, s, i)
    }
    // Future: handle other settings
}

func (f *Feature) GetMenuButton() *bot.MenuButton {
    return nil  // Not in menu (automatic)
}
```

### Menu Feature (Entry Point)

```go
package menu

func (f *Feature) HandleInteraction(ctx, s, i) error {
    // Check if guild ready
    isReady, missing := f.init.CheckRequired(ctx, guildID)
    
    if !isReady {
        // Start init wizard
        return f.init.StartInitWizard(ctx, s, i, missing)
    }
    
    // Show menu
    return f.displayMenu(ctx, s, i)
}
```

---

## Benefits

✅ **No Flag Maintenance** - Just check if data exists  
✅ **Self-Healing** - Missing data auto-prompts to set  
✅ **Extensible** - Add new required settings easily  
✅ **Reusable** - Features work standalone and in init  
✅ **Concurrent-Safe** - Database transactions handle races  
✅ **Simple** - Validation logic is straightforward  

---

## Testing Guild Initialization

### Test Uninitialized Guild

```bash
# In test/dev environment:
# 1. Delete language for a guild
DELETE FROM guild_languages WHERE guild_id = 'TEST_GUILD_ID';

# 2. Run /menu as admin
# Expected: Language picker appears

# 3. Select language
# Expected: Language saved, menu appears

# 4. Run /menu again
# Expected: Menu appears immediately (no init)
```

### Test Language Change

```bash
# 1. Run /menu
# 2. Click "🌐 Language Settings"
# 3. Select different language
# Expected: Language updated, menu in new language
```

### Test Permissions

```bash
# 1. Regular user runs /menu on uninitialized guild
# Expected: "This server needs to be set up by an admin"

# 2. Admin runs /menu
# Expected: Init wizard starts
```

---

## Summary

**Init System Design:**

1. **Validation-Based** - No separate state flags
2. **Delegating** - Init doesn't implement UI, delegates to features
3. **Extensible** - Easy to add new required settings
4. **Concurrent-Safe** - Database handles race conditions
5. **User-Friendly** - Automatic, guides admins through setup

**Result**: Clean, maintainable initialization system that grows with your bot!

