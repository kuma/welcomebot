# Feature: Gender Roles Configuration

## User-Facing Description
Configure which Discord roles represent male and female members in your server. This helps the bot recognize gender-based features.

## Commands/Interactions
- Accessible via: `/menu` → Admin → Configuration → Set Gender Roles
- Two-step wizard using Discord role pickers
- Admin only

## Flow

### First Time Setup
1. Admin: Clicks "🚻 Set Gender Roles" in menu
2. Bot: "Step 1/2: Select male role" [Role picker]
3. Admin: Selects @Boys role
4. Bot: "Step 2/2: Select female role" [Role picker]
5. Admin: Selects @Girls role
6. Bot: Validates (must be different)
7. Bot: Saves to database + cache
8. Bot: "✅ Gender roles configured"

### Updating Existing Configuration
1. Admin: Clicks "🚻 Set Gender Roles"
2. Bot: "⚠️ Gender roles already configured:
        Male: @Boys
        Female: @Girls
        
        Do you want to reconfigure?"
        [Yes, Reconfigure] [Cancel]
3a. Admin clicks [Cancel]: Returns to menu
3b. Admin clicks [Reconfigure]: Shows step 1 wizard

## Data Models

### Database
```sql
guild_gender_roles:
- guild_id (PK)
- male_role_id
- female_role_id
- created_at
- updated_at
```

### Cache
- Key: `welcomebot:gender:{guild_id}`
- Value: JSON {male_role_id, female_role_id}
- TTL: Indefinite (cached until changed)

## Business Logic

### Validation
- Male and female roles MUST be different
- If same role selected: Show error and restart wizard
- Both roles MUST exist in the guild

### Permissions
- Admin only (Discord Administrator OR "welcomebotbotadmin" OR custom admin role)
- Guild-isolated (each guild has own configuration)

### Storage
- Save to database (persistent)
- Cache in Redis (performance)
- Update both atomically

### Overwrite Protection
- If already configured: Show confirmation
- User must explicitly confirm reconfiguration
- Cancel returns to menu

## Examples

### Example 1: First Time Setup
```
Admin: [Clicks "🚻 Set Gender Roles"]
Bot: "Step 1/2: Select male role"
     [Discord role picker]
Admin: Selects @Boys
Bot: "Step 2/2: Select female role"
     [Discord role picker]
Admin: Selects @Girls
Bot: "✅ Gender roles configured!
      Male: @Boys
      Female: @Girls"
```

### Example 2: Update Configuration
```
Admin: [Clicks "🚻 Set Gender Roles"]
Bot: "⚠️ Current configuration:
      Male: @Boys
      Female: @Girls
      
      Reconfigure gender roles?"
     [Yes, Reconfigure] [Cancel]
Admin: [Clicks Yes]
Bot: [Shows step 1 wizard]
```

### Example 3: Validation Error
```
Admin: Step 1 → Selects @Members
Admin: Step 2 → Selects @Members (same!)
Bot: "❌ Error: Male and female roles must be different.
      Please start over."
     [← Back to Menu]
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
    Label: "🚻 Set Gender Roles"
    Category: "admin"
    SubCategory: "configuration"
    Tier: 3
    AdminOnly: true
}
```

### Stateless Wizard
```
Step 1: "gender:step1"
Step 2: "gender:step2:MALE_ROLE_ID" ← State passed
Confirm: "gender:confirm_overwrite"
```

### Database Schema
- guild_id indexed for fast lookups
- ON CONFLICT UPDATE for atomic updates
- Timestamps for audit trail


