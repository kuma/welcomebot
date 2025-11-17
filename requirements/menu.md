# Feature: Menu System

## User-Facing Description
Central hub for discovering and accessing all bot features through an interactive menu.

## Commands/Interactions
- `/menu`: Shows categorized list of all features
- Clicking buttons triggers respective features

## Flow
1. User runs `/menu`
2. Menu checks: Guild initialized? (has language?)
   - NO → Delegate to init feature → Language setup → Back to menu
   - YES → Show menu directly
3. Menu displays all features grouped by category
4. User clicks a button → Routed to that feature

## Data Models
None (reads from feature registry)

## Business Logic
- Ephemeral display (only user sees their menu)
- Public command (anyone can run `/menu`)
- Admin-only features hidden from regular users
- Categorized display (configuration, management, information, etc.)
- Permission-aware button visibility

## Categories
- configuration: Bot setup (language, admin roles)
- management: Channel/role management
- information: Bot info, help, stats
- interactive: Games, fun features
- moderation: Mod tools

## Examples

### Example 1: First Time (Needs Init)
```
Admin: /menu
Menu: Checks language → Missing
Init: Starts language wizard
User: Selects 日本語
Language: Saved
Menu: Shows full menu
```

### Example 2: Regular Use
```
User: /menu
Menu: Shows categorized features
      [Configuration] - Only admins see
      [Information] - Everyone sees
```

### Example 3: Admin vs Regular User
```
Admin sees:
  🔧 Configuration
    🌐 Language Settings
  📊 Information  
    🏓 Ping
    ℹ️ Bot Info

Regular user sees:
  📊 Information
    🏓 Ping
    ℹ️ Bot Info
```

## Technical Requirements
- Guild-aware
- Uses feature registry to collect menu buttons
- Filters admin-only buttons based on user permission
- Groups by category
- Delegates to init if not ready
- Routes button clicks to appropriate features

