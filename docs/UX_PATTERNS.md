# UX Patterns & Best Practices

## Overview

welcomebot bot uses **interactive, wizard-based UI** for a better user experience. This document defines the UX patterns all features should follow.

---

## Core UX Principles

### 1. Menu-Driven Discovery
✅ **DOMenuUsers discover features via `/menu`  
❌ **DON'TMenuExpect users to memorize slash commands

### 2. Step-by-Step Wizards
✅ **DOMenuGuide users through configuration with wizards  
❌ **DON'T**: Show complex forms with many fields at once

### 3. Ephemeral Feedback
✅ **DO**: Use ephemeral messages for admin operations  
❌ **DON'T**: Spam channels with config messages

### 4. Clear Visual Feedback
✅ **DO**: Use emojis, colors, and clear labels  
❌ **DON'T**: Show walls of text

### 5. Permission-Based UI
✅ **DO**: Hide admin features from regular users  
❌ **DON'T**: Show buttons that will fail permission check

---

## Standard Patterns

### Pattern 1: Menu Entry Point

**Every feature registers a menu button:**

```go
func (f *Feature) GetMenuButton() *bot.MenuButton {
    return &bot.MenuButton{
        Label:     "🏠 Setup Room Creation",
        CustomID:  "menu:rooms:setup",
        Category:  "management",
        AdminOnly: true,
    }
}
```

**User Flow:**
```
/menu → [Shows all features] → User clicks button → Feature wizard starts
```

---

### Pattern 2: Simple Selection (1 step)

For features needing one input:

```go
// Example: Language selection
User: /menu → Clicks "🌐 Set Language"
   ↓
Bot: [Select menu with languages]
     ┌──────────────┐
     │ English      │
     │ 日本語       │
     └──────────────┘
   ↓
User: Selects "日本語"
   ↓
Bot: "✅ Language set to Japanese"
```

**Code:**
```go
// Show language picker
components := []discordgo.MessageComponent{
    discordgo.ActionsRow{
        Components: []discordgo.MessageComponent{
            discordgo.SelectMenu{
                CustomID: "lang:select",
                Options: []discordgo.SelectMenuOption{
                    {Label: "English", Value: "en"},
                    {Label: "日本語", Value: "ja"},
                },
            },
        },
    },
}

// Handle selection
langCode := i.MessageComponentData().Values[0]
f.setLanguage(ctx, guildID, langCode)
```

---

### Pattern 3: Multi-Step Wizard (Stateless)

For features needing multiple inputs:

**Example: Room Creation (3 steps)**

```
Step 1: Select trigger channel
   CustomID: "rooms:step1"
   
Step 2: Select category (with channel in CustomID)
   CustomID: "rooms:step2:CHANNEL_ID"
   
Step 3: Enter name (with channel+category in CustomID)
   CustomID: "rooms:step3:CHANNEL_ID:CATEGORY_ID"
```

**Code Pattern:**
```go
// Step 1: Show channel picker
func (f *Feature) startWizard(s *discordgo.Session, i *discordgo.InteractionCreate) error {
    return showChannelPicker(s, i, "rooms:step1")
}

// Step 2: Channel selected, show category picker
func (f *Feature) handleStep1(s *discordgo.Session, i *discordgo.InteractionCreate) error {
    channelID := i.MessageComponentData().Values[0]  // From step 1
    
    // Pass channelID to next step via CustomID
    customID := fmt.Sprintf("rooms:step2:%s", channelID)
    return showCategoryPicker(s, i, customID)
}

// Step 3: Category selected, show modal
func (f *Feature) handleStep2(s *discordgo.Session, i *discordgo.InteractionCreate) error {
    // Parse previous state
    parts := strings.Split(i.MessageComponentData().CustomID, ":")
    channelID := parts[2]  // From step 1
    categoryID := i.MessageComponentData().Values[0]  // From step 2
    
    // Pass both to modal
    modalID := fmt.Sprintf("rooms:step3:%s:%s", channelID, categoryID)
    return showNameModal(s, i, modalID)
}

// Step 4: Modal submitted, create config
func (f *Feature) handleStep3(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
    // Parse all state from CustomID
    parts := strings.Split(i.ModalSubmitData().CustomID, ":")
    channelID := parts[2]
    categoryID := parts[3]
    
    // Get name from modal
    name := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).
        Components[0].(*discordgo.TextInput).Value
    
    // All values retrieved! No Redis needed!
    return f.saveConfig(ctx, i.GuildID, channelID, categoryID, name)
}
```

---

### Pattern 4: Confirmation Step

Always confirm destructive operations:

```go
// User clicks delete
CustomID: "rooms:delete:ROOM_ID"
   ↓
Bot: "⚠️ Are you sure you want to delete Room A?"
     [Confirm] [Cancel]
     CustomID: "rooms:confirm_delete:ROOM_ID"
     CustomID: "rooms:cancel"
   ↓
User clicks: [Confirm]
   ↓
Bot: Parse CustomID to get ROOM_ID
     Delete room
     "✅ Room deleted"
```

---

### Pattern 5: Discord Native Pickers (REQUIRED)

**ALWAYS use Discord's native pickers for channels, roles, and users.**

#### Rule: Use SelectMenu with MenuType

When your feature needs user input for:
- **Channels** → Use `ChannelSelectMenu`
- **Roles** → Use `RoleSelectMenu`
- **Users** → Use `UserSelectMenu`

**NEVER create custom select menus with manual options for these!**

#### Channel Picker

```go
// CORRECT: Native channel picker
discordgo.SelectMenu{
    MenuType: discordgo.ChannelSelectMenu,
    CustomID: "feature:step1:select",
    Placeholder: "Select a channel",
    ChannelTypes: []discordgo.ChannelType{
        discordgo.ChannelTypeGuildText,  // Filter to text channels
    },
    DefaultValues: []discordgo.SelectMenuDefaultValue{}, // Reset picker
}

// WRONG: Manual channel list
discordgo.SelectMenu{
    Options: []SelectMenuOption{
        {Label: "#general", Value: "123"},
        {Label: "#chat", Value: "456"},
    },
}  ❌ NO! Use ChannelSelectMenu instead!
```

#### Role Picker

```go
// CORRECT: Native role picker
discordgo.SelectMenu{
    MenuType: discordgo.RoleSelectMenu,
    CustomID: "feature:role:select",
    Placeholder: "Select a role",
    DefaultValues: []discordgo.SelectMenuDefaultValue{}, // Reset picker
}

// WRONG: Manual role list
discordgo.SelectMenu{
    Options: []SelectMenuOption{
        {Label: "@Admin", Value: "789"},
    },
}  ❌ NO! Use RoleSelectMenu instead!
```

#### User Picker

```go
// CORRECT: Native user picker
discordgo.SelectMenu{
    MenuType: discordgo.UserSelectMenu,
    CustomID: "feature:user:select",
    Placeholder: "Select a user",
    DefaultValues: []discordgo.SelectMenuDefaultValue{}, // Reset picker
}
```

**Benefits:**
- ✅ Built-in search
- ✅ Shows ALL items (no 25-item limit)
- ✅ Native Discord UI
- ✅ Type-safe (Discord validates)
- ✅ Auto-filtered by type
- ✅ Better UX

---

## UI Components

### Embeds for Information

```go
embed := &discordgo.MessageEmbed{
    Title:       f.i18n.T(ctx, guildID, "commands.room.title"),
    Description: f.i18n.T(ctx, guildID, "commands.room.description"),
    Color:       int(shared.ColorSuccess),
    Fields: []*discordgo.MessageEmbedField{
        {
            Name:   f.i18n.T(ctx, guildID, "common.status"),
            Value:  "Active",
            Inline: true,
        },
    },
    Timestamp: time.Now().Format(time.RFC3339),
}
```

### Buttons for Actions

```go
components := []discordgo.MessageComponent{
    discordgo.ActionsRow{
        Components: []discordgo.MessageComponent{
            discordgo.Button{
                Label:    f.i18n.T(ctx, guildID, "common.confirm"),
                Style:    discordgo.SuccessButton,  // Green
                CustomID: "action:confirm:ITEM_ID",
            },
            discordgo.Button{
                Label:    f.i18n.T(ctx, guildID, "common.cancel"),
                Style:    discordgo.DangerButton,   // Red
                CustomID: "action:cancel",
            },
        },
    },
}
```

### Select Menus for Choices

```go
discordgo.SelectMenu{
    CustomID:    "feature:select:step1",
    Placeholder: f.i18n.T(ctx, guildID, "common.choose"),
    MinValues:   1,
    MaxValues:   1,
    Options: []discordgo.SelectMenuOption{
        {
            Label:       "Option 1",
            Value:       "option_1",
            Description: "Description here",
            Emoji: &discordgo.ComponentEmoji{
                Name: "🎯",
            },
        },
    },
}
```

### Modals for Text Input

```go
modal := &discordgo.InteractionResponseData{
    CustomID: "feature:modal:PREVIOUS_STATE",
    Title:    f.i18n.T(ctx, guildID, "feature.modal_title"),
    Components: []discordgo.MessageComponent{
        discordgo.ActionsRow{
            Components: []discordgo.MessageComponent{
                discordgo.TextInput{
                    CustomID:    "input_name",
                    Label:       f.i18n.T(ctx, guildID, "feature.name_label"),
                    Style:       discordgo.TextInputShort,
                    Placeholder: f.i18n.T(ctx, guildID, "feature.name_placeholder"),
                    Required:    true,
                    MinLength:   1,
                    MaxLength:   100,
                },
            },
        },
    },
}
```

---

## Color Coding

Use consistent colors for feedback:

| Color | Usage | Value |
|-------|-------|-------|
| Blue (Default) | Information, neutral | `shared.ColorDefault` (0x00AAFF) |
| Green (Success) | Success messages | `shared.ColorSuccess` (0x2ECC71) |
| Yellow (Warning) | Warnings, cautions | `shared.ColorWarning` (0xFEE75C) |
| Red (Error) | Errors, destructive actions | `shared.ColorError` (0xED4245) |
| Blurple (Info) | Discord-style info | `shared.ColorInfo` (0x7289DA) |

---

## CustomID Best Practices

### Format Convention

```
{feature}:{action}:{state1}:{state2}...
```

**Examples:**
```
rooms:setup:step1
rooms:setup:step2:channel_123
rooms:delete:confirm:room_456
lang:select
admin:role:set:step2:role_789
```

### Encoding State

```go
// Simple IDs
customID := fmt.Sprintf("rooms:step2:%s", channelID)

// Multiple values
customID := fmt.Sprintf("rooms:step3:%s:%s", channelID, categoryID)

// With action
customID := fmt.Sprintf("rooms:delete:confirm:%s", roomID)
```

### Parsing State

```go
parts := strings.Split(customID, ":")
// parts[0] = feature name
// parts[1] = action/step
// parts[2+] = state values

feature := parts[0]
action := parts[1]
value1 := parts[2]  // if exists
value2 := parts[3]  // if exists
```

### Length Limit

CustomID max: **100 characters**

```go
// ✅ GOOD: 45 chars
"rooms:setup:step3:123456789012345678:987654321098765432"

// ❌ TOO LONG: Would exceed 100
"rooms:setup:step5:very_long_id_1:very_long_id_2:very_long_id_3:..."

// Solution: Use shorter IDs or Redis for complex state
```

---

## Error Handling UX

### Show Friendly Errors

```go
// Bad
return fmt.Errorf("invalid input")

// Good
errorEmbed := &discordgo.MessageEmbed{
    Title:       f.i18n.T(ctx, guildID, "common.error"),
    Description: f.i18n.T(ctx, guildID, "errors.invalid_channel"),
    Color:       int(shared.ColorError),
}
s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
    Type: discordgo.InteractionResponseChannelMessageWithSource,
    Data: &discordgo.InteractionResponseData{
        Embeds: []*discordgo.MessageEmbed{errorEmbed},
        Flags:  discordgo.MessageFlagsEphemeral,
    },
})
```

### Permission Denied

```go
if !hasPermission {
    embed := &discordgo.MessageEmbed{
        Title:       f.i18n.T(ctx, guildID, "common.error"),
        Description: f.i18n.T(ctx, guildID, "errors.permission_denied"),
        Color:       int(shared.ColorError),
    }
    // Show ephemeral
}
```

---

## Checklist for UX

When implementing a feature:

- [ ] Registers menu button (if applicable)
- [ ] Uses step-by-step wizard (not complex forms)
- [ ] Ephemeral messages for admin operations
- [ ] Clear visual feedback (embeds with colors)
- [ ] Translated strings (no hardcoded text)
- [ ] Permission checks before showing options
- [ ] Confirmation for destructive actions
- [ ] Uses Discord's built-in pickers when possible
- [ ] CustomID contains state (stateless)
- [ ] Error messages are user-friendly

---

## Summary

**The welcomebot UX Philosophy:**

1. **Discoverable** - `/menu` shows everything
2. **Guided** - Wizards walk users through steps
3. **Clean** - Ephemeral messages, no spam
4. **Visual** - Emojis, colors, embeds
5. **Translated** - Works in user's language
6. **Safe** - Permission-checked, confirmations
7. **Concurrent** - Stateless = multiple users OK

**Goal**: Make complex bot configuration feel simple and intuitive!

