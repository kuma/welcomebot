# Menu System Architecture

## Overview

welcomebot uses a **central menu system** for feature discovery and access. Users run `/menu` to see all available features organized by category.

---

## User Flow

```
User: /menu
   ↓
Bot: [Ephemeral message with categorized buttons]
   ╔═══════════════════════════════════╗
   ║  🤖 welcomebot Bot - Feature Menu       ║
   ╠═══════════════════════════════════╣
   ║  🔧 Configuration                 ║
   ║  ┌─────────────────────────────┐ ║
   ║  │ 🌐 Set Language             │ ║
   ║  │ 👑 Set Admin Role           │ ║
   ║  └─────────────────────────────┘ ║
   ║                                   ║
   ║  🏠 Channel Management            ║
   ║  ┌─────────────────────────────┐ ║
   ║  │ 🎤 Setup Room Creation      │ ║
   ║  │ 💬 Setup Welcome Message    │ ║
   ║  └─────────────────────────────┘ ║
   ║                                   ║
   ║  📊 Information                  ║
   ║  ┌─────────────────────────────┐ ║
   ║  │ ℹ️  Bot Info                │ ║
   ║  │ 🏓 Ping                     │ ║
   ║  └─────────────────────────────┘ ║
   ╚═══════════════════════════════════╝
   ↓
User clicks: "🎤 Setup Room Creation"
   ↓
Bot: [Starts feature-specific wizard]
   Step 1/3: Select trigger channel...
```

---

## Architecture Design

### Menu Feature Interface

Each feature can optionally register a menu button:

```go
type Feature interface {
    Name() string
    HandleInteraction(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error
    RegisterCommands() []*discordgo.ApplicationCommand
    
    // Optional: Register menu button
    GetMenuButton() *MenuButton  // Returns nil if no menu entry
}

type MenuButton struct {
    Label      string   // Button text: "🎤 Setup Room Creation"
    CustomID   string   // Button ID: "menu:rooms:setup"
    Category   string   // Category: "configuration", "management", "info"
    AdminOnly  bool     // If true, only admins see this button
}
```

### Menu Feature Implementation

```go
// internal/features/menu/feature.go
type Feature struct {
    registry *bot.Registry  // Access to all features
    logger   logger.Logger
    i18n     i18n.I18n
}

func (f *Feature) HandleInteraction(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
    if i.Type == discordgo.InteractionApplicationCommand {
        if i.ApplicationCommandData().Name == "menu" {
            return f.showMenu(ctx, s, i)
        }
    }
    
    // Handle menu button clicks
    if strings.HasPrefix(i.MessageComponentData().CustomID, "menu:") {
        return f.routeToFeature(ctx, s, i)
    }
    
    return bot.ErrNotHandled
}

func (f *Feature) showMenu(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
    guildID := i.GuildID
    userID := i.Member.User.ID
    
    // Check if user is admin (for showing admin buttons)
    isAdmin := f.checkAdminPermission(ctx, s, guildID, userID)
    
    // Collect menu buttons from all features
    buttons := f.collectMenuButtons(ctx, guildID, isAdmin)
    
    // Organize by category
    components := f.buildCategorizedMenu(ctx, guildID, buttons)
    
    // Show menu (ephemeral)
    return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content:    f.i18n.T(ctx, guildID, "menu.title"),
            Components: components,
            Flags:      discordgo.MessageFlagsEphemeral,  // Only user sees
        },
    })
}
```

---

## Permission Handling

### Public Menu, Permission-Checked Buttons

```go
// Menu is public - anyone can run /menu
// But buttons check permissions when clicked

User (Regular): /menu
Bot shows:
  ✅ 🏓 Ping            (no permission needed)
  ✅ ℹ️  Bot Info        (no permission needed)
  ❌ 🌐 Set Language    (admin only - hidden or disabled)
  ❌ 🎤 Setup Rooms     (admin only - hidden or disabled)

User (Admin): /menu
Bot shows:
  ✅ 🏓 Ping            (no permission needed)
  ✅ ℹ️  Bot Info        (no permission needed)
  ✅ 🌐 Set Language    (admin - shown!)
  ✅ 🎤 Setup Rooms     (admin - shown!)
```

**Implementation:**
```go
func (f *Feature) collectMenuButtons(ctx context.Context, guildID string, isAdmin bool) []*MenuButton {
    buttons := []*MenuButton{}
    
    for _, feature := range f.registry.GetAllFeatures() {
        menuBtn := feature.GetMenuButton()
        if menuBtn == nil {
            continue  // Feature doesn't have menu button
        }
        
        // Skip admin-only buttons if user is not admin
        if menuBtn.AdminOnly && !isAdmin {
            continue
        }
        
        buttons = append(buttons, menuBtn)
    }
    
    return buttons
}
```

---

## Concurrent Access - How It Works

### Scenario: 3 Users, Same Time

```
Time T0:
  User A: /menu
  User B: /menu
  User C: /menu
    ↓
  All three get SAME menu (ephemeral, only they see their own)

Time T1:
  User A clicks: "Setup Rooms" → CustomID: "menu:rooms:setup"
  User B clicks: "Set Language" → CustomID: "menu:language:setup"
  User C clicks: "Bot Info" → CustomID: "menu:info"
    ↓
  Three DIFFERENT interactions, routed to different features

Time T2:
  User A: [In room wizard] CustomID: "room_wizard:step2:cat_123"
  User B: [In language picker] CustomID: "lang:select:en"
  User C: [Sees bot info]
    ↓
  All isolated - no conflicts!
```

### Why No Conflicts:

**1. Ephemeral Messages**
- Each user sees only their own menu
- Messages don't interfere

**2. Unique Interaction IDs**
- Discord assigns unique `interaction.ID` to each
- Bot processes them independently

**3. CustomID Includes Context**
- `"room_wizard:step2:cat_123"` is User A's state
- `"room_wizard:step2:cat_456"` is User B's state
- Different CustomIDs = Different state chains

**4. No Shared State**
- Nothing in memory
- Nothing in Redis
- Each interaction is self-contained

---

## Categories

Features are organized by category:

### Category Types

| Category | Purpose | Permission |
|----------|---------|------------|
| `configuration` | Bot setup (language, admin roles) | Admin only |
| `management` | Channel/role management | Admin only |
| `information` | Bot info, help, stats | Public |
| `interactive` | Games, fun features | Public |
| `moderation` | Mod tools | Admin only |

**Features specify their category when registering menu button.**

---

## Code Pattern

### Feature Registering Menu Button

```go
// In your feature
func (f *Feature) GetMenuButton() *bot.MenuButton {
    return &bot.MenuButton{
        Label:     "🏠 Setup Room Creation",
        CustomID:  "menu:rooms:setup",
        Category:  "management",
        AdminOnly: true,  // Only admins see this
    }
}
```

### Menu Feature Routing

```go
func (m *MenuFeature) routeToFeature(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
    customID := i.MessageComponentData().CustomID
    // "menu:rooms:setup"
    
    // Extract feature name
    parts := strings.Split(customID, ":")
    featureName := parts[1]  // "rooms"
    
    // Update customID to trigger feature wizard
    // Modify interaction customID for feature to handle
    modifiedCustomID := strings.TrimPrefix(customID, "menu:")
    // Now: "rooms:setup"
    
    // Forward to appropriate feature
    // Features will see customID: "rooms:setup" and start their wizard
}
```

---

## Benefits

✅ **Discoverability** - Users find features easily  
✅ **Organized** - Features grouped by category  
✅ **Permission-Aware** - Admin features hidden from regular users  
✅ **Scalable** - Adding features automatically adds to menu  
✅ **Clean UX** - One command to rule them all  
✅ **Concurrent** - Multiple users can use simultaneously  
✅ **Ephemeral** - No channel spam  

---

## Example Flow: User Configures Room Creation

```
1. User: /menu
   Bot: [Shows menu, ephemeral]

2. User clicks: "🏠 Setup Room Creation"
   CustomID: "menu:rooms:setup"
   Bot: "Step 1/3: Select trigger channel"
        [Channel select menu]
        CustomID: "rooms:setup:step1"

3. User selects: #create-room-trigger
   Bot: "Step 2/3: Select category"
        [Category select menu]
        CustomID: "rooms:setup:step2:CHANNEL_ID"
                                    ^^^^^^^^^^
                                    State passed!

4. User selects: "Voice Rooms" category
   Bot: [Opens modal for room name]
        CustomID: "rooms:setup:step3:CHANNEL_ID:CATEGORY_ID"
                                    ^^^^^^^^^^^^^^^^^^^^^^^^
                                    All state passed!

5. User enters: "Room {number}"
   Bot: Parses CustomID to get channel + category
        Gets name from modal
        Saves config to database
        "✅ Room creation configured!"
```

**All stateless! No Redis needed! Multiple users can do this simultaneously!**

---

## Should I Implement?

I can create:
1. **Menu feature** (`internal/features/menu/`)
2. **Update bot.Feature interface** to include `GetMenuButton()`
3. **Update template** to show how features register menu buttons
4. **Documentation** for menu system

**Want me to proceed?**
