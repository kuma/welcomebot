package language

import (
	"context"
	"fmt"
	"strings"

	"welcomebot/internal/bot"
	"welcomebot/internal/core/i18n"
	"welcomebot/internal/core/logger"
	"welcomebot/internal/shared"

	"github.com/bwmarrin/discordgo"
)

const featureName = "language"

// Feature implements language configuration.
type Feature struct {
	i18n   i18n.I18n
	logger logger.Logger
}

// New creates a new language feature.
func New(deps Dependencies) (*Feature, error) {
	if err := deps.Validate(); err != nil {
		return nil, fmt.Errorf("validate dependencies: %w", err)
	}

	return &Feature{
		i18n:   deps.I18n,
		logger: deps.Logger,
	}, nil
}

// Name returns the feature name.
func (f *Feature) Name() string {
	return featureName
}

// HandleInteraction handles language selection interactions.
func (f *Feature) HandleInteraction(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	customID := extractCustomID(i)

	// Handle menu button click
	if customID == "menu:language:setup" {
		return f.showLanguagePicker(ctx, s, i)
	}

	// Handle language selection
	if strings.HasPrefix(customID, "lang:select:") {
		return f.handleLanguageSelection(ctx, s, i)
	}

	return bot.ErrNotHandled
}

// RegisterCommands returns slash commands for this feature.
func (f *Feature) RegisterCommands() []*discordgo.ApplicationCommand {
	return nil // No direct slash commands, only menu-driven
}

// GetMenuButton returns the menu button for this feature.
func (f *Feature) GetMenuButton() *bot.MenuButton {
	return &bot.MenuButton{
		Label:       "🌐 Language Settings",
		CustomID:    "menu:language:setup",
		Tier:        3,
		Category:    "admin",
		SubCategory: "configuration",
		AdminOnly:   true,
		IsCategory:  false,
	}
}

// ShowLanguagePicker shows the language selection UI.
// This is public so init feature can call it.
func (f *Feature) ShowLanguagePicker(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return f.showLanguagePicker(ctx, s, i)
}

// showLanguagePicker displays language selection buttons.
func (f *Feature) showLanguagePicker(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := buildLanguageEmbed(guildID, f.i18n)
	components := buildLanguageButtons()

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})

	if err != nil {
		return fmt.Errorf("show language picker: %w", err)
	}

	f.logger.Info("language picker shown",
		"guild_id", guildID,
		"user_id", i.Member.User.ID,
	)

	return nil
}

// handleLanguageSelection processes language selection.
func (f *Feature) handleLanguageSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	customID := i.MessageComponentData().CustomID

	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		return fmt.Errorf("invalid custom ID format")
	}

	langCode := parts[2] // "lang:select:en" → "en"

	if err := f.setLanguage(ctx, guildID, langCode); err != nil {
		return f.respondError(ctx, s, i, guildID, err)
	}

	return f.respondSuccess(ctx, s, i, guildID, langCode)
}

// setLanguage updates the guild's language preference.
func (f *Feature) setLanguage(ctx context.Context, guildID, langCode string) error {
	if err := f.i18n.SetGuildLanguage(ctx, guildID, langCode); err != nil {
		return fmt.Errorf("set guild language: %w", err)
	}

	f.logger.Info("guild language updated",
		"guild_id", guildID,
		"language", langCode,
	)

	return nil
}

// respondSuccess sends success message in selected language.
func (f *Feature) respondSuccess(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID, langCode string) error {
	langName := getLanguageName(langCode)
	msg := f.i18n.TWithArgs(ctx, guildID, "init.language_set",
		map[string]string{"language": langName})

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "common.success"),
		Description: msg,
		Color:       int(shared.ColorSuccess),
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{},
		},
	})
}

// respondError sends error message.
func (f *Feature) respondError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID string, err error) error {
	embed := &discordgo.MessageEmbed{
		Title:       "Error / エラー",
		Description: err.Error(),
		Color:       int(shared.ColorError),
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	return err
}

// buildLanguageEmbed creates the language selection embed.
func buildLanguageEmbed(guildID string, i18nSvc i18n.I18n) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title: "🌐 Language Settings / 言語設定",
		Description: "Choose your preferred language for this server.\n" +
			"このサーバーの優先言語を選択してください。",
		Color: int(shared.ColorInfo),
	}
}

// buildLanguageButtons creates language selection buttons.
func buildLanguageButtons() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "English",
					Style:    discordgo.PrimaryButton,
					CustomID: "lang:select:en",
					Emoji: &discordgo.ComponentEmoji{
						Name: "🇺🇸",
					},
				},
				discordgo.Button{
					Label:    "日本語",
					Style:    discordgo.PrimaryButton,
					CustomID: "lang:select:ja",
					Emoji: &discordgo.ComponentEmoji{
						Name: "🇯🇵",
					},
				},
			},
		},
	}
}

// getLanguageName returns display name for language code.
func getLanguageName(code string) string {
	switch code {
	case shared.LangEnglish:
		return "English"
	case shared.LangJapanese:
		return "Japanese / 日本語"
	default:
		return code
	}
}

// extractCustomID extracts custom ID from interaction.
func extractCustomID(i *discordgo.InteractionCreate) string {
	switch i.Type {
	case discordgo.InteractionMessageComponent:
		return i.MessageComponentData().CustomID
	case discordgo.InteractionModalSubmit:
		return i.ModalSubmitData().CustomID
	default:
		return ""
	}
}
