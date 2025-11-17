package voicetype

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

const featureName = "voicetype"

// Feature implements voice type role configuration.
type Feature struct {
	db     database.Client
	cache  cache.Client
	i18n   i18n.I18n
	logger logger.Logger
}

// New creates a new voice type feature.
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

// HandleInteraction handles voice type configuration interactions.
func (f *Feature) HandleInteraction(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	customID := extractCustomID(i)
	guildID := i.GuildID

	// Menu button click - start configuration wizard
	if customID == "menu:voicetype:setup" {
		return f.startWizard(ctx, s, i)
	}

	// Overwrite confirmation
	if customID == "voicetype:confirm_overwrite" {
		return f.showStep1(ctx, s, i)
	}

	if customID == "voicetype:cancel" {
		return f.respondCancelled(ctx, s, i, guildID)
	}

	// Step 1: 高音 role selection
	if strings.HasPrefix(customID, "voicetype:high_role:") {
		return f.handleHighRoleSelection(ctx, s, i)
	}

	// Step 2: 中高音 role selection
	if strings.HasPrefix(customID, "voicetype:mid_high_role:") {
		return f.handleMidHighRoleSelection(ctx, s, i)
	}

	// Step 3: 中音 role selection
	if strings.HasPrefix(customID, "voicetype:mid_role:") {
		return f.handleMidRoleSelection(ctx, s, i)
	}

	// Step 4: 中低音 role selection
	if strings.HasPrefix(customID, "voicetype:mid_low_role:") {
		return f.handleMidLowRoleSelection(ctx, s, i)
	}

	// Step 5: 低音 role selection
	if strings.HasPrefix(customID, "voicetype:low_role:") {
		return f.handleLowRoleSelection(ctx, s, i)
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
		Label:       "🎵 Set Voice Type Roles",
		CustomID:    "menu:voicetype:setup",
		Tier:        3,
		Category:    "admin",
		SubCategory: "configuration",
		AdminOnly:   true,
		IsCategory:  false,
	}
}

// getWizardState retrieves wizard state from cache.
func (f *Feature) getWizardState(ctx context.Context, guildID string) (*WizardState, error) {
	key := fmt.Sprintf("welcomebot:voicetype:wizard:%s", guildID)
	var state WizardState
	if err := f.cache.GetJSON(ctx, key, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// saveWizardState saves wizard state to cache.
func (f *Feature) saveWizardState(ctx context.Context, state *WizardState) error {
	key := fmt.Sprintf("welcomebot:voicetype:wizard:%s", state.GuildID)
	return f.cache.SetJSON(ctx, key, state, 30*time.Minute)
}

// deleteWizardState removes wizard state from cache.
func (f *Feature) deleteWizardState(ctx context.Context, guildID string) error {
	key := fmt.Sprintf("welcomebot:voicetype:wizard:%s", guildID)
	return f.cache.Delete(ctx, key)
}

// startWizard initiates the voice type configuration wizard.
func (f *Feature) startWizard(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	config, err := f.getVoiceTypeConfig(ctx, guildID)
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
func (f *Feature) showOverwriteConfirmation(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, config *VoiceTypeConfig) error {
	guildID := i.GuildID

	desc := f.i18n.T(ctx, guildID, "voicetype.current_config")

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "voicetype.overwrite_title"),
		Description: desc,
		Color:       int(shared.ColorWarning),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    f.i18n.T(ctx, guildID, "voicetype.reconfigure"),
					Style:    discordgo.DangerButton,
					CustomID: "voicetype:confirm_overwrite",
				},
				discordgo.Button{
					Label:    f.i18n.T(ctx, guildID, "common.cancel"),
					Style:    discordgo.SecondaryButton,
					CustomID: "voicetype:cancel",
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep1 shows 高音 role selection.
func (f *Feature) showStep1(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "voicetype.step1_title"),
		Description: f.i18n.T(ctx, guildID, "voicetype.step1_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "voicetype:high_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "voicetype.select_high_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep2 shows 中高音 role selection.
func (f *Feature) showStep2(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "voicetype.step2_title"),
		Description: f.i18n.T(ctx, guildID, "voicetype.step2_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "voicetype:mid_high_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "voicetype.select_mid_high_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep3 shows 中音 role selection.
func (f *Feature) showStep3(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "voicetype.step3_title"),
		Description: f.i18n.T(ctx, guildID, "voicetype.step3_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "voicetype:mid_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "voicetype.select_mid_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep4 shows 中低音 role selection.
func (f *Feature) showStep4(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "voicetype.step4_title"),
		Description: f.i18n.T(ctx, guildID, "voicetype.step4_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "voicetype:mid_low_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "voicetype.select_mid_low_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep5 shows 低音 role selection.
func (f *Feature) showStep5(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "voicetype.step5_title"),
		Description: f.i18n.T(ctx, guildID, "voicetype.step5_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "voicetype:low_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "voicetype.select_low_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// handleHighRoleSelection processes 高音 role selection.
func (f *Feature) handleHighRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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
	state.HighRoleID = roleID
	state.CurrentStep = 2
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep2(ctx, s, i)
}

// handleMidHighRoleSelection processes 中高音 role selection.
func (f *Feature) handleMidHighRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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
	state.MidHighRoleID = roleID
	state.CurrentStep = 3
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep3(ctx, s, i)
}

// handleMidRoleSelection processes 中音 role selection.
func (f *Feature) handleMidRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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
	state.MidRoleID = roleID
	state.CurrentStep = 4
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep4(ctx, s, i)
}

// handleMidLowRoleSelection processes 中低音 role selection.
func (f *Feature) handleMidLowRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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
	state.MidLowRoleID = roleID
	state.CurrentStep = 5
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep5(ctx, s, i)
}

// handleLowRoleSelection processes 低音 role selection and completes wizard.
func (f *Feature) handleLowRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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
	state.LowRoleID = roleID

	// Convert wizard state to config and save
	config := &VoiceTypeConfig{
		GuildID:       guildID,
		HighRoleID:    state.HighRoleID,
		MidHighRoleID: state.MidHighRoleID,
		MidRoleID:     state.MidRoleID,
		MidLowRoleID:  state.MidLowRoleID,
		LowRoleID:     state.LowRoleID,
	}

	if err := f.saveVoiceTypeConfig(ctx, config); err != nil {
		return f.respondError(ctx, s, i, guildID, err)
	}

	// Delete wizard state
	if err := f.deleteWizardState(ctx, guildID); err != nil {
		f.logger.Error("failed to delete wizard state", "error", err)
	}

	return f.respondSuccess(ctx, s, i, guildID)
}

// saveVoiceTypeConfig saves voice type configuration to database and cache.
func (f *Feature) saveVoiceTypeConfig(ctx context.Context, config *VoiceTypeConfig) error {
	query := `
		INSERT INTO guild_voice_type_config (
			guild_id, high_role_id, mid_high_role_id,
			mid_role_id, mid_low_role_id, low_role_id, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (guild_id)
		DO UPDATE SET 
			high_role_id = $2,
			mid_high_role_id = $3,
			mid_role_id = $4,
			mid_low_role_id = $5,
			low_role_id = $6,
			updated_at = NOW()
	`

	_, err := f.db.Exec(ctx, query,
		config.GuildID,
		config.HighRoleID,
		config.MidHighRoleID,
		config.MidRoleID,
		config.MidLowRoleID,
		config.LowRoleID,
	)
	if err != nil {
		return fmt.Errorf("save to database: %w", err)
	}

	config.UpdatedAt = time.Now()

	// Cache configuration
	cacheKey := cacheKeyPrefix + config.GuildID
	if err := f.cache.SetJSON(ctx, cacheKey, config, 0); err != nil {
		f.logger.Warn("failed to cache voice type config", "error", err)
	}

	f.logger.Info("voice type config saved", "guild_id", config.GuildID)

	return nil
}

// getVoiceTypeConfig retrieves voice type configuration.
func (f *Feature) getVoiceTypeConfig(ctx context.Context, guildID string) (*VoiceTypeConfig, error) {
	cacheKey := cacheKeyPrefix + guildID

	var config VoiceTypeConfig
	if err := f.cache.GetJSON(ctx, cacheKey, &config); err == nil {
		return &config, nil
	}

	query := `
		SELECT guild_id, high_role_id, mid_high_role_id,
		       mid_role_id, mid_low_role_id, low_role_id,
		       created_at, updated_at
		FROM guild_voice_type_config 
		WHERE guild_id = $1
	`
	row := f.db.QueryRow(ctx, query, guildID)

	var high, midHigh, mid, midLow, low *string
	err := row.Scan(&config.GuildID,
		&high, &midHigh, &mid, &midLow, &low,
		&config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if high != nil {
		config.HighRoleID = *high
	}
	if midHigh != nil {
		config.MidHighRoleID = *midHigh
	}
	if mid != nil {
		config.MidRoleID = *mid
	}
	if midLow != nil {
		config.MidLowRoleID = *midLow
	}
	if low != nil {
		config.LowRoleID = *low
	}

	f.cache.SetJSON(ctx, cacheKey, &config, 0)

	return &config, nil
}

// respondSuccess sends success message.
func (f *Feature) respondSuccess(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) error {
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "common.success"),
		Description: f.i18n.T(ctx, guildID, "voicetype.success"),
		Color:       int(shared.ColorSuccess),
	}

	return respond(s, i, embed, []discordgo.MessageComponent{})
}

// respondError sends error message.
func (f *Feature) respondError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID string, err error) error {
	f.logger.Error("voice type configuration error", "error", err)

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "common.error"),
		Description: f.i18n.T(ctx, guildID, "voicetype.error_save"),
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
		Description: f.i18n.T(ctx, guildID, "voicetype.cancelled"),
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

