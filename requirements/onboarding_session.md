# Feature: Welcome Onboarding Session Flow

## Overview

After a user clicks "Start Onboarding", a slave bot creates a private voice channel and guides the user through an interactive voice-based onboarding experience. The user first selects their preferred guide, then follows a structured tutorial with audio narration.

## Onboarding Flow

### Phase 1: VC Creation & Initial Setup

**When**: User clicks "Start Onboarding" button

**Actions**:
1. Master assigns task to available slave
2. Slave creates private voice channel with specific settings
3. Slave joins the voice channel
4. Slave mentions the user in VC text chat

### Phase 2: Guide Selection

**Display**: Text message + Interactive UI

**Message**: "Select your guide" / "説明会のガイドを選んでください"

**UI Components**:
- **Button**: Labeled with guide name (e.g., "kk")
  - Action: Plays `0-voice-select.dca` from that guide's folder
  - Purpose: Preview the guide's voice
- **Dropdown Menu**: List of available guides
  - Shows all available guides (currently: "kk")
  - User selects their preferred guide
  - Action: Stops preview audio, continues to tutorial

### Phase 3: Tutorial Steps

**After guide selection**, play audio files sequentially:

1. `1-intro.dca` - Introduction
2. `2-profile.dca` - Profile setup
3. `3-role.dca` - Role explanation
4. `4-point.dca` - Point system
5. `5-club.dca` - Club information
6. `6-membership.dca` - Membership details
7. `7-end.dca` - Completion

**Between steps**: Show interactive buttons for user progression

### Phase 4: Completion

1. Add completion role (if configured)
2. Remove in-progress role
3. Delete voice channel immediately

## Voice Channel Configuration

### Channel Settings

```go
ChannelCreate(&discordgo.GuildChannelCreateData{
    Name:       "onboarding-{username}",
    Type:       discordgo.ChannelTypeGuildVoice,
    ParentID:   categoryID,
    Bitrate:    128000,  // 128kbps
    UserLimit:  2,       // Max 2 users (user + bot)
})
```

### Permission Overwrites

**User** (who clicked button):
- ✅ `ViewChannel` = Allow
- ✅ `Connect` = Allow
- ✅ `Speak` = Allow

**Bot** (slave):
- ✅ `ViewChannel` = Allow
- ✅ `Connect` = Allow
- ✅ `Speak` = Allow

**@everyone**:
- ❌ `ViewChannel` = Deny

**Server Owner/Admins**:
- ✅ Can see and join (inherent permission)

### Channel Lifetime

- **Created**: When onboarding task starts
- **Deleted**: Immediately after completion OR timeout
- **Timeout**: 10 minutes total session OR 5 minutes inactivity

## Audio Files Structure

### Directory Layout

```
audio/
├── README.md
└── {guide_name}/
    ├── 0-voice-select.dca  # Guide preview (plays on button click)
    ├── 1-intro.dca         # Introduction
    ├── 2-profile.dca       # Profile setup
    ├── 3-role.dca          # Role explanation
    ├── 4-point.dca         # Point system
    ├── 5-club.dca          # Club information
    ├── 6-membership.dca    # Membership details
    └── 7-end.dca           # Completion
```

### Current Guides

- `kk/` - First guide (Kei-chan)

### Future Guides

**TODO**: Implement mechanism to:
- Add new guides dynamically
- Remove guides
- List available guides from directory scan

## Guide Selection UI

### Message Embed

```go
Embed{
    Title: i18n.T(ctx, guildID, "onboarding.select_guide_title"),
    Description: i18n.T(ctx, guildID, "onboarding.select_guide_description"),
    Color: ColorInfo,
}
```

### Button Component

```go
Button{
    Label: "kk",  // Guide name
    Style: PrimaryButton,
    CustomID: "onboarding:preview:kk",
    Emoji: "🎧",
}
```

**Behavior**:
1. User clicks button
2. Bot plays `audio/kk/0-voice-select.dca`
3. User can click other guide buttons to preview different voices
4. Audio stops when user selects from dropdown

### Dropdown Component

```go
SelectMenu{
    CustomID: "onboarding:select_guide",
    Placeholder: i18n.T(ctx, guildID, "onboarding.choose_guide"),
    Options: []SelectMenuOption{
        {
            Label: "kk",
            Value: "kk",
            Description: i18n.T(ctx, guildID, "onboarding.guide.kk.description"),
            Emoji: "👤",
        },
    },
}
```

**Behavior**:
1. User selects guide from dropdown
2. Bot stops any playing audio
3. Bot proceeds to tutorial step 1
4. Bot plays `audio/{guide}/1-intro.dca`

## Technical Requirements

### Guild-Aware ⚠️

All operations MUST filter by `guild_id`:

```go
// Cache key
sessionKey := fmt.Sprintf("welcomebot:session:%s:%s", guildID, userID)

// Voice channel permissions
perms := []*discordgo.PermissionOverwrite{
    {
        ID:   userID,
        Type: discordgo.PermissionOverwriteTypeMember,
        Allow: discordgo.PermissionViewChannel | 
               discordgo.PermissionConnect | 
               discordgo.PermissionSpeak,
    },
    // ...
}
```

### i18n - All Text Translated

```json
{
  "onboarding": {
    "select_guide_title": "Select Your Guide",
    "select_guide_description": "Choose who will guide you through onboarding",
    "choose_guide": "Choose a guide...",
    "guide": {
      "kk": {
        "description": "Friendly and energetic guide"
      }
    },
    "preview_playing": "🎧 Preview playing...",
    "continuing": "Starting onboarding with {guide}...",
    "vc_created": "Voice channel created: {channel}"
  }
}
```

Japanese:
```json
{
  "onboarding": {
    "select_guide_title": "説明会のガイドを選んでください",
    "select_guide_description": "オンボーディングを案内してくれる人を選択してください",
    "choose_guide": "ガイドを選択...",
    "guide": {
      "kk": {
        "description": "フレンドリーで元気なガイド"
      }
    },
    "preview_playing": "🎧 プレビュー再生中...",
    "continuing": "{guide}でオンボーディングを開始します...",
    "vc_created": "ボイスチャンネルが作成されました: {channel}"
  }
}
```

### Stateless Interaction Handling

**CustomID Format**:
```
onboarding:preview:{guide_name}          # Preview button
onboarding:select_guide                   # Dropdown selection
onboarding:step:{step_number}:{guide}    # Step progression
onboarding:complete:{guide}              # Completion
```

**State Storage**: Redis cache

```go
type OnboardingSession struct {
    GuildID      string    `json:"guild_id"`
    UserID       string    `json:"user_id"`
    SlaveID      string    `json:"slave_id"`
    Guide        string    `json:"guide"`
    CurrentStep  int       `json:"current_step"`
    VCChannelID  string    `json:"vc_channel_id"`
    LastActivity time.Time `json:"last_activity"`
}
```

## Audio Playback

### DCA File Format

- **Format**: Opus-encoded DCA (Discord Compatible Audio)
- **Sample Rate**: 48kHz
- **Channels**: Stereo (2)
- **Bitrate**: 64-128 kb/s

### Playback Implementation

```go
func (s *OnboardingSession) playAudio(guide, filename string) error {
    path := fmt.Sprintf("audio/%s/%s", guide, filename)
    
    // Open DCA file
    file, err := os.Open(path)
    if err != nil {
        return fmt.Errorf("open audio file: %w", err)
    }
    defer file.Close()
    
    // Create DCA decoder
    decoder := dca.NewDecoder(file)
    
    // Stream to voice connection
    for {
        frame, err := decoder.OpusFrame()
        if err == io.EOF {
            break
        }
        if err != nil {
            return fmt.Errorf("decode frame: %w", err)
        }
        
        s.voiceConn.OpusSend <- frame
    }
    
    return nil
}
```

### Stop Audio Playback

When user selects guide, stop any playing preview:

```go
func (s *OnboardingSession) stopAudio() {
    // Clear the voice send channel
    select {
    case <-s.voiceConn.OpusSend:
        // Drain channel
    default:
        // Already empty
    }
}
```

## Examples

### Example 1: Happy Path

```
User: [Clicks "Start Onboarding" button]
Master: Assigns task to slave-1
Slave-1: Creates VC "onboarding-john"
Slave-1: Joins VC
Slave-1: @john Welcome to the onboarding! 👋

[Embed shows guide selection]
Title: "Select Your Guide"
Description: "Choose who will guide you through onboarding"
[Button: kk 🎧] [Dropdown: Choose a guide...]

User: [Clicks "kk" button]
Slave-1: Plays audio/kk/0-voice-select.dca
        [Audio: "こんにちは！私はKKです。一緒にサーバーの説明をしましょう！"]

User: [Selects "kk" from dropdown]
Slave-1: Stops audio
Slave-1: "Starting onboarding with kk..."
Slave-1: Plays audio/kk/1-intro.dca
        [Audio: "それでは、最初に..."]

[Tutorial continues through steps 2-7]

Slave-1: Plays audio/kk/7-end.dca
Slave-1: Adds completion role
Slave-1: Removes in-progress role
Slave-1: Deletes VC "onboarding-john"

User: [Onboarding complete! ✅]
```

### Example 2: Multiple Guides (Future)

```
User: [Clicks "Start Onboarding" button]
Slave-2: Creates VC, joins, mentions user

[Embed shows guide selection]
[Button: kk 🎧] [Button: yuki 🎧] [Button: sato 🎧]
[Dropdown: kk, yuki, sato]

User: [Clicks "yuki" button]
Slave-2: Plays audio/yuki/0-voice-select.dca
        [Audio in different voice]

User: [Clicks "sato" button]
Slave-2: Stops yuki preview
Slave-2: Plays audio/sato/0-voice-select.dca

User: [Selects "sato" from dropdown]
Slave-2: Continues with Sato as guide
```

### Example 3: User Inactivity

```
User: [Clicks "Start Onboarding" button]
Slave-1: Creates VC, shows guide selection
User: [Doesn't interact for 5 minutes]
Slave-1: Inactivity timeout detected
Slave-1: Sends: "⏰ Session timed out due to inactivity"
Slave-1: Removes in-progress role
Slave-1: Deletes VC
```

## Error Handling

### No Audio File Found

```go
if _, err := os.Stat(audioPath); os.IsNotExist(err) {
    s.logger.Error("audio file not found", 
        "guide", guide, 
        "file", filename,
    )
    return s.sendMessage(i18n.T(ctx, guildID, "onboarding.audio_missing"))
}
```

### VC Creation Failed

```go
if err := createVC(); err != nil {
    s.logger.Error("failed to create VC", "error", err)
    // Notify user
    s.session.ChannelMessageSend(welcomeChannelID, 
        i18n.TWithArgs(ctx, guildID, "onboarding.vc_failed", map[string]string{
            "user": fmt.Sprintf("<@%s>", userID),
        }),
    )
    // Mark slave as available again
    s.queue.Enqueue(ctx, queue.Task{
        Type: "slave_status_update",
        Payload: map[string]interface{}{
            "slave_id": slaveID,
            "status": "available",
        },
    })
}
```

### Voice Connection Lost

```go
func (s *OnboardingSession) monitorVoiceConnection() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            if !s.voiceConn.Ready {
                s.logger.Warn("voice connection lost")
                s.cleanup()
                return
            }
        case <-s.ctx.Done():
            return
        }
    }
}
```

## Edge Cases

### User Already in Onboarding

```go
sessionKey := fmt.Sprintf("welcomebot:session:%s:%s", guildID, userID)
var existingSession OnboardingSession
if err := cache.GetJSON(ctx, sessionKey, &existingSession); err == nil {
    return respondError(s, i, guildID, "onboarding.session_active")
}
```

### User Leaves VC During Onboarding

Monitor voice state updates:

```go
func (s *OnboardingSession) handleVoiceStateUpdate(vs *discordgo.VoiceStateUpdate) {
    if vs.UserID == s.userID && vs.ChannelID != s.vcChannelID {
        s.logger.Info("user left onboarding VC")
        s.Complete() // Cleanup
    }
}
```

### All Guides Deleted

```go
guides, err := listGuides()
if err != nil || len(guides) == 0 {
    s.logger.Error("no guides available")
    return respondError(s, i, guildID, "onboarding.no_guides")
}
```

## Future Enhancements

### Phase 2: Dynamic Guide Discovery

```go
// TODO: Scan audio/ directory for guides
func listAvailableGuides() ([]string, error) {
    entries, err := os.ReadDir("audio")
    if err != nil {
        return nil, err
    }
    
    var guides []string
    for _, entry := range entries {
        if entry.IsDir() && entry.Name() != "." && entry.Name() != ".." {
            // Check if required audio files exist
            if validateGuideFiles(entry.Name()) {
                guides = append(guides, entry.Name())
            }
        }
    }
    
    return guides, nil
}
```

### Phase 3: Admin Guide Management

Via `/welcomebot` menu:
- Add new guide (upload audio files)
- Remove guide
- Set guide descriptions
- Preview guides

### Phase 4: Interactive Tutorial Steps

Each step can have:
- Multiple choice questions
- User responses affect next steps
- Branching logic
- Custom audio based on choices

### Phase 5: Analytics

Track:
- Which guides are most popular
- Average completion time per guide
- Drop-off points in tutorial
- User satisfaction ratings

## Database Schema (Future)

```sql
-- Guide metadata (Phase 2+)
CREATE TABLE guild_onboarding_guides (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(20) NOT NULL,
    guide_name VARCHAR(50) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(guild_id, guide_name)
);

-- Session analytics (Phase 5)
CREATE TABLE onboarding_sessions (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(20) NOT NULL,
    user_id VARCHAR(20) NOT NULL,
    guide_name VARCHAR(50),
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    duration_seconds INT,
    completed BOOLEAN DEFAULT false,
    steps_completed INT DEFAULT 0,
    total_steps INT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## Testing Scenarios

1. **Basic Flow**: User selects guide, completes onboarding
2. **Preview Multiple Guides**: Click different guide buttons before selecting
3. **User Timeout**: Inactive for 5 minutes
4. **User Leaves VC**: User disconnects during onboarding
5. **Audio File Missing**: Guide folder exists but audio file missing
6. **No Guides Available**: Empty audio directory
7. **Concurrent Sessions**: 3 users onboarding simultaneously
8. **Slave Crash**: Slave goes offline mid-session
9. **VC Permission Check**: Verify other users can't see VC
10. **Admin Override**: Server owner can join private VC

## Dependencies

- `github.com/bwmarrin/discordgo` - Discord API
- `github.com/bwmarrin/dca` - DCA audio decoder
- `welcomebot/internal/core/cache` - Redis caching
- `welcomebot/internal/core/database` - PostgreSQL
- `welcomebot/internal/core/queue` - Task queue
- `welcomebot/internal/core/i18n` - Internationalization
- `welcomebot/internal/core/logger` - Structured logging

## Performance Considerations

### Audio File Caching

**Don't** cache in memory (files are small, disk read is fast):

```go
// Simple disk read is fine
file, err := os.Open(audioPath)
```

### Voice Connection Pooling

Reuse voice connections when possible:

```go
// Keep connection alive between steps
// Don't disconnect/reconnect for each audio file
```

### Concurrent Session Limit

Hard limit of 3 concurrent sessions (one per slave):

```go
if activeSessions >= maxSlaves {
    return respondError(s, i, guildID, "onboarding.all_busy")
}
```

## Security Considerations

### VC Privacy

- Only user + bot can see/join
- Server owner inherently can join (Discord limitation)
- Log admin joins for audit trail

### Audio File Validation

```go
func validateAudioFile(path string) error {
    // Check file exists
    info, err := os.Stat(path)
    if err != nil {
        return err
    }
    
    // Check file size (prevent abuse)
    if info.Size() > 10*1024*1024 { // 10MB max
        return errors.New("audio file too large")
    }
    
    // Check file extension
    if !strings.HasSuffix(path, ".dca") {
        return errors.New("invalid audio format")
    }
    
    return nil
}
```

### Rate Limiting

Prevent spam of onboarding button:

```go
rateLimitKey := fmt.Sprintf("welcomebot:ratelimit:%s:%s", guildID, userID)
if exists := cache.Exists(ctx, rateLimitKey); exists {
    return respondError(s, i, guildID, "onboarding.cooldown")
}
cache.Set(ctx, rateLimitKey, "1", 60*time.Second) // 1 minute cooldown
```

