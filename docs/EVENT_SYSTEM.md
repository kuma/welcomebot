# Event System Architecture

**Version**: 1.0  
**Status**: Design Document  
**Last Updated**: 2025-10-28

---

## Overview

The event system handles Discord events (messages, voice state, member events) and routes them efficiently to interested features.

**Design Philosophy:**
- **Hybrid Approach**: Indexed routing for high-frequency events, filtering for low-frequency
- **Guild-Aware**: All events are guild-scoped
- **Validation-Based**: Like init system - check actual config, not flags
- **Type-Safe**: No `interface{}`, use actual Discord types
- **Error-Isolated**: One feature's error doesn't break others

---

## Event Categories

### High-Frequency Events (Indexed Routing)

Events that occur frequently and target specific channels:

| Event | Frequency | Routing Strategy |
|-------|-----------|------------------|
| Message Create | 100+ per second | O(1) index by guild+channel |
| Message Delete | 10+ per second | O(1) index by guild+channel |
| Message Update | 10+ per second | O(1) index by guild+channel |
| Voice State Update | 10-50 per second | O(1) index by guild+channel |

**Why Indexed:**
- ✅ Fast O(1) lookup
- ✅ Only relevant features called
- ✅ No repeated config queries
- ✅ Scales with message volume

### Low-Frequency Events (Filtered Routing)

Events that occur rarely or don't have specific targets:

| Event | Frequency | Routing Strategy |
|-------|-----------|------------------|
| Member Join | Few per hour | Filter with ShouldHandle() |
| Member Leave | Few per hour | Filter with ShouldHandle() |
| Reaction Add | Sporadic | Filter with ShouldHandle() |
| Role Update | Rare | Filter with ShouldHandle() |

**Why Filtered:**
- ✅ Simple implementation
- ✅ Flexible (features decide)
- ✅ No index management
- ✅ Low frequency = acceptable cost

---

## Architecture

### Core Components

```
┌──────────────────────────────────────────────────┐
│  Discord Event                                   │
│  (MessageCreate, VoiceStateUpdate, etc.)         │
└────────────────┬─────────────────────────────────┘
                 │
                 ▼
┌──────────────────────────────────────────────────┐
│  Bot Handler                                     │
│  - Extracts guild_id, channel_id, user_id        │
│  - Wraps in context                              │
└────────────────┬─────────────────────────────────┘
                 │
                 ▼
┌──────────────────────────────────────────────────┐
│  Event Router (Hybrid)                           │
│  ┌────────────────┐  ┌────────────────────────┐ │
│  │ Indexed Route  │  │ Filtered Route         │ │
│  │ (fast)         │  │ (flexible)             │ │
│  └────────────────┘  └────────────────────────┘ │
└────────────────┬─────────────────────────────────┘
                 │
         ┌───────┼───────┐
         ▼       ▼       ▼
    Feature1  Feature2  Feature3
    (only interested features)
```

### File Structure

```
internal/bot/
├── feature.go           # Existing: Base Feature interface
├── registry.go          # Existing: Feature registry
├── event_interfaces.go  # NEW: Event handler interfaces
├── event_router.go      # NEW: Hybrid event router
└── bot.go              # Updated: Connect events to router
```

---

## Event Interfaces

### Message Events (Indexed)

```go
// MessageEventFeature handles message events in specific channels
type MessageEventFeature interface {
    Feature
    
    // RegisterMessageHandlers registers for message events in specific channels
    // Called when feature is loaded or config changes
    RegisterMessageHandlers(router MessageRouter, guildID string) error
    
    // UnregisterMessageHandlers cleans up registrations
    // Called when config is deleted or feature is unloaded
    UnregisterMessageHandlers(router MessageRouter, guildID string) error
}

// MessageRouter provides message event registration
type MessageRouter interface {
    // OnMessageCreate registers handler for messages in a specific channel
    OnMessageCreate(guildID, channelID string, handler MessageCreateHandler)
    
    // OnMessageDelete registers handler for message deletions
    OnMessageDelete(guildID, channelID string, handler MessageDeleteHandler)
    
    // OffMessage unregisters all handlers for a channel
    OffMessage(guildID, channelID string)
}

// Handler types
type MessageCreateHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) error
type MessageDeleteHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.MessageDelete) error
```

### Voice Events (Indexed)

```go
// VoiceEventFeature handles voice state changes in specific channels
type VoiceEventFeature interface {
    Feature
    
    RegisterVoiceHandlers(router VoiceRouter, guildID string) error
    UnregisterVoiceHandlers(router VoiceRouter, guildID string) error
}

type VoiceRouter interface {
    OnVoiceJoin(guildID, channelID string, handler VoiceJoinHandler)
    OnVoiceLeave(guildID, channelID string, handler VoiceLeaveHandler)
    OffVoice(guildID, channelID string)
}

type VoiceJoinHandler func(ctx context.Context, s *discordgo.Session, userID string, v *discordgo.VoiceStateUpdate) error
type VoiceLeaveHandler func(ctx context.Context, s *discordgo.Session, userID string, v *discordgo.VoiceStateUpdate) error
```

### Member Events (Filtered)

```go
// MemberEventFeature handles member join/leave (guild-wide)
type MemberEventFeature interface {
    Feature
    
    // ShouldHandleMemberEvent checks if feature cares about member events in this guild
    ShouldHandleMemberEvent(ctx context.Context, guildID string) bool
    
    HandleMemberJoin(ctx context.Context, s *discordgo.Session, m *discordgo.GuildMemberAdd) error
    HandleMemberLeave(ctx context.Context, s *discordgo.Session, m *discordgo.GuildMemberRemove) error
}
```

### Reaction Events (Filtered)

```go
// ReactionEventFeature handles reaction events
type ReactionEventFeature interface {
    Feature
    
    ShouldHandleReaction(ctx context.Context, guildID, channelID, messageID string) bool
    HandleReactionAdd(ctx context.Context, s *discordgo.Session, r *discordgo.MessageReactionAdd) error
    HandleReactionRemove(ctx context.Context, s *discordgo.Session, r *discordgo.MessageReactionRemove) error
}
```

---

## Event Router Implementation

### Data Structure

```go
type EventRouter struct {
    logger logger.Logger
    
    // Indexed handlers (high-frequency)
    // messageHandlers[eventType][guildID][channelID] = []handler
    messageHandlers map[EventType]map[string]map[string][]MessageCreateHandler
    voiceHandlers   map[string]map[string][]VoiceJoinHandler  // [guildID][channelID]
    
    // Filtered features (low-frequency)
    memberFeatures   []MemberEventFeature
    reactionFeatures []ReactionEventFeature
    
    mu sync.RWMutex  // Protect concurrent access
}
```

### Registration Methods

```go
// OnMessageCreate registers a handler for messages in specific channel
func (r *EventRouter) OnMessageCreate(guildID, channelID string, handler MessageCreateHandler) {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if r.messageHandlers[EventMessageCreate] == nil {
        r.messageHandlers[EventMessageCreate] = make(map[string]map[string][]MessageCreateHandler)
    }
    if r.messageHandlers[EventMessageCreate][guildID] == nil {
        r.messageHandlers[EventMessageCreate][guildID] = make(map[string][]MessageCreateHandler)
    }
    
    r.messageHandlers[EventMessageCreate][guildID][channelID] = append(
        r.messageHandlers[EventMessageCreate][guildID][channelID],
        handler,
    )
    
    r.logger.Debug("message handler registered",
        "guild_id", guildID,
        "channel_id", channelID,
    )
}
```

### Routing Methods

```go
// RouteMessageCreate routes message creation events (indexed)
func (r *EventRouter) RouteMessageCreate(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) {
    guildID := m.GuildID
    channelID := m.ChannelID
    
    r.mu.RLock()
    handlers := r.messageHandlers[EventMessageCreate][guildID][channelID]
    r.mu.RUnlock()
    
    // Call indexed handlers
    for _, handler := range handlers {
        if err := handler(ctx, s, m); err != nil {
            r.logger.Error("message handler error",
                "guild_id", guildID,
                "channel_id", channelID,
                "error", err,
            )
            // Continue to next handler
        }
    }
}

// RouteMemberJoin routes member join events (filtered)
func (r *EventRouter) RouteMemberJoin(ctx context.Context, s *discordgo.Session, m *discordgo.GuildMemberAdd) {
    guildID := m.GuildID
    
    // Call filtered features
    for _, feature := range r.memberFeatures {
        if !feature.ShouldHandleMemberEvent(ctx, guildID) {
            continue  // Feature not interested in this guild
        }
        
        if err := feature.HandleMemberJoin(ctx, s, m); err != nil {
            r.logger.Error("member join handler error",
                "guild_id", guildID,
                "feature", feature.Name(),
                "error", err,
            )
            // Continue to next feature
        }
    }
}
```

---

## Feature Implementation Pattern

### Self-Intro Feature Example

```go
package selfintro

// Implements MessageEventFeature (indexed)
type Feature struct {
    db     database.Client
    cache  cache.Client
    i18n   i18n.I18n
    logger logger.Logger
    router bot.MessageRouter  // Inject event router
}

// RegisterMessageHandlers is called when config is saved
func (f *Feature) RegisterMessageHandlers(router bot.MessageRouter, guildID string) error {
    config, err := f.getConfig(ctx.Background(), guildID)
    if err != nil {
        return nil  // Not configured, skip registration
    }
    
    // Register male channel
    router.OnMessageCreate(guildID, config.MaleChannelID, f.handleMaleIntroMessage)
    router.OnMessageDelete(guildID, config.MaleChannelID, f.handleMaleIntroDelete)
    
    // Register female channel
    router.OnMessageCreate(guildID, config.FemaleChannelID, f.handleFemaleIntroMessage)
    router.OnMessageDelete(guildID, config.FemaleChannelID, f.handleFemaleIntroDelete)
    
    return nil
}

// UnregisterMessageHandlers is called when config is deleted
func (f *Feature) UnregisterMessageHandlers(router bot.MessageRouter, guildID string) error {
    config, err := f.getConfig(ctx.Background(), guildID)
    if err != nil {
        return nil
    }
    
    router.OffMessage(guildID, config.MaleChannelID)
    router.OffMessage(guildID, config.FemaleChannelID)
    
    return nil
}

// Handler for male intro messages
func (f *Feature) handleMaleIntroMessage(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) error {
    // Save intro post
    url := fmt.Sprintf("https://discord.com/channels/%s/%s/%s",
        m.GuildID, m.ChannelID, m.ID)
    
    return f.saveIntroPost(ctx, m.GuildID, m.Author.ID, url, "male")
}

// Handler for intro message deletion
func (f *Feature) handleMaleIntroDelete(ctx context.Context, s *discordgo.Session, m *discordgo.MessageDelete) error {
    return f.deleteIntroPost(ctx, m.GuildID, m.ID)
}

// When saving config, register handlers
func (f *Feature) saveConfig(ctx, guildID, maleChannelID, femaleChannelID) error {
    // 1. Save to database
    query := `INSERT INTO ... ON CONFLICT UPDATE ...`
    _, err := f.db.Exec(ctx, query, guildID, maleChannelID, femaleChannelID)
    if err != nil {
        return err
    }
    
    // 2. Update cache
    config := &SelfIntroConfig{...}
    f.cache.SetJSON(ctx, cacheKey, config, 0)
    
    // 3. Re-register event handlers
    f.UnregisterMessageHandlers(f.router, guildID)  // Clean old
    f.RegisterMessageHandlers(f.router, guildID)    // Register new
    
    f.logger.Info("selfintro handlers registered",
        "guild_id", guildID,
        "male_channel", maleChannelID,
        "female_channel", femaleChannelID,
    )
    
    return nil
}
```

### Voice Join Feature Example

```go
// Implements VoiceEventFeature (indexed or filtered)
type Feature struct {
    db     database.Client
    i18n   i18n.I18n
    logger logger.Logger
}

// Option A: Indexed (if specific voice channels known)
func (f *Feature) RegisterVoiceHandlers(router bot.VoiceRouter, guildID string) error {
    // If you know specific VCs to monitor
    channelIDs := f.getMonitoredChannels(guildID)
    
    for _, channelID := range channelIDs {
        router.OnVoiceJoin(guildID, channelID, f.handleVoiceJoin)
    }
    
    return nil
}

// Option B: Filtered (if all VCs need monitoring)
func (f *Feature) ShouldHandleVoice(ctx, guildID, channelID) bool {
    // Check if this guild has voice intro posting enabled
    return f.isVoiceIntroEnabled(ctx, guildID)
}

func (f *Feature) HandleVoiceJoin(ctx, s, userID, v) error {
    // Get user's intro URL
    introURL, err := f.getIntroURL(ctx, v.GuildID, userID)
    if err != nil {
        return nil  // User has no intro, skip
    }
    
    // Post to voice channel's text chat
    textChannelID := f.getVoiceTextChannel(v.ChannelID)
    
    msg := f.i18n.TWithArgs(ctx, v.GuildID, "voice.intro_posted",
        map[string]string{
            "user": fmt.Sprintf("<@%s>", userID),
            "intro": introURL,
        })
    
    return f.discord.SendMessage(ctx, textChannelID, discord.Message{
        Content: msg,
    })
}
```

---

## Event Router Interface

### MessageRouter

```go
type MessageRouter interface {
    // OnMessageCreate registers handler for message creation in specific channel
    OnMessageCreate(guildID, channelID string, handler MessageCreateHandler)
    
    // OnMessageDelete registers handler for message deletion
    OnMessageDelete(guildID, channelID string, handler MessageDeleteHandler)
    
    // OnMessageUpdate registers handler for message updates
    OnMessageUpdate(guildID, channelID string, handler MessageUpdateHandler)
    
    // OffMessage unregisters all message handlers for a channel
    OffMessage(guildID, channelID string)
    
    // OffMessageGuild unregisters all handlers for a guild
    OffMessageGuild(guildID string)
}
```

### VoiceRouter

```go
type VoiceRouter interface {
    // OnVoiceJoin registers handler for users joining voice channel
    OnVoiceJoin(guildID, channelID string, handler VoiceJoinHandler)
    
    // OnVoiceLeave registers handler for users leaving voice channel
    OnVoiceLeave(guildID, channelID string, handler VoiceLeaveHandler)
    
    // OffVoice unregisters all voice handlers for a channel
    OffVoice(guildID, channelID string)
}
```

### Handler Types

```go
// Message handlers
type MessageCreateHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) error
type MessageDeleteHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.MessageDelete) error
type MessageUpdateHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.MessageUpdate) error

// Voice handlers
type VoiceJoinHandler func(ctx context.Context, s *discordgo.Session, userID string, v *discordgo.VoiceStateUpdate) error
type VoiceLeaveHandler func(ctx context.Context, s *discordgo.Session, userID string, v *discordgo.VoiceStateUpdate) error

// Member handlers (filtered features)
type MemberJoinHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.GuildMemberAdd) error
type MemberLeaveHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.GuildMemberRemove) error
```

---

## Registration Lifecycle

### On Bot Startup

```go
// 1. Bot starts
// 2. Features are registered
// 3. For each guild in cache/database:
for _, guildID := range knownGuilds {
    // Ask each feature to register handlers for this guild
    for _, feature := range features {
        if msgFeature, ok := feature.(MessageEventFeature); ok {
            msgFeature.RegisterMessageHandlers(router, guildID)
        }
        if voiceFeature, ok := feature.(VoiceEventFeature); ok {
            voiceFeature.RegisterVoiceHandlers(router, guildID)
        }
    }
}
```

### On Config Change

```go
// When admin configures selfintro channels:
func (f *SelfIntroFeature) saveConfig(ctx, guildID, maleChannelID, femaleChannelID) error {
    // 1. Save configuration
    db.Exec(...)
    cache.Set(...)
    
    // 2. Unregister old handlers
    f.UnregisterMessageHandlers(f.router, guildID)
    
    // 3. Register new handlers with updated config
    f.RegisterMessageHandlers(f.router, guildID)
    
    return nil
}
```

### On Config Delete

```go
func (f *SelfIntroFeature) deleteConfig(ctx, guildID) error {
    // 1. Delete from database
    db.Exec("DELETE FROM guild_selfintro_channels WHERE guild_id = $1", guildID)
    
    // 2. Delete from cache
    cache.Delete(ctx, key)
    
    // 3. Unregister event handlers
    f.UnregisterMessageHandlers(f.router, guildID)
    
    return nil
}
```

---

## Performance Characteristics

### Indexed Events (Messages, Voice)

```
Event arrives → O(1) map lookup → Call 1-3 handlers

Example:
  Message in #male-intro (guild 123)
  → index[MessageCreate]["123"]["#male-intro"] → [selfintroHandler]
  → Call only selfintro feature
  → Total: ~100 microseconds
```

### Filtered Events (Members, Reactions)

```
Event arrives → O(N) check all features → Call matching handlers

Example:
  Member joins guild 123
  → Check 10 features: shouldHandle(123)? → 3 say yes
  → Call 3 handlers
  → Total: ~500 microseconds (acceptable for rare events)
```

### Comparison

| Approach | Messages/sec | Latency | Memory |
|----------|--------------|---------|--------|
| **No filtering** | 100 | 5ms | Low |
| **Filter only** | 100 | 2ms | Low |
| **Indexed** | 100 | 0.1ms | Medium |
| **Hybrid** | 100 | 0.1-0.5ms | Medium |

**Hybrid gives best balance!** ✅

---

## Error Handling

### Isolated Errors

```go
// One handler error doesn't stop others
for _, handler := range handlers {
    if err := handler(ctx, s, event); err != nil {
        logger.Error("handler error",
            "feature", featureName,
            "event", eventType,
            "error", err,
        )
        // Continue to next handler (don't return!)
    }
}
```

### Handler Timeout (Optional)

```go
// Prevent slow handlers from blocking
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

if err := handler(ctx, s, event); err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        logger.Error("handler timeout", "feature", name)
    }
}
```

---

## Use Case: Self-Intro Feature

### Data Flow

```
1. Admin configures selfintro channels
   → saveConfig() is called
   → Registers: OnMessageCreate(guildID, maleChannelID, handleMaleMessage)
   → Registers: OnMessageCreate(guildID, femaleChannelID, handleFemaleMessage)
   → Registers: OnMessageDelete(guildID, maleChannelID, handleDelete)
   → Registers: OnMessageDelete(guildID, femaleChannelID, handleDelete)

2. User posts message in #male-intro
   → Discord sends MessageCreate event
   → Router: index[MessageCreate][guildID][maleChannelID]
   → Calls: handleMaleMessage()
   → Saves: intro post URL to database

3. User deletes message
   → Discord sends MessageDelete event
   → Router: index[MessageDelete][guildID][maleChannelID]
   → Calls: handleDelete()
   → Deletes: saved intro post

4. User joins voice channel
   → Discord sends VoiceStateUpdate
   → Router: calls all VoiceEventFeatures.ShouldHandle()
   → SelfIntro: YES (has intro saved)
   → Calls: HandleVoiceJoin()
   → Posts: intro URL to voice text channel
```

---

## Benefits

### Efficiency
- ✅ O(1) for high-frequency events
- ✅ Only relevant features called
- ✅ Cache hits on config lookups
- ✅ Scales with event volume

### Maintainability
- ✅ Clear interfaces
- ✅ Features opt-in to events
- ✅ Easy to add new event types
- ✅ No global state

### Flexibility
- ✅ Dynamic registration
- ✅ Config changes take effect immediately
- ✅ Guild-specific handlers
- ✅ Mix indexed + filtered as needed

### Reliability
- ✅ Error isolation
- ✅ One feature can't break others
- ✅ Graceful degradation
- ✅ Observable (logging)

---

## Implementation Checklist

### Phase 1: Event Infrastructure
- [ ] Create `internal/bot/event_interfaces.go`
- [ ] Create `internal/bot/event_router.go`
- [ ] Update `internal/bot/bot.go` to use router
- [ ] Add tests for event router

### Phase 2: Feature Integration
- [ ] Add router to Dependencies
- [ ] Features implement event interfaces (opt-in)
- [ ] Features register/unregister handlers

### Phase 3: Self-Intro Events
- [ ] Implement message handlers (save/delete intro posts)
- [ ] Implement voice join handler (post intro URL)
- [ ] Add database schema for intro posts
- [ ] Test end-to-end

---

## Future Extensions

### Easy to Add:

**New Event Types:**
```go
const EventChannelCreate EventType = "channel.create"
const EventRoleUpdate EventType = "role.update"
```

**New Routing Strategies:**
```go
// By message content
router.OnMessagePattern(guildID, regex, handler)

// By user role
router.OnMessageFromRole(guildID, roleID, handler)

// By time
router.OnMessageBetween(guildID, startHour, endHour, handler)
```

---

## ⚠️ CRITICAL: Always Determine Event Frequency First!

**BEFORE implementing any event-based feature, you MUST determine:**

### Question to Ask:

**"Is this event high-frequency or low-frequency?"**

| Event Type | Frequency | Examples | Routing Strategy |
|------------|-----------|----------|------------------|
| **High-Frequency** | 10+ per second | Messages in specific channels, Voice joins/leaves in specific VCs | **Indexed** (O(1)) |
| **Low-Frequency** | < 1 per minute | Member join/leave, Role updates, Config changes | **Filtered** (O(N)) |

### Decision Flow

```
New event-based feature?
  ↓
Ask: "How often does this event occur?"
  ↓
  ├─ High (10+ per second)
  │   → Use Indexed Routing
  │   → Implement: RegisterXHandlers(router, guildID)
  │   → Register specific channels/resources
  │
  └─ Low (< 1 per minute)
      → Use Filtered Routing
      → Implement: ShouldHandle(ctx, guildID) + HandleEvent()
      → Check config dynamically
```

### Examples

**High-Frequency Features:**
```
✅ Self-intro posts (messages in 2 specific channels)
   → Indexed: OnMessageCreate(guildID, channelID, handler)
   
✅ Auto-mod (messages in all channels)
   → Indexed: OnMessageCreate for each channel
   
✅ Voice intro posting (voice joins)
   → Indexed: OnVoiceJoin(guildID, voiceChannelID, handler)
```

**Low-Frequency Features:**
```
✅ Welcome messages (member joins - few per hour)
   → Filtered: ShouldHandleMemberEvent() + HandleMemberJoin()
   
✅ Role auto-assign (member joins)
   → Filtered: ShouldHandleMemberEvent() + HandleMemberJoin()
   
✅ Audit logging (various rare events)
   → Filtered: Multiple ShouldHandle methods
```

### ⚠️ Wrong Choice = Performance Issues

**If you index a low-frequency event:**
- ❌ Wasted memory (index overhead)
- ❌ Complex lifecycle management
- ❌ Over-engineering

**If you filter a high-frequency event:**
- ❌ Repeated cache/DB queries
- ❌ High CPU usage
- ❌ Slow response time

**Always ask about frequency before implementing!**

---

## Summary

**Hybrid Event System:**
- **Indexed** for high-frequency (messages, voice) → O(1) fast
- **Filtered** for low-frequency (members, reactions) → Simple
- **Guild-aware** throughout
- **Type-safe** (no `interface{}`)
- **Error-isolated** (one failure doesn't cascade)
- **Dynamic** (config changes → handler updates)

**ALWAYS determine event frequency BEFORE implementing an event-based feature!**

**This architecture will scale to hundreds of guilds with thousands of events per second.** 🚀

