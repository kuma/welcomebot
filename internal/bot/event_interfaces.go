package bot

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

// MessageEventFeature handles message events with indexed routing (high-frequency).
type MessageEventFeature interface {
	Feature
	// RegisterMessageHandlers registers handlers for specific channels
	RegisterMessageHandlers(router MessageRouter, guildID string) error
	// UnregisterMessageHandlers cleans up handlers
	UnregisterMessageHandlers(router MessageRouter, guildID string) error
}

// VoiceEventFeature handles voice events with indexed routing (high-frequency).
type VoiceEventFeature interface {
	Feature
	// RegisterVoiceHandlers registers handlers for specific voice channels
	RegisterVoiceHandlers(router VoiceRouter, guildID string) error
	// UnregisterVoiceHandlers cleans up handlers
	UnregisterVoiceHandlers(router VoiceRouter, guildID string) error
}

// MemberEventFeature handles member events with filtered routing (low-frequency).
// Already defined in feature.go as MemberFeature, keeping for compatibility

// ReactionEventFeature handles reaction events with filtered routing (low-frequency).
// Already defined in feature.go as ReactionFeature, keeping for compatibility

// MessageRouter provides message event registration.
type MessageRouter interface {
	OnMessageCreate(guildID, channelID string, handler MessageCreateHandler)
	OnMessageDelete(guildID, channelID string, handler MessageDeleteHandler)
	OffMessage(guildID, channelID string)
	OffMessageGuild(guildID string)
}

// VoiceRouter provides voice event registration.
type VoiceRouter interface {
	OnVoiceJoin(guildID, channelID string, handler VoiceJoinHandler)
	OnVoiceLeave(guildID, channelID string, handler VoiceLeaveHandler)
	OffVoice(guildID, channelID string)
	OffVoiceGuild(guildID string)
}

// Handler function types.
type MessageCreateHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) error
type MessageDeleteHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.MessageDelete) error
type VoiceJoinHandler func(ctx context.Context, s *discordgo.Session, userID string, v *discordgo.VoiceStateUpdate) error
type VoiceLeaveHandler func(ctx context.Context, s *discordgo.Session, userID string, v *discordgo.VoiceStateUpdate) error

