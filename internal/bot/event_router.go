package bot

import (
	"context"
	"sync"

	"welcomebot/internal/core/logger"

	"github.com/bwmarrin/discordgo"
)

// EventRouter handles hybrid event routing.
type EventRouter struct {
	logger logger.Logger

	// Indexed handlers (high-frequency)
	messageCreateHandlers map[string]map[string][]MessageCreateHandler // [guildID][channelID][]
	messageDeleteHandlers map[string]map[string][]MessageDeleteHandler
	voiceJoinHandlers     map[string]map[string][]VoiceJoinHandler
	voiceLeaveHandlers    map[string]map[string][]VoiceLeaveHandler

	mu sync.RWMutex
}

// NewEventRouter creates a new event router.
func NewEventRouter(log logger.Logger) *EventRouter {
	return &EventRouter{
		logger:                log,
		messageCreateHandlers: make(map[string]map[string][]MessageCreateHandler),
		messageDeleteHandlers: make(map[string]map[string][]MessageDeleteHandler),
		voiceJoinHandlers:     make(map[string]map[string][]VoiceJoinHandler),
		voiceLeaveHandlers:    make(map[string]map[string][]VoiceLeaveHandler),
	}
}

// OnMessageCreate registers a message creation handler.
func (r *EventRouter) OnMessageCreate(guildID, channelID string, handler MessageCreateHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.messageCreateHandlers[guildID] == nil {
		r.messageCreateHandlers[guildID] = make(map[string][]MessageCreateHandler)
	}

	r.messageCreateHandlers[guildID][channelID] = append(
		r.messageCreateHandlers[guildID][channelID],
		handler,
	)

	r.logger.Debug("message create handler registered",
		"guild_id", guildID,
		"channel_id", channelID,
	)
}

// OnMessageDelete registers a message deletion handler.
func (r *EventRouter) OnMessageDelete(guildID, channelID string, handler MessageDeleteHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.messageDeleteHandlers[guildID] == nil {
		r.messageDeleteHandlers[guildID] = make(map[string][]MessageDeleteHandler)
	}

	r.messageDeleteHandlers[guildID][channelID] = append(
		r.messageDeleteHandlers[guildID][channelID],
		handler,
	)

	r.logger.Debug("message delete handler registered",
		"guild_id", guildID,
		"channel_id", channelID,
	)
}

// OffMessage unregisters all message handlers for a channel.
func (r *EventRouter) OffMessage(guildID, channelID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.messageCreateHandlers[guildID] != nil {
		delete(r.messageCreateHandlers[guildID], channelID)
	}
	if r.messageDeleteHandlers[guildID] != nil {
		delete(r.messageDeleteHandlers[guildID], channelID)
	}

	r.logger.Debug("message handlers unregistered",
		"guild_id", guildID,
		"channel_id", channelID,
	)
}

// OffMessageGuild unregisters all message handlers for a guild.
func (r *EventRouter) OffMessageGuild(guildID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.messageCreateHandlers, guildID)
	delete(r.messageDeleteHandlers, guildID)

	r.logger.Debug("guild message handlers unregistered", "guild_id", guildID)
}

// OnVoiceJoin registers a voice join handler.
func (r *EventRouter) OnVoiceJoin(guildID, channelID string, handler VoiceJoinHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.voiceJoinHandlers[guildID] == nil {
		r.voiceJoinHandlers[guildID] = make(map[string][]VoiceJoinHandler)
	}

	r.voiceJoinHandlers[guildID][channelID] = append(
		r.voiceJoinHandlers[guildID][channelID],
		handler,
	)
}

// OnVoiceLeave registers a voice leave handler.
func (r *EventRouter) OnVoiceLeave(guildID, channelID string, handler VoiceLeaveHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.voiceLeaveHandlers[guildID] == nil {
		r.voiceLeaveHandlers[guildID] = make(map[string][]VoiceLeaveHandler)
	}

	r.voiceLeaveHandlers[guildID][channelID] = append(
		r.voiceLeaveHandlers[guildID][channelID],
		handler,
	)
}

// OffVoice unregisters all voice handlers for a channel.
func (r *EventRouter) OffVoice(guildID, channelID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.voiceJoinHandlers[guildID] != nil {
		delete(r.voiceJoinHandlers[guildID], channelID)
	}
	if r.voiceLeaveHandlers[guildID] != nil {
		delete(r.voiceLeaveHandlers[guildID], channelID)
	}
}

// OffVoiceGuild unregisters all voice handlers for a guild.
func (r *EventRouter) OffVoiceGuild(guildID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.voiceJoinHandlers, guildID)
	delete(r.voiceLeaveHandlers, guildID)
}

// RouteMessageCreate routes message creation events (indexed).
func (r *EventRouter) RouteMessageCreate(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) {
	guildID := m.GuildID
	channelID := m.ChannelID

	r.mu.RLock()
	handlers := r.messageCreateHandlers[guildID][channelID]
	r.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, s, m); err != nil {
			r.logger.Error("message create handler error",
				"guild_id", guildID,
				"channel_id", channelID,
				"error", err,
			)
			// Continue to next handler
		}
	}
}

// RouteMessageDelete routes message deletion events (indexed).
func (r *EventRouter) RouteMessageDelete(ctx context.Context, s *discordgo.Session, m *discordgo.MessageDelete) {
	guildID := m.GuildID
	channelID := m.ChannelID

	r.mu.RLock()
	handlers := r.messageDeleteHandlers[guildID][channelID]
	r.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, s, m); err != nil {
			r.logger.Error("message delete handler error",
				"guild_id", guildID,
				"channel_id", channelID,
				"error", err,
			)
			// Continue to next handler
		}
	}
}

// RouteVoiceStateUpdate routes voice state update events (indexed + filtered).
func (r *EventRouter) RouteVoiceStateUpdate(ctx context.Context, s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	// Detect join vs leave
	isJoin := v.BeforeUpdate == nil || v.BeforeUpdate.ChannelID == ""
	isLeave := v.ChannelID == ""

	if isJoin {
		r.routeVoiceJoin(ctx, s, v.UserID, v)
	} else if isLeave {
		r.routeVoiceLeave(ctx, s, v.UserID, v)
	}
	// Else: move between channels (could add separate handler if needed)
}

// routeVoiceJoin routes voice join events.
func (r *EventRouter) routeVoiceJoin(ctx context.Context, s *discordgo.Session, userID string, v *discordgo.VoiceStateUpdate) {
	guildID := v.GuildID
	channelID := v.ChannelID

	r.mu.RLock()
	handlers := r.voiceJoinHandlers[guildID][channelID]
	r.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, s, userID, v); err != nil {
			r.logger.Error("voice join handler error",
				"guild_id", guildID,
				"channel_id", channelID,
				"user_id", userID,
				"error", err,
			)
		}
	}
}

// routeVoiceLeave routes voice leave events.
func (r *EventRouter) routeVoiceLeave(ctx context.Context, s *discordgo.Session, userID string, v *discordgo.VoiceStateUpdate) {
	guildID := v.GuildID
	channelID := v.BeforeUpdate.ChannelID

	r.mu.RLock()
	handlers := r.voiceLeaveHandlers[guildID][channelID]
	r.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, s, userID, v); err != nil {
			r.logger.Error("voice leave handler error",
				"guild_id", guildID,
				"channel_id", channelID,
				"user_id", userID,
				"error", err,
			)
		}
	}
}

