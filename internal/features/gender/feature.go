package gender

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

const featureName = "gender"

// Feature implements gender role configuration.
type Feature struct {
	db     database.Client
	cache  cache.Client
	i18n   i18n.I18n
	logger logger.Logger
}

// New creates a new gender feature.
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

// HandleInteraction handles gender configuration interactions.
func (f *Feature) HandleInteraction(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	customID := extractCustomID(i)
	guildID := i.GuildID

	// Menu button click
	if customID == "menu:gender:setup" {
		return f.startWizard(ctx, s, i)
	}

	// Overwrite confirmation
	if customID == "gender:confirm_overwrite" {
		return f.showStep1(ctx, s, i)
	}

	if customID == "gender:cancel" {
		return f.respondCancelled(ctx, s, i, guildID)
	}

	// Step 1: Male role selection
	if strings.HasPrefix(customID, "gender:male:") {
		return f.handleMaleRoleSelection(ctx, s, i)
	}

	// Step 2: Female role selection
	if strings.HasPrefix(customID, "gender:female:") {
		return f.handleFemaleRoleSelection(ctx, s, i)
	}

	return bot.ErrNotHandled
}

// RegisterCommands returns slash commands for this feature.
func (f *Feature) RegisterCommands() []*discordgo.ApplicationCommand {
	return nil // Menu-driven only
}

// GetMenuButton returns the menu button for this feature.
func (f *Feature) GetMenuButton() *bot.MenuButton {
	return nil // Hidden from menu
}

// startWizard initiates the gender configuration wizard.
func (f *Feature) startWizard(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	config, err := f.getGenderConfig(ctx, guildID)
	if err == nil && config != nil {
		return f.showOverwriteConfirmation(ctx, s, i, config)
	}

	return f.showStep1(ctx, s, i)
}

// showOverwriteConfirmation shows confirmation for overwriting.
func (f *Feature) showOverwriteConfirmation(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, config *GenderConfig) error {
	guildID := i.GuildID

	maleRole := fmt.Sprintf("<@&%s>", config.MaleRoleID)
	femaleRole := fmt.Sprintf("<@&%s>", config.FemaleRoleID)

	desc := f.i18n.TWithArgs(ctx, guildID, "gender.current_config",
		map[string]string{
			"male":   maleRole,
			"female": femaleRole,
		})

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "gender.overwrite_title"),
		Description: desc,
		Color:       int(shared.ColorWarning),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    f.i18n.T(ctx, guildID, "gender.reconfigure"),
					Style:    discordgo.DangerButton,
					CustomID: "gender:confirm_overwrite",
				},
				discordgo.Button{
					Label:    f.i18n.T(ctx, guildID, "common.cancel"),
					Style:    discordgo.SecondaryButton,
					CustomID: "gender:cancel",
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep1 shows male role selection.
func (f *Feature) showStep1(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "gender.step1_title"),
		Description: f.i18n.T(ctx, guildID, "gender.step1_description"),
		Color:       int(shared.ColorInfo),
	}

	components := f.buildRoleSelectMenu(ctx, guildID, s, "gender:male:", "gender.select_male")

	return respond(s, i, embed, components)
}

// showStep2 shows female role selection.
func (f *Feature) showStep2(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, maleRoleID string) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "gender.step2_title"),
		Description: f.i18n.T(ctx, guildID, "gender.step2_description"),
		Color:       int(shared.ColorInfo),
	}

	customIDPrefix := fmt.Sprintf("gender:female:%s:", maleRoleID)
	components := f.buildRoleSelectMenu(ctx, guildID, s, customIDPrefix, "gender.select_female")

	return respond(s, i, embed, components)
}

// handleMaleRoleSelection processes male role selection.
func (f *Feature) handleMaleRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return fmt.Errorf("no role selected")
	}

	maleRoleID := values[0]

	f.logger.Info("male role selected",
		"guild_id", i.GuildID,
		"role_id", maleRoleID,
	)

	return f.showStep2(ctx, s, i, maleRoleID)
}

// handleFemaleRoleSelection processes female role selection.
func (f *Feature) handleFemaleRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	customID := i.MessageComponentData().CustomID
	values := i.MessageComponentData().Values

	if len(values) == 0 {
		return fmt.Errorf("no role selected")
	}

	femaleRoleID := values[0]

	// Extract male role ID from CustomID
	maleRoleID, err := extractMaleRoleID(customID)
	if err != nil {
		return err
	}

	// Validate: roles must be different
	if maleRoleID == femaleRoleID {
		return f.respondValidationError(ctx, s, i, guildID)
	}

	// Save configuration
	if err := f.saveGenderConfig(ctx, guildID, maleRoleID, femaleRoleID); err != nil {
		return f.respondError(ctx, s, i, guildID, err)
	}

	return f.respondSuccess(ctx, s, i, guildID, maleRoleID, femaleRoleID)
}

// saveGenderConfig saves gender configuration to database and cache.
func (f *Feature) saveGenderConfig(ctx context.Context, guildID, maleRoleID, femaleRoleID string) error {
	query := `
		INSERT INTO guild_gender_roles (guild_id, male_role_id, female_role_id, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (guild_id)
		DO UPDATE SET male_role_id = $2, female_role_id = $3, updated_at = NOW()
	`

	_, err := f.db.Exec(ctx, query, guildID, maleRoleID, femaleRoleID)
	if err != nil {
		return fmt.Errorf("save to database: %w", err)
	}

	// Cache indefinitely
	config := &GenderConfig{
		GuildID:      guildID,
		MaleRoleID:   maleRoleID,
		FemaleRoleID: femaleRoleID,
		UpdatedAt:    time.Now(),
	}

	cacheKey := cacheKeyPrefix + guildID
	if err := f.cache.SetJSON(ctx, cacheKey, config, 0); err != nil {
		f.logger.Warn("failed to cache gender config", "error", err)
	}

	f.logger.Info("gender roles configured",
		"guild_id", guildID,
		"male_role", maleRoleID,
		"female_role", femaleRoleID,
	)

	return nil
}

// getGenderConfig retrieves gender configuration.
func (f *Feature) getGenderConfig(ctx context.Context, guildID string) (*GenderConfig, error) {
	cacheKey := cacheKeyPrefix + guildID

	var config GenderConfig
	if err := f.cache.GetJSON(ctx, cacheKey, &config); err == nil {
		return &config, nil
	}

	query := "SELECT guild_id, male_role_id, female_role_id, created_at, updated_at FROM guild_gender_roles WHERE guild_id = $1"
	row := f.db.QueryRow(ctx, query, guildID)

	err := row.Scan(&config.GuildID, &config.MaleRoleID, &config.FemaleRoleID, &config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		return nil, err
	}

	f.cache.SetJSON(ctx, cacheKey, &config, 0)

	return &config, nil
}

// buildRoleSelectMenu builds a role select menu.
func (f *Feature) buildRoleSelectMenu(ctx context.Context, guildID string, s *discordgo.Session, customIDPrefix, placeholderKey string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:      discordgo.RoleSelectMenu,
					CustomID:      customIDPrefix + "select",
					Placeholder:   f.i18n.T(ctx, guildID, placeholderKey),
					DefaultValues: []discordgo.SelectMenuDefaultValue{}, // Explicitly empty
				},
			},
		},
	}
}

// extractMaleRoleID extracts male role ID from CustomID.
func extractMaleRoleID(customID string) (string, error) {
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
func (f *Feature) respondSuccess(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID, maleRoleID, femaleRoleID string) error {
	desc := f.i18n.TWithArgs(ctx, guildID, "gender.success",
		map[string]string{
			"male":   fmt.Sprintf("<@&%s>", maleRoleID),
			"female": fmt.Sprintf("<@&%s>", femaleRoleID),
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
		Description: f.i18n.T(ctx, guildID, "gender.same_role_error"),
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

	f.logger.Error("gender configuration error",
		"guild_id", guildID,
		"error", err,
	)

	return respond(s, i, embed, []discordgo.MessageComponent{})
}

// respondCancelled sends cancellation message.
func (f *Feature) respondCancelled(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) error {
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "common.cancelled"),
		Description: f.i18n.T(ctx, guildID, "gender.cancelled"),
		Color:       int(shared.ColorInfo),
	}

	return respond(s, i, embed, []discordgo.MessageComponent{})
}

