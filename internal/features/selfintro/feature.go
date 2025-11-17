package selfintro

import (
	"context"
	"fmt"
	"strings"
	"time"

	"welcomebot/internal/bot"
	"welcomebot/internal/core/cache"
	"welcomebot/internal/core/database"
	"welcomebot/internal/core/i18n"
	"welcomebot/internal/core/logger"
	"welcomebot/internal/shared"

	"github.com/bwmarrin/discordgo"
)

const featureName = "selfintro"

// Feature implements self-intro channel configuration.
type Feature struct {
	db     database.Client
	cache  cache.Client
	i18n   i18n.I18n
	logger logger.Logger
}

// New creates a new selfintro feature.
func New(deps Dependencies) (*Feature, error) {
	if err := deps.Validate(); err != nil {
		return nil, fmt.Errorf("validate dependencies: %w", err)
	}

	return &Feature{
		db:     deps.DB,
		cache:  deps.Cache,
		i18n:   deps.I18n,
		logger: deps.Logger,
	}, nil
}

// Name returns the feature name.
func (f *Feature) Name() string {
	return featureName
}

// HandleInteraction handles selfintro configuration interactions.
func (f *Feature) HandleInteraction(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	customID := extractCustomID(i)
	guildID := i.GuildID

	if customID == "menu:selfintro:setup" {
		return f.startWizard(ctx, s, i)
	}

	if customID == "selfintro:confirm_overwrite" {
		return f.showStep1(ctx, s, i)
	}

	if customID == "selfintro:cancel" {
		return f.respondCancelled(ctx, s, i, guildID)
	}

	if strings.HasPrefix(customID, "selfintro:male:") {
		return f.handleMaleChannelSelection(ctx, s, i)
	}

	if strings.HasPrefix(customID, "selfintro:female:") {
		return f.handleFemaleChannelSelection(ctx, s, i)
	}

	return bot.ErrNotHandled
}

// RegisterCommands returns slash commands for this feature.
func (f *Feature) RegisterCommands() []*discordgo.ApplicationCommand {
	return nil
}

// GetMenuButton returns the menu button for this feature.
func (f *Feature) GetMenuButton() *bot.MenuButton {
	return nil // Hidden from menu
}

// startWizard initiates the configuration wizard.
func (f *Feature) startWizard(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	config, err := f.getConfig(ctx, guildID)
	if err == nil && config != nil {
		return f.showOverwriteConfirmation(ctx, s, i, config)
	}

	return f.showStep1(ctx, s, i)
}

// showOverwriteConfirmation shows confirmation for overwriting.
func (f *Feature) showOverwriteConfirmation(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, config *SelfIntroConfig) error {
	guildID := i.GuildID

	maleChannel := fmt.Sprintf("<#%s>", config.MaleChannelID)
	femaleChannel := fmt.Sprintf("<#%s>", config.FemaleChannelID)

	desc := f.i18n.TWithArgs(ctx, guildID, "selfintro.current_config",
		map[string]string{
			"male":   maleChannel,
			"female": femaleChannel,
		})

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "selfintro.overwrite_title"),
		Description: desc,
		Color:       int(shared.ColorWarning),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    f.i18n.T(ctx, guildID, "selfintro.reconfigure"),
					Style:    discordgo.DangerButton,
					CustomID: "selfintro:confirm_overwrite",
				},
				discordgo.Button{
					Label:    f.i18n.T(ctx, guildID, "common.cancel"),
					Style:    discordgo.SecondaryButton,
					CustomID: "selfintro:cancel",
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep1 shows male channel selection.
func (f *Feature) showStep1(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "selfintro.step1_title"),
		Description: f.i18n.T(ctx, guildID, "selfintro.step1_description"),
		Color:       int(shared.ColorInfo),
	}

	components := f.buildChannelSelectMenu(ctx, guildID, s, "selfintro:male:", "selfintro.select_male")

	return respond(s, i, embed, components)
}

// showStep2 shows female channel selection.
func (f *Feature) showStep2(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, maleChannelID string) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "selfintro.step2_title"),
		Description: f.i18n.T(ctx, guildID, "selfintro.step2_description"),
		Color:       int(shared.ColorInfo),
	}

	customIDPrefix := fmt.Sprintf("selfintro:female:%s:", maleChannelID)
	components := f.buildChannelSelectMenu(ctx, guildID, s, customIDPrefix, "selfintro.select_female")

	return respond(s, i, embed, components)
}

// handleMaleChannelSelection processes male channel selection.
func (f *Feature) handleMaleChannelSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return fmt.Errorf("no channel selected")
	}

	maleChannelID := values[0]

	f.logger.Info("male channel selected",
		"guild_id", i.GuildID,
		"channel_id", maleChannelID,
	)

	return f.showStep2(ctx, s, i, maleChannelID)
}

// handleFemaleChannelSelection processes female channel selection.
func (f *Feature) handleFemaleChannelSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	customID := i.MessageComponentData().CustomID
	values := i.MessageComponentData().Values

	if len(values) == 0 {
		return fmt.Errorf("no channel selected")
	}

	femaleChannelID := values[0]

	maleChannelID, err := extractMaleChannelID(customID)
	if err != nil {
		return err
	}

	if maleChannelID == femaleChannelID {
		return f.respondValidationError(ctx, s, i, guildID)
	}

	if err := f.saveConfig(ctx, guildID, maleChannelID, femaleChannelID); err != nil {
		return f.respondError(ctx, s, i, guildID, err)
	}

	return f.respondSuccess(ctx, s, i, guildID, maleChannelID, femaleChannelID)
}

// saveConfig saves configuration to database and cache.
func (f *Feature) saveConfig(ctx context.Context, guildID, maleChannelID, femaleChannelID string) error {
	query := `
		INSERT INTO guild_selfintro_channels (guild_id, male_channel_id, female_channel_id, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (guild_id)
		DO UPDATE SET male_channel_id = $2, female_channel_id = $3, updated_at = NOW()
	`

	_, err := f.db.Exec(ctx, query, guildID, maleChannelID, femaleChannelID)
	if err != nil {
		return fmt.Errorf("save to database: %w", err)
	}

	config := &SelfIntroConfig{
		GuildID:         guildID,
		MaleChannelID:   maleChannelID,
		FemaleChannelID: femaleChannelID,
		UpdatedAt:       time.Now(),
	}

	cacheKey := cacheKeyPrefix + guildID
	if err := f.cache.SetJSON(ctx, cacheKey, config, 0); err != nil {
		f.logger.Warn("failed to cache selfintro config", "error", err)
	}

	f.logger.Info("selfintro channels configured",
		"guild_id", guildID,
		"male_channel", maleChannelID,
		"female_channel", femaleChannelID,
	)

	return nil
}

// getConfig retrieves configuration from cache or database.
func (f *Feature) getConfig(ctx context.Context, guildID string) (*SelfIntroConfig, error) {
	cacheKey := cacheKeyPrefix + guildID

	var config SelfIntroConfig
	if err := f.cache.GetJSON(ctx, cacheKey, &config); err == nil {
		return &config, nil
	}

	query := "SELECT guild_id, male_channel_id, female_channel_id, created_at, updated_at FROM guild_selfintro_channels WHERE guild_id = $1"
	row := f.db.QueryRow(ctx, query, guildID)

	err := row.Scan(&config.GuildID, &config.MaleChannelID, &config.FemaleChannelID, &config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		return nil, err
	}

	f.cache.SetJSON(ctx, cacheKey, &config, 0)

	return &config, nil
}

// buildChannelSelectMenu builds a channel select menu for text channels.
func (f *Feature) buildChannelSelectMenu(ctx context.Context, guildID string, s *discordgo.Session, customIDPrefix, placeholderKey string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:      discordgo.ChannelSelectMenu,
					CustomID:      customIDPrefix + "select",
					Placeholder:   f.i18n.T(ctx, guildID, placeholderKey),
					ChannelTypes: []discordgo.ChannelType{
						discordgo.ChannelTypeGuildText,
					},
					DefaultValues: []discordgo.SelectMenuDefaultValue{}, // Explicitly empty
				},
			},
		},
	}
}

// extractMaleChannelID extracts male channel ID from CustomID.
func extractMaleChannelID(customID string) (string, error) {
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid customID format")
	}
	return parts[2], nil
}

// extractCustomID extracts custom ID from interaction.
func extractCustomID(i *discordgo.InteractionCreate) string {
	switch i.Type {
	case discordgo.InteractionMessageComponent:
		return i.MessageComponentData().CustomID
	default:
		return ""
	}
}

// respond sends an interaction response.
func respond(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
	responseType := discordgo.InteractionResponseChannelMessageWithSource
	if i.Type == discordgo.InteractionMessageComponent {
		responseType = discordgo.InteractionResponseUpdateMessage
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: responseType,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

// respondSuccess sends success message.
func (f *Feature) respondSuccess(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID, maleChannelID, femaleChannelID string) error {
	desc := f.i18n.TWithArgs(ctx, guildID, "selfintro.success",
		map[string]string{
			"male":   fmt.Sprintf("<#%s>", maleChannelID),
			"female": fmt.Sprintf("<#%s>", femaleChannelID),
		})

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "common.success"),
		Description: desc,
		Color:       int(shared.ColorSuccess),
	}

	return respond(s, i, embed, []discordgo.MessageComponent{})
}

// respondValidationError sends validation error.
func (f *Feature) respondValidationError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) error {
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "common.error"),
		Description: f.i18n.T(ctx, guildID, "selfintro.same_channel_error"),
		Color:       int(shared.ColorError),
	}

	return respond(s, i, embed, []discordgo.MessageComponent{})
}

// respondError sends error message.
func (f *Feature) respondError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID string, err error) error {
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "common.error"),
		Description: f.i18n.T(ctx, guildID, "errors.database_error"),
		Color:       int(shared.ColorError),
	}

	f.logger.Error("selfintro configuration error",
		"guild_id", guildID,
		"error", err,
	)

	return respond(s, i, embed, []discordgo.MessageComponent{})
}

// respondCancelled sends cancellation message.
func (f *Feature) respondCancelled(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) error {
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "common.cancelled"),
		Description: f.i18n.T(ctx, guildID, "selfintro.cancelled"),
		Color:       int(shared.ColorInfo),
	}

	return respond(s, i, embed, []discordgo.MessageComponent{})
}

