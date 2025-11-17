package ping

import (
	"context"
	"fmt"
	"time"

	"welcomebot/internal/bot"
	"welcomebot/internal/core/logger"

	"github.com/bwmarrin/discordgo"
)

const featureName = "ping"

// Feature implements the ping command.
type Feature struct {
	logger logger.Logger
}

// New creates a new ping feature.
func New(deps Dependencies) (*Feature, error) {
	if err := deps.Validate(); err != nil {
		return nil, fmt.Errorf("validate dependencies: %w", err)
	}

	return &Feature{
		logger: deps.Logger,
	}, nil
}

// Name returns the feature name.
func (f *Feature) Name() string {
	return featureName
}

// HandleInteraction handles ping command interactions.
func (f *Feature) HandleInteraction(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Handle slash command
	if i.Type == discordgo.InteractionApplicationCommand {
		data := i.ApplicationCommandData()
		if data.Name != "ping" {
			return bot.ErrNotHandled
		}
	} else if i.Type == discordgo.InteractionMessageComponent {
		// Handle menu button
		if i.MessageComponentData().CustomID != "menu:ping" {
			return bot.ErrNotHandled
		}
	} else {
		return bot.ErrNotHandled
	}

	f.logger.Info("ping command received",
		"user_id", i.Member.User.ID,
		"guild_id", i.GuildID,
	)

	// Calculate latency
	startTime := time.Now()
	latency := s.HeartbeatLatency()

	// Respond with pong message
	embed := buildPongEmbed(latency, startTime)

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})

	if err != nil {
		return fmt.Errorf("respond to ping: %w", err)
	}

	return nil
}

// RegisterCommands returns the slash commands for this feature.
func (f *Feature) RegisterCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "ping",
			Description: "Check bot latency and responsiveness",
		},
	}
}

// GetMenuButton returns the menu button for this feature.
func (f *Feature) GetMenuButton() *bot.MenuButton {
	return &bot.MenuButton{
		Label:       "🏓 Ping",
		CustomID:    "menu:ping",
		Tier:        3,
		Category:    "information",
		SubCategory: "", // Information has no sub-categories
		AdminOnly:   false,
		IsCategory:  false,
	}
}

// buildPongEmbed creates the pong response embed.
func buildPongEmbed(heartbeat time.Duration, startTime time.Time) *discordgo.MessageEmbed {
	apiLatency := time.Since(startTime)

	return &discordgo.MessageEmbed{
		Title: "🏓 Pong!",
		Color: 0x00AAFF,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "WebSocket Latency",
				Value:  fmt.Sprintf("%dms", heartbeat.Milliseconds()),
				Inline: true,
			},
			{
				Name:   "API Latency",
				Value:  fmt.Sprintf("%dms", apiLatency.Milliseconds()),
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

