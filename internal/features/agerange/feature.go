package agerange

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

const featureName = "agerange"

// Feature implements age range role configuration.
type Feature struct {
	db     database.Client
	cache  cache.Client
	i18n   i18n.I18n
	logger logger.Logger
}

// New creates a new age range feature.
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

// HandleInteraction handles age range configuration interactions.
func (f *Feature) HandleInteraction(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	customID := extractCustomID(i)
	guildID := i.GuildID

	// Menu button click - start configuration wizard
	if customID == "menu:agerange:setup" {
		return f.startWizard(ctx, s, i)
	}

	// Overwrite confirmation
	if customID == "agerange:confirm_overwrite" {
		return f.showStep1(ctx, s, i)
	}

	if customID == "agerange:cancel" {
		return f.respondCancelled(ctx, s, i, guildID)
	}

	// Step 1: 20代前半 role selection
	if strings.HasPrefix(customID, "agerange:age_20_early_role:") {
		return f.handleAge20EarlyRoleSelection(ctx, s, i)
	}

	// Step 2: 20代後半 role selection
	if strings.HasPrefix(customID, "agerange:age_20_late_role:") {
		return f.handleAge20LateRoleSelection(ctx, s, i)
	}

	// Step 3: 30代前半 role selection
	if strings.HasPrefix(customID, "agerange:age_30_early_role:") {
		return f.handleAge30EarlyRoleSelection(ctx, s, i)
	}

	// Step 4: 30代後半 role selection
	if strings.HasPrefix(customID, "agerange:age_30_late_role:") {
		return f.handleAge30LateRoleSelection(ctx, s, i)
	}

	// Step 5: 40代前半 role selection
	if strings.HasPrefix(customID, "agerange:age_40_early_role:") {
		return f.handleAge40EarlyRoleSelection(ctx, s, i)
	}

	// Step 6: 40代後半 role selection
	if strings.HasPrefix(customID, "agerange:age_40_late_role:") {
		return f.handleAge40LateRoleSelection(ctx, s, i)
	}

	return bot.ErrNotHandled
}

// RegisterCommands returns slash commands for this feature.
func (f *Feature) RegisterCommands() []*discordgo.ApplicationCommand {
	return nil // Menu-driven only
}

// GetMenuButton returns the menu button for this feature.
func (f *Feature) GetMenuButton() *bot.MenuButton {
	return &bot.MenuButton{
		Label:       "📅 Set Age Range Roles",
		CustomID:    "menu:agerange:setup",
		Tier:        3,
		Category:    "admin",
		SubCategory: "configuration",
		AdminOnly:   true,
		IsCategory:  false,
	}
}

// getWizardState retrieves wizard state from cache.
func (f *Feature) getWizardState(ctx context.Context, guildID string) (*WizardState, error) {
	key := fmt.Sprintf("welcomebot:agerange:wizard:%s", guildID)
	var state WizardState
	if err := f.cache.GetJSON(ctx, key, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// saveWizardState saves wizard state to cache.
func (f *Feature) saveWizardState(ctx context.Context, state *WizardState) error {
	key := fmt.Sprintf("welcomebot:agerange:wizard:%s", state.GuildID)
	return f.cache.SetJSON(ctx, key, state, 30*time.Minute)
}

// deleteWizardState removes wizard state from cache.
func (f *Feature) deleteWizardState(ctx context.Context, guildID string) error {
	key := fmt.Sprintf("welcomebot:agerange:wizard:%s", guildID)
	return f.cache.Delete(ctx, key)
}

// startWizard initiates the age range configuration wizard.
func (f *Feature) startWizard(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	config, err := f.getAgeRangeConfig(ctx, guildID)
	if err == nil && config != nil {
		return f.showOverwriteConfirmation(ctx, s, i, config)
	}

	// Initialize wizard state
	state := &WizardState{
		GuildID:     guildID,
		CurrentStep: 1,
	}
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep1(ctx, s, i)
}

// showOverwriteConfirmation shows confirmation for overwriting existing config.
func (f *Feature) showOverwriteConfirmation(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, config *AgeRangeConfig) error {
	guildID := i.GuildID

	desc := f.i18n.T(ctx, guildID, "agerange.current_config")

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "agerange.overwrite_title"),
		Description: desc,
		Color:       int(shared.ColorWarning),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    f.i18n.T(ctx, guildID, "agerange.reconfigure"),
					Style:    discordgo.DangerButton,
					CustomID: "agerange:confirm_overwrite",
				},
				discordgo.Button{
					Label:    f.i18n.T(ctx, guildID, "common.cancel"),
					Style:    discordgo.SecondaryButton,
					CustomID: "agerange:cancel",
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep1 shows 20代前半 role selection.
func (f *Feature) showStep1(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "agerange.step1_title"),
		Description: f.i18n.T(ctx, guildID, "agerange.step1_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "agerange:age_20_early_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "agerange.select_age_20_early_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep2 shows 20代後半 role selection.
func (f *Feature) showStep2(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "agerange.step2_title"),
		Description: f.i18n.T(ctx, guildID, "agerange.step2_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "agerange:age_20_late_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "agerange.select_age_20_late_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep3 shows 30代前半 role selection.
func (f *Feature) showStep3(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "agerange.step3_title"),
		Description: f.i18n.T(ctx, guildID, "agerange.step3_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "agerange:age_30_early_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "agerange.select_age_30_early_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep4 shows 30代後半 role selection.
func (f *Feature) showStep4(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "agerange.step4_title"),
		Description: f.i18n.T(ctx, guildID, "agerange.step4_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "agerange:age_30_late_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "agerange.select_age_30_late_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep5 shows 40代前半 role selection.
func (f *Feature) showStep5(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "agerange.step5_title"),
		Description: f.i18n.T(ctx, guildID, "agerange.step5_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "agerange:age_40_early_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "agerange.select_age_40_early_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep6 shows 40代後半 role selection.
func (f *Feature) showStep6(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "agerange.step6_title"),
		Description: f.i18n.T(ctx, guildID, "agerange.step6_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "agerange:age_40_late_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "agerange.select_age_40_late_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// handleAge20EarlyRoleSelection processes 20代前半 role selection.
func (f *Feature) handleAge20EarlyRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	values := i.MessageComponentData().Values

	if len(values) == 0 {
		return fmt.Errorf("no role selected")
	}

	roleID := values[0]

	// Update wizard state
	state, err := f.getWizardState(ctx, guildID)
	if err != nil {
		state = &WizardState{GuildID: guildID}
	}
	state.Age20EarlyRoleID = roleID
	state.CurrentStep = 2
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep2(ctx, s, i)
}

// handleAge20LateRoleSelection processes 20代後半 role selection.
func (f *Feature) handleAge20LateRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	values := i.MessageComponentData().Values

	if len(values) == 0 {
		return fmt.Errorf("no role selected")
	}

	roleID := values[0]

	// Update wizard state
	state, err := f.getWizardState(ctx, guildID)
	if err != nil {
		return fmt.Errorf("get wizard state: %w", err)
	}
	state.Age20LateRoleID = roleID
	state.CurrentStep = 3
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep3(ctx, s, i)
}

// handleAge30EarlyRoleSelection processes 30代前半 role selection.
func (f *Feature) handleAge30EarlyRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	values := i.MessageComponentData().Values

	if len(values) == 0 {
		return fmt.Errorf("no role selected")
	}

	roleID := values[0]

	// Update wizard state
	state, err := f.getWizardState(ctx, guildID)
	if err != nil {
		return fmt.Errorf("get wizard state: %w", err)
	}
	state.Age30EarlyRoleID = roleID
	state.CurrentStep = 4
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep4(ctx, s, i)
}

// handleAge30LateRoleSelection processes 30代後半 role selection.
func (f *Feature) handleAge30LateRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	values := i.MessageComponentData().Values

	if len(values) == 0 {
		return fmt.Errorf("no role selected")
	}

	roleID := values[0]

	// Update wizard state
	state, err := f.getWizardState(ctx, guildID)
	if err != nil {
		return fmt.Errorf("get wizard state: %w", err)
	}
	state.Age30LateRoleID = roleID
	state.CurrentStep = 5
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep5(ctx, s, i)
}

// handleAge40EarlyRoleSelection processes 40代前半 role selection.
func (f *Feature) handleAge40EarlyRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	values := i.MessageComponentData().Values

	if len(values) == 0 {
		return fmt.Errorf("no role selected")
	}

	roleID := values[0]

	// Update wizard state
	state, err := f.getWizardState(ctx, guildID)
	if err != nil {
		return fmt.Errorf("get wizard state: %w", err)
	}
	state.Age40EarlyRoleID = roleID
	state.CurrentStep = 6
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep6(ctx, s, i)
}

// handleAge40LateRoleSelection processes 40代後半 role selection and completes wizard.
func (f *Feature) handleAge40LateRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	values := i.MessageComponentData().Values

	if len(values) == 0 {
		return fmt.Errorf("no role selected")
	}

	roleID := values[0]

	// Get final wizard state
	state, err := f.getWizardState(ctx, guildID)
	if err != nil {
		return fmt.Errorf("get wizard state: %w", err)
	}
	state.Age40LateRoleID = roleID

	// Convert wizard state to config and save
	config := &AgeRangeConfig{
		GuildID:          guildID,
		Age20EarlyRoleID: state.Age20EarlyRoleID,
		Age20LateRoleID:  state.Age20LateRoleID,
		Age30EarlyRoleID: state.Age30EarlyRoleID,
		Age30LateRoleID:  state.Age30LateRoleID,
		Age40EarlyRoleID: state.Age40EarlyRoleID,
		Age40LateRoleID:  state.Age40LateRoleID,
	}

	if err := f.saveAgeRangeConfig(ctx, config); err != nil {
		return f.respondError(ctx, s, i, guildID, err)
	}

	// Delete wizard state
	if err := f.deleteWizardState(ctx, guildID); err != nil {
		f.logger.Error("failed to delete wizard state", "error", err)
	}

	return f.respondSuccess(ctx, s, i, guildID)
}

// saveAgeRangeConfig saves age range configuration to database and cache.
func (f *Feature) saveAgeRangeConfig(ctx context.Context, config *AgeRangeConfig) error {
	query := `
		INSERT INTO guild_age_range_config (
			guild_id, age_20_early_role_id, age_20_late_role_id,
			age_30_early_role_id, age_30_late_role_id,
			age_40_early_role_id, age_40_late_role_id, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (guild_id)
		DO UPDATE SET 
			age_20_early_role_id = $2,
			age_20_late_role_id = $3,
			age_30_early_role_id = $4,
			age_30_late_role_id = $5,
			age_40_early_role_id = $6,
			age_40_late_role_id = $7,
			updated_at = NOW()
	`

	_, err := f.db.Exec(ctx, query,
		config.GuildID,
		config.Age20EarlyRoleID,
		config.Age20LateRoleID,
		config.Age30EarlyRoleID,
		config.Age30LateRoleID,
		config.Age40EarlyRoleID,
		config.Age40LateRoleID,
	)
	if err != nil {
		return fmt.Errorf("save to database: %w", err)
	}

	config.UpdatedAt = time.Now()

	// Cache configuration
	cacheKey := cacheKeyPrefix + config.GuildID
	if err := f.cache.SetJSON(ctx, cacheKey, config, 0); err != nil {
		f.logger.Warn("failed to cache age range config", "error", err)
	}

	f.logger.Info("age range config saved", "guild_id", config.GuildID)

	return nil
}

// getAgeRangeConfig retrieves age range configuration.
func (f *Feature) getAgeRangeConfig(ctx context.Context, guildID string) (*AgeRangeConfig, error) {
	cacheKey := cacheKeyPrefix + guildID

	var config AgeRangeConfig
	if err := f.cache.GetJSON(ctx, cacheKey, &config); err == nil {
		return &config, nil
	}

	query := `
		SELECT guild_id, age_20_early_role_id, age_20_late_role_id,
		       age_30_early_role_id, age_30_late_role_id,
		       age_40_early_role_id, age_40_late_role_id,
		       created_at, updated_at
		FROM guild_age_range_config 
		WHERE guild_id = $1
	`
	row := f.db.QueryRow(ctx, query, guildID)

	var age20Early, age20Late, age30Early, age30Late, age40Early, age40Late *string
	err := row.Scan(&config.GuildID,
		&age20Early, &age20Late,
		&age30Early, &age30Late,
		&age40Early, &age40Late,
		&config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if age20Early != nil {
		config.Age20EarlyRoleID = *age20Early
	}
	if age20Late != nil {
		config.Age20LateRoleID = *age20Late
	}
	if age30Early != nil {
		config.Age30EarlyRoleID = *age30Early
	}
	if age30Late != nil {
		config.Age30LateRoleID = *age30Late
	}
	if age40Early != nil {
		config.Age40EarlyRoleID = *age40Early
	}
	if age40Late != nil {
		config.Age40LateRoleID = *age40Late
	}

	f.cache.SetJSON(ctx, cacheKey, &config, 0)

	return &config, nil
}

// respondSuccess sends success message.
func (f *Feature) respondSuccess(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) error {
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "common.success"),
		Description: f.i18n.T(ctx, guildID, "agerange.success"),
		Color:       int(shared.ColorSuccess),
	}

	return respond(s, i, embed, []discordgo.MessageComponent{})
}

// respondError sends error message.
func (f *Feature) respondError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID string, err error) error {
	f.logger.Error("age range configuration error", "error", err)

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "common.error"),
		Description: f.i18n.T(ctx, guildID, "agerange.error_save"),
		Color:       int(shared.ColorError),
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

// respondCancelled sends cancellation message.
func (f *Feature) respondCancelled(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) error {
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "common.cancelled"),
		Description: f.i18n.T(ctx, guildID, "agerange.cancelled"),
		Color:       int(shared.ColorInfo),
	}

	return respond(s, i, embed, []discordgo.MessageComponent{})
}

// respond is a helper to respond to interactions.
func respond(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

// extractCustomID extracts the custom ID from an interaction.
func extractCustomID(i *discordgo.InteractionCreate) string {
	if i.Type == discordgo.InteractionMessageComponent {
		return i.MessageComponentData().CustomID
	}
	return ""
}

