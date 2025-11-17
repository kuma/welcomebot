package bot

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

// MenuButton represents a button in the /menu interface.
type MenuButton struct {
	Label       string // Display text: "🚻 Set Gender Roles"
	CustomID    string // Button ID: "menu:admin:configuration:gender"
	Tier        int    // 1=Main category, 2=Sub-category, 3=Feature
	Category    string // Tier 1: "admin", "user" | Tier 2: parent category
	SubCategory string // Tier 2 only: "configuration", "tools", etc.
	AdminOnly   bool   // If true, only admins see this button
	IsCategory  bool   // If true, navigates to sub-menu; if false, triggers feature
}

// Feature defines the interface all bot features must implement.
type Feature interface {
	// Name returns the unique name of this feature.
	Name() string

	// HandleInteraction handles slash commands and component interactions.
	HandleInteraction(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error

	// RegisterCommands returns slash commands to register with Discord.
	RegisterCommands() []*discordgo.ApplicationCommand

	// GetMenuButton returns the menu button for this feature.
	// Return nil if feature should not appear in /menu.
	GetMenuButton() *MenuButton
}

// MessageFeature is an optional interface for features that handle messages.
type MessageFeature interface {
	Feature
	HandleMessage(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) error
}

// ReactionFeature is an optional interface for features that handle reactions.
type ReactionFeature interface {
	Feature
	HandleReactionAdd(ctx context.Context, s *discordgo.Session, r *discordgo.MessageReactionAdd) error
	HandleReactionRemove(ctx context.Context, s *discordgo.Session, r *discordgo.MessageReactionRemove) error
}

// VoiceFeature is an optional interface for features that handle voice state.
type VoiceFeature interface {
	Feature
	HandleVoiceStateUpdate(ctx context.Context, s *discordgo.Session, v *discordgo.VoiceStateUpdate) error
}

// MemberFeature is an optional interface for features that handle member events.
type MemberFeature interface {
	Feature
	HandleMemberJoin(ctx context.Context, s *discordgo.Session, m *discordgo.GuildMemberAdd) error
	HandleMemberLeave(ctx context.Context, s *discordgo.Session, m *discordgo.GuildMemberRemove) error
}
