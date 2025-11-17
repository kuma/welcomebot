# Feature: Self-Introduction Text Channels

## User-Facing Description
Configure separate text channels for male and female self-introductions in your server.

## Commands/Interactions
- Accessible via: `/menu` → Admin → Configuration → Set Self-Introduction TC
- Two-step wizard using channel select menus
- Admin only

## Flow

### First Time Setup
1. Admin: Clicks "📝 Set Self-Introduction TC" in menu
2. Bot: "Step 1/2: Select male self-introduction channel" [Channel select menu]
3. Admin: Selects #male-intro channel
4. Bot: "Step 2/2: Select female self-introduction channel" [Channel select menu]
5. Admin: Selects #female-intro channel
6. Bot: Validates (must be different)
7. Bot: Saves to database + cache
8. Bot: "✅ Self-introduction channels configured"

### Updating Existing Configuration
1. Admin: Clicks "📝 Set Self-Introduction TC"
2. Bot: "⚠️ Self-introduction channels already configured:
        Male: #male-intro
        Female: #female-intro
        
        Do you want to reconfigure?"
        [Yes, Reconfigure] [Cancel]
3a. Admin clicks [Cancel]: Returns to menu
3b. Admin clicks [Reconfigure]: Shows step 1 wizard

## Data Models

### Database
```sql
guild_selfintro_channels:
- guild_id (PK)
- male_channel_id
- female_channel_id
- created_at
- updated_at
```

### Cache
- Key: `welcomebot:selfintro:{guild_id}`
- Value: JSON {male_channel_id, female_channel_id}
- TTL: Indefinite (cached until changed)

## Business Logic

### Validation
- Male and female channels MUST be different
- If same channel selected: Show error and restart wizard
- Channels MUST be text channels
- Channels MUST exist in the guild

### Permissions
- Admin only (server owner OR Discord Administrator OR "welcomebotbotadmin" OR custom admin role)
- Guild-isolated (each guild has own configuration)

### Storage
- Save to database (persistent)
- Cache in Redis (performance, indefinite)
- Update both atomically

### Overwrite Protection
- If already configured: Show confirmation
- User must explicitly confirm reconfiguration
- Cancel returns to menu

## Examples

### Example 1: First Time Setup
```
Admin: [Clicks "📝 Set Self-Introduction TC"]
Bot: "Step 1/2: Select male self-introduction channel"
     [Channel select menu]
Admin: Selects #male-intro
Bot: "Step 2/2: Select female self-introduction channel"
     [Channel select menu]
Admin: Selects #female-intro
Bot: "✅ Self-introduction channels configured!
      Male: #male-intro
      Female: #female-intro"
```

### Example 2: Update Configuration
```
Admin: [Clicks "📝 Set Self-Introduction TC"]
Bot: "⚠️ Current configuration:
      Male: #male-intro
      Female: #female-intro
      
      Reconfigure self-introduction channels?"
     [Yes, Reconfigure] [Cancel]
Admin: [Clicks Yes]
Bot: [Shows step 1 wizard]
```

### Example 3: Validation Error
```
Admin: Step 1 → Selects #introductions
Admin: Step 2 → Selects #introductions (same!)
Bot: "❌ Error: Male and female channels must be different.
      Please try again."
```

## Technical Requirements

### Guild-Aware
- All queries filter by `guild_id`
- Cache key includes `guild_id`
- Functions accept `guildID` parameter

### i18n
- All messages translated (en, ja)
- Error messages translated
- Confirmation messages translated

### Menu Integration
```go
MenuButton{
    Label: "📝 Set Self-Introduction TC"
    Category: "admin"
    SubCategory: "configuration"
    Tier: 3
    AdminOnly: true
}
```

### Stateless Wizard
```
Step 1: "selfintro:step1"
Step 2: "selfintro:step2:MALE_CHANNEL_ID" ← State passed
Confirm: "selfintro:confirm_overwrite"
Cancel: "selfintro:cancel"
```

### Database Schema
- guild_id indexed for fast lookups
- ON CONFLICT UPDATE for atomic updates
- Timestamps for audit trail
- Only text channels allowed

