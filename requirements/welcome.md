# Feature: Welcome Onboarding Bot

## Overview

The Welcome Onboarding Bot provides a voice-based interactive onboarding experience for new members joining a Discord guild. The system uses a master-slave architecture where:

- **1 Master Bot**: Handles configuration, button interactions, and slave coordination
- **3 Slave Bots**: Join voice channels and conduct actual onboarding sessions

## Architecture

### Master Bot Responsibilities
- Admin configuration of welcome channel and VC category
- Posting persistent "Start Onboarding" button
- Tracking slave availability
- Assigning users to available slaves via task queue
- Monitoring slave health via heartbeats

### Slave Bot Responsibilities
- Creating temporary voice channels
- Joining voice channels
- Playing audio files
- Showing interactive buttons
- Managing user roles (in-progress, completed)
- Cleaning up resources after session

## User Flow

1. **User Clicks Button**: User clicks "Start Onboarding" in welcome channel
2. **Slave Assignment**: Master checks for available slave
   - If available: Creates task and assigns to slave
   - If busy: Shows "All bots busy, try again later"
3. **VC Creation**: Assigned slave creates private voice channel in configured category
4. **Voice Onboarding**: Slave joins VC, plays audio, shows buttons
5. **Interactive Steps**: User progresses through onboarding steps
6. **Completion**: Slave adds completion role, removes in-progress role
7. **Cleanup**: VC is automatically deleted

## Admin Configuration

### `/menu` → Admin → Configuration → Welcome Onboarding

**Step 1**: Select welcome text channel
- Where the "Start Onboarding" button will appear

**Step 2**: Select VC category
- Where temporary onboarding voice channels will be created

### Future Configuration (Phase 2+)
- In-progress role selection
- Completed role selection
- Audio file customization
- Custom onboarding flow steps

## Database Schema

```sql
CREATE TABLE guild_welcome_config (
    guild_id VARCHAR(20) PRIMARY KEY,
    welcome_channel_id VARCHAR(20) NOT NULL,
    vc_category_id VARCHAR(20) NOT NULL,
    button_message_id VARCHAR(20),
    in_progress_role_id VARCHAR(20),
    completed_role_id VARCHAR(20),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

## Cache Keys

- `welcomebot:config:{guild_id}` - Configuration cache
- `welcomebot:slaves:status:{slave_id}` - Slave status (available/busy/offline)
- `welcomebot:session:{guild_id}:{user_id}` - Active session data

## Task Queue

### Task Types

**`onboarding_start`** (Master → Slave)
```json
{
  "guild_id": "123",
  "user_id": "456",
  "category_id": "789",
  "slave_id": "slave-1",
  "in_progress_role": "role_id",
  "completed_role": "role_id"
}
```

**`onboarding_complete`** (Slave → Master)
```json
{
  "guild_id": "123",
  "user_id": "456",
  "slave_id": "slave-1"
}
```

**`slave_heartbeat`** (Slave → Master)
```json
{
  "slave_id": "slave-1",
  "status": "available"
}
```

## Concurrency

- Each slave handles ONE user at a time
- Maximum 3 concurrent onboarding sessions (1 per slave)
- Sessions timeout after 10 minutes total
- Inactivity timeout after 5 minutes

## Role Management

### Progress Tracking via Roles

**In-Progress Role** (optional):
- Added when onboarding starts
- Removed on completion or timeout
- Prevents duplicate sessions

**Completed Role** (optional):
- Added when onboarding completes
- Never removed automatically
- Can be used for access control

## Voice Audio

### Audio Files Location
`./audio/` directory in project root

### Required Files (Phase 1)
- `welcome.mp3` - Initial welcome message
- Additional files for each step

### Future: Dynamic Audio
- Text-to-speech generation
- Per-guild custom audio
- Multi-language support

## Error Handling

### Session Timeout
- Total session limit: 10 minutes
- Inactivity limit: 5 minutes
- Auto-cleanup on timeout

### Slave Crash Recovery
- Master monitors heartbeats (1-minute intervals)
- Mark slave offline if no heartbeat for 2 minutes
- Orphaned sessions timeout automatically

### VC Cleanup
- Immediate deletion on completion
- Deletion on timeout
- Periodic orphan scan by master (future)

## Permissions Required

### Master Bot
- Read Messages
- Send Messages
- Manage Messages (for button posting)
- Manage Roles (optional, for role assignment)

### Slave Bots
- Read Messages
- Send Messages
- Create Voice Channels
- Connect to Voice
- Speak in Voice
- Manage Channels (for deletion)
- Manage Roles (for role assignment)

## Environment Variables

### Master Bot
Standard configuration + none additional

### Slave Bots
- `SLAVE_ID` - Unique ID (slave-1, slave-2, slave-3)
- `DISCORD_BOT_TOKEN` - Bot token for this slave
- Standard database/cache/queue configuration

## Testing Scenarios

1. **Single User Onboarding**: Basic flow completion
2. **Concurrent Users**: 3 users simultaneously
3. **All Slaves Busy**: 4th user sees "busy" message
4. **User Timeout**: Inactive user, session cleanup
5. **Slave Crash**: Slave goes offline mid-session
6. **VC Permissions**: Verify only user and bot can see/join

## Future Enhancements

### Phase 2: Role Configuration
- Admin selects in-progress and completed roles
- Automatic role assignment

### Phase 3: Custom Flow Builder
- Admin defines custom steps
- Branching logic based on user choices
- Custom audio/text per step

### Phase 4: Analytics
- Track completion rates
- Average session duration
- Drop-off points

## Translation Keys

See `internal/core/i18n/translations/en.json` and `ja.json` for full list.

Key sections:
- `welcome.step1_title` - Configuration wizard
- `welcome.button_title` - Welcome button
- `welcome.starting_title` - User feedback
- `welcome.config_not_found` - Error messages

