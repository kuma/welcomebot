# Feature: Guild Initialization

## User-Facing Description
Automatically guides admins through required setup when they first use the bot.

## Commands/Interactions
- Triggered automatically from `/menu` if guild not initialized
- No direct command (invisible orchestrator)
- Delegates to other setup features

## Flow
1. Admin runs `/menu` for first time
2. Init feature checks: What's missing?
3. If language missing → Delegate to language feature
4. (Future: If other settings missing → Delegate to those features)
5. After all required settings complete → Show menu

## Data Models

### No Dedicated Table
- Checks existence of required data:
  - Language: `SELECT FROM guild_languages WHERE guild_id = $1`
  - (Future: Check other required settings)

## Business Logic
- Admin permission required
- Checks required settings every time `/menu` runs
- Delegates to specific features (doesn't implement UI itself)
- Sequential: Complete one setting before showing next
- Concurrent protection: First admin wins (database transaction)

## Required Settings (Current)
1. Language (REQUIRED) - delegates to language feature

## Required Settings (Future Examples)
- Timezone (optional)
- Welcome channel (optional)
- Admin role (optional)

## Examples

### Example 1: First Time Setup
```
Admin: /menu
Init checks: Language configured? → NO
Init: Delegate to language feature
Language feature: Shows [English] [日本語]
Admin: Clicks 日本語
Language feature: Saves to database
Init checks: All required settings? → YES
Init: Done, show menu
Menu: Shows all features
```

### Example 2: Already Initialized
```
Admin: /menu
Init checks: Language configured? → YES
Init: All good, skip
Menu: Shows all features immediately
```

### Example 3: Concurrent Init (Two Admins)
```
Admin A: /menu (first)
Init: Checks language → Missing
Language feature: Shows buttons to Admin A

Admin B: /menu (2 seconds later)
Init: Checks language → Still missing (Admin A not done)
Init: Shows "⚠️ Setup in progress by another admin"
```

## Technical Requirements
- Guild-aware
- No separate init state table
- Uses database transaction for concurrent protection
- Extensible (easy to add new required settings)
- Delegates to other features (no UI implementation)

