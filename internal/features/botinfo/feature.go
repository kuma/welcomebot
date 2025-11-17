package botinfo

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"welcomebot/internal/bot"
	"welcomebot/internal/core/logger"

	"github.com/bwmarrin/discordgo"
)

const (
	featureName = "botinfo"
	botVersion  = "1.0.0"
)

// Feature implements the botinfo command.
type Feature struct {
	logger    logger.Logger
	startTime time.Time
}

// New creates a new botinfo feature.
func New(deps Dependencies) (*Feature, error) {
	if err := deps.Validate(); err != nil {
		return nil, fmt.Errorf("validate dependencies: %w", err)
	}

	return &Feature{
		logger:    deps.Logger,
		startTime: time.Now(),
	}, nil
}

// Name returns the feature name.
func (f *Feature) Name() string {
	return featureName
}

// HandleInteraction handles botinfo command interactions.
func (f *Feature) HandleInteraction(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Handle slash command
	if i.Type == discordgo.InteractionApplicationCommand {
		data := i.ApplicationCommandData()
		if data.Name != "botinfo" {
			return bot.ErrNotHandled
		}
	} else if i.Type == discordgo.InteractionMessageComponent {
		// Handle menu button
		if i.MessageComponentData().CustomID != "menu:botinfo" {
			return bot.ErrNotHandled
		}
	} else {
		return bot.ErrNotHandled
	}

	f.logger.Info("botinfo command received",
		"user_id", i.Member.User.ID,
		"guild_id", i.GuildID,
	)

	embed := f.buildInfoEmbed(s)

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		return fmt.Errorf("respond to botinfo: %w", err)
	}

	return nil
}

// RegisterCommands returns the slash commands for this feature.
func (f *Feature) RegisterCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "botinfo",
			Description: "Display bot information and statistics",
		},
	}
}

// GetMenuButton returns the menu button for this feature.
func (f *Feature) GetMenuButton() *bot.MenuButton {
	return &bot.MenuButton{
		Label:       "ℹ️ Bot Info",
		CustomID:    "menu:botinfo",
		Tier:        3,
		Category:    "information",
		SubCategory: "", // Information has no sub-categories
		AdminOnly:   false,
		IsCategory:  false,
	}
}

// buildInfoEmbed creates the bot info embed.
func (f *Feature) buildInfoEmbed(s *discordgo.Session) *discordgo.MessageEmbed {
	uptime := formatUptime(time.Since(f.startTime))
	guildCount := len(s.State.Guilds)

	return &discordgo.MessageEmbed{
		Title:       "🤖 welcomebot Bot Information",
		Color:       0x7289DA,
		Description: "A modern Discord bot built with Go",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Version",
				Value:  botVersion,
				Inline: true,
			},
			{
				Name:   "Servers",
				Value:  fmt.Sprintf("%d", guildCount),
				Inline: true,
			},
			{
				Name:   "Uptime",
				Value:  uptime,
				Inline: true,
			},
			{
				Name:   "Language",
				Value:  fmt.Sprintf("Go %s", runtime.Version()),
				Inline: true,
			},
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: s.State.User.AvatarURL(""),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// formatUptime converts duration to readable format.
func formatUptime(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

