# Feature: Role Assignment Buttons

## User-Facing Description
Admins can create role assignment panels with buttons. Users click buttons to add/remove roles.

## Commands/Interactions
- `/setup-role-panel`: Admin command to create a role panel
  - Options:
    - `title`: Panel title (optional)
    - `description`: Panel description (optional)
- Modal: Configure roles and button labels
- Buttons: One button per role, toggle add/remove

## Data Models

### RolePanel
- `guild_id`: Discord guild ID
- `channel_id`: Channel where panel is posted
- `message_id`: Message ID of the panel
- `roles`: Array of role configurations
- `created_by`: User ID who created it
- `created_at`: Timestamp

### RoleConfig
- `role_id`: Discord role ID
- `button_label`: Button text
- `button_style`: Button color (1=blue, 2=gray, 3=green, 4=red)
- `emoji`: Optional emoji

## Business Logic
- Only admins can create panels
- Users can toggle roles on/off by clicking
- Max 5 roles per panel (Discord button limit per row)
- Panels are cached for performance
- Button interactions are persistent (survive bot restart)

## Examples

### Example 1: Admin Creates Panel
```
Admin: /setup-role-panel title:"Choose Your Roles" description:"Click to add/remove roles"
Bot: [Shows modal with fields for each role]
Admin: [Fills in roles]
Bot: [Posts panel in channel with buttons]
```

### Example 2: User Adds Role
```
[Panel shows buttons: "🎮 Gamer" "🎨 Artist" "🎵 Musician"]
User: [Clicks "🎮 Gamer" button]
Bot: "Added role: Gamer" (ephemeral message)
User: [Clicks "🎮 Gamer" button again]
Bot: "Removed role: Gamer" (ephemeral message)
```

## Technical Requirements
- Store panel config in database
- Cache active panels (30 minute TTL)
- Button custom IDs: `role_panel:{guild_id}:{role_id}`
- Admin permission check

