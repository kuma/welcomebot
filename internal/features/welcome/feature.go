package welcome

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
	"welcomebot/internal/core/queue"
	"welcomebot/internal/shared"

	"github.com/bwmarrin/discordgo"
)

const featureName = "welcome"

// Feature implements welcome onboarding configuration.
type Feature struct {
	db      database.Client
	cache   cache.Client
	queue   queue.Client
	i18n    i18n.I18n
	logger  logger.Logger
	session *discordgo.Session
}

// New creates a new welcome feature.
func New(deps Dependencies) (*Feature, error) {
	if err := deps.Validate(); err != nil {
		return nil, fmt.Errorf("validate dependencies: %w", err)
	}

	return &Feature{
		db:      deps.DB,
		cache:   deps.Cache,
		queue:   deps.Queue,
		i18n:    deps.I18n,
		logger:  deps.Logger,
		session: deps.Session,
	}, nil
}

// Name returns the feature name.
func (f *Feature) Name() string {
	return featureName
}

// HandleInteraction handles welcome configuration interactions.
func (f *Feature) HandleInteraction(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	customID := extractCustomID(i)
	guildID := i.GuildID

	// Menu button click - start configuration wizard
	if customID == "menu:welcome:setup" {
		return f.startWizard(ctx, s, i)
	}

	// Welcome button click - start onboarding
	if customID == "welcome:start_onboarding" {
		return f.handleOnboardingStart(ctx, s, i)
	}

	// Overwrite confirmation
	if customID == "welcome:confirm_overwrite" {
		return f.showStep1(ctx, s, i)
	}

	if customID == "welcome:cancel" {
		return f.respondCancelled(ctx, s, i, guildID)
	}

	// Step 1: Welcome channel selection
	if strings.HasPrefix(customID, "welcome:channel:") {
		return f.handleChannelSelection(ctx, s, i)
	}

	// Step 2: VC category selection
	if strings.HasPrefix(customID, "welcome:category:") {
		return f.handleCategorySelection(ctx, s, i)
	}

	// Step 3: Entrance role selection
	if strings.HasPrefix(customID, "welcome:entrance_role:") {
		return f.handleEntranceRoleSelection(ctx, s, i)
	}

	// Step 4: Nyukai role selection
	if strings.HasPrefix(customID, "welcome:nyukai_role:") {
		return f.handleNyukaiRoleSelection(ctx, s, i)
	}

	// Step 5: Setsumeikai 1 role selection
	if strings.HasPrefix(customID, "welcome:setsumeikai1_role:") {
		return f.handleSetsumeikai1RoleSelection(ctx, s, i)
	}

	// Step 6: Setsumeikai 2 role selection
	if strings.HasPrefix(customID, "welcome:setsumeikai2_role:") {
		return f.handleSetsumeikai2RoleSelection(ctx, s, i)
	}

	// Step 7: Setsumeikai 3 role selection
	if strings.HasPrefix(customID, "welcome:setsumeikai3_role:") {
		return f.handleSetsumeikai3RoleSelection(ctx, s, i)
	}

	// Step 8: Member role selection
	if strings.HasPrefix(customID, "welcome:member_role:") {
		return f.handleMemberRoleSelection(ctx, s, i)
	}

	// Step 9: Visitor role selection
	if strings.HasPrefix(customID, "welcome:visitor_role:") {
		return f.handleVisitorRoleSelection(ctx, s, i)
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
		Label:       "👋 Setup Welcome Onboarding",
		CustomID:    "menu:welcome:setup",
		Tier:        3,
		Category:    "admin",
		SubCategory: "configuration",
		AdminOnly:   true,
		IsCategory:  false,
	}
}

// getWizardState retrieves wizard state from cache.
func (f *Feature) getWizardState(ctx context.Context, guildID string) (*WizardState, error) {
	key := fmt.Sprintf("welcomebot:wizard:%s", guildID)
	var state WizardState
	if err := f.cache.GetJSON(ctx, key, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// saveWizardState saves wizard state to cache.
func (f *Feature) saveWizardState(ctx context.Context, state *WizardState) error {
	key := fmt.Sprintf("welcomebot:wizard:%s", state.GuildID)
	return f.cache.SetJSON(ctx, key, state, 30*time.Minute)
}

// deleteWizardState removes wizard state from cache.
func (f *Feature) deleteWizardState(ctx context.Context, guildID string) error {
	key := fmt.Sprintf("welcomebot:wizard:%s", guildID)
	return f.cache.Delete(ctx, key)
}

// startWizard initiates the welcome configuration wizard.
func (f *Feature) startWizard(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	config, err := f.getWelcomeConfig(ctx, guildID)
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
func (f *Feature) showOverwriteConfirmation(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, config *WelcomeConfig) error {
	guildID := i.GuildID

	channel := fmt.Sprintf("<#%s>", config.WelcomeChannelID)
	category := fmt.Sprintf("<#%s>", config.VCCategoryID)

	desc := f.i18n.TWithArgs(ctx, guildID, "welcome.current_config",
		map[string]string{
			"channel":  channel,
			"category": category,
		})

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "welcome.overwrite_title"),
		Description: desc,
		Color:       int(shared.ColorWarning),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    f.i18n.T(ctx, guildID, "welcome.reconfigure"),
					Style:    discordgo.DangerButton,
					CustomID: "welcome:confirm_overwrite",
				},
				discordgo.Button{
					Label:    f.i18n.T(ctx, guildID, "common.cancel"),
					Style:    discordgo.SecondaryButton,
					CustomID: "welcome:cancel",
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep1 shows welcome channel selection.
func (f *Feature) showStep1(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "welcome.step1_title"),
		Description: f.i18n.T(ctx, guildID, "welcome.step1_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.ChannelSelectMenu,
					CustomID:    "welcome:channel:select",
					Placeholder: f.i18n.T(ctx, guildID, "welcome.select_channel"),
					ChannelTypes: []discordgo.ChannelType{
						discordgo.ChannelTypeGuildText,
					},
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep2 shows VC category selection.
func (f *Feature) showStep2(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "welcome.step2_title"),
		Description: f.i18n.T(ctx, guildID, "welcome.step2_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.ChannelSelectMenu,
					CustomID:    "welcome:category:select",
					Placeholder: f.i18n.T(ctx, guildID, "welcome.select_category"),
					ChannelTypes: []discordgo.ChannelType{
						discordgo.ChannelTypeGuildCategory,
					},
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// handleChannelSelection processes welcome channel selection.
func (f *Feature) handleChannelSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return fmt.Errorf("no channel selected")
	}

	channelID := values[0]

	f.logger.Info("welcome channel selected",
		"guild_id", guildID,
		"channel_id", channelID,
	)

	// Update wizard state
	state, err := f.getWizardState(ctx, guildID)
	if err != nil {
		state = &WizardState{GuildID: guildID}
	}
	state.WelcomeChannelID = channelID
	state.CurrentStep = 2
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep2(ctx, s, i)
}

// handleCategorySelection processes VC category selection.
func (f *Feature) handleCategorySelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	values := i.MessageComponentData().Values

	if len(values) == 0 {
		return fmt.Errorf("no category selected")
	}

	categoryID := values[0]

	f.logger.Info("VC category selected",
		"guild_id", guildID,
		"category_id", categoryID,
	)

	// Update wizard state
	state, err := f.getWizardState(ctx, guildID)
	if err != nil {
		return fmt.Errorf("get wizard state: %w", err)
	}
	state.VCCategoryID = categoryID
	state.CurrentStep = 3
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep3(ctx, s, i)
}

// saveWelcomeConfig saves welcome configuration to database and cache.
func (f *Feature) saveWelcomeConfig(ctx context.Context, config *WelcomeConfig) error {
	query := `
		INSERT INTO guild_welcome_config (
			guild_id, welcome_channel_id, vc_category_id,
			entrance_role_id, nyukai_role_id,
			setsumeikai_1_role_id, setsumeikai_2_role_id, setsumeikai_3_role_id,
			member_role_id, visitor_role_id, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (guild_id)
		DO UPDATE SET 
			welcome_channel_id = $2,
			vc_category_id = $3,
			entrance_role_id = $4,
			nyukai_role_id = $5,
			setsumeikai_1_role_id = $6,
			setsumeikai_2_role_id = $7,
			setsumeikai_3_role_id = $8,
			member_role_id = $9,
			visitor_role_id = $10,
			updated_at = NOW()
	`

	_, err := f.db.Exec(ctx, query,
		config.GuildID,
		config.WelcomeChannelID,
		config.VCCategoryID,
		config.EntranceRoleID,
		config.NyukaiRoleID,
		config.Setsumeikai1RoleID,
		config.Setsumeikai2RoleID,
		config.Setsumeikai3RoleID,
		config.MemberRoleID,
		config.VisitorRoleID,
	)
	if err != nil {
		return fmt.Errorf("save to database: %w", err)
	}

	config.UpdatedAt = time.Now()

	// Cache configuration
	cacheKey := cacheKeyPrefix + config.GuildID
	if err := f.cache.SetJSON(ctx, cacheKey, config, 0); err != nil {
		f.logger.Warn("failed to cache welcome config", "error", err)
	}

	f.logger.Info("welcome config saved",
		"guild_id", config.GuildID,
		"channel_id", config.WelcomeChannelID,
		"category_id", config.VCCategoryID,
	)

	return nil
}

// getWelcomeConfig retrieves welcome configuration.
func (f *Feature) getWelcomeConfig(ctx context.Context, guildID string) (*WelcomeConfig, error) {
	cacheKey := cacheKeyPrefix + guildID

	var config WelcomeConfig
	if err := f.cache.GetJSON(ctx, cacheKey, &config); err == nil {
		return &config, nil
	}

	query := `
		SELECT guild_id, welcome_channel_id, vc_category_id, button_message_id, 
		       in_progress_role_id, completed_role_id,
		       entrance_role_id, nyukai_role_id,
		       setsumeikai_1_role_id, setsumeikai_2_role_id, setsumeikai_3_role_id,
		       member_role_id, visitor_role_id, created_at, updated_at
		FROM guild_welcome_config 
		WHERE guild_id = $1
	`
	row := f.db.QueryRow(ctx, query, guildID)

	var inProgressRole, completedRole, buttonMsg *string
	var entranceRole, nyukaiRole, setsumeikai1Role, setsumeikai2Role, setsumeikai3Role, memberRole, visitorRole *string
	err := row.Scan(&config.GuildID, &config.WelcomeChannelID, &config.VCCategoryID,
		&buttonMsg, &inProgressRole, &completedRole,
		&entranceRole, &nyukaiRole,
		&setsumeikai1Role, &setsumeikai2Role, &setsumeikai3Role,
		&memberRole, &visitorRole, &config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if buttonMsg != nil {
		config.ButtonMessageID = *buttonMsg
	}
	if inProgressRole != nil {
		config.InProgressRoleID = *inProgressRole
	}
	if completedRole != nil {
		config.CompletedRoleID = *completedRole
	}
	if entranceRole != nil {
		config.EntranceRoleID = *entranceRole
	}
	if nyukaiRole != nil {
		config.NyukaiRoleID = *nyukaiRole
	}
	if setsumeikai1Role != nil {
		config.Setsumeikai1RoleID = *setsumeikai1Role
	}
	if setsumeikai2Role != nil {
		config.Setsumeikai2RoleID = *setsumeikai2Role
	}
	if setsumeikai3Role != nil {
		config.Setsumeikai3RoleID = *setsumeikai3Role
	}
	if memberRole != nil {
		config.MemberRoleID = *memberRole
	}
	if visitorRole != nil {
		config.VisitorRoleID = *visitorRole
	}

	f.cache.SetJSON(ctx, cacheKey, &config, 0)

	return &config, nil
}

// postWelcomeButton posts the welcome button in the configured channel.
func (f *Feature) postWelcomeButton(ctx context.Context, guildID, channelID string) error {
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "welcome.button_title"),
		Description: f.i18n.T(ctx, guildID, "welcome.button_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    f.i18n.T(ctx, guildID, "welcome.start_button"),
					Style:    discordgo.PrimaryButton,
					CustomID: "welcome:start_onboarding",
					Emoji: &discordgo.ComponentEmoji{
						Name: "👋",
					},
				},
			},
		},
	}

	msg, err := f.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	// Update button message ID in database
	query := `UPDATE guild_welcome_config SET button_message_id = $1 WHERE guild_id = $2`
	_, err = f.db.Exec(ctx, query, msg.ID, guildID)
	if err != nil {
		f.logger.Warn("failed to update button message ID", "error", err)
	}

	f.logger.Info("welcome button posted",
		"guild_id", guildID,
		"channel_id", channelID,
		"message_id", msg.ID,
	)

	return nil
}

// handleOnboardingStart handles when a user clicks the start onboarding button.
func (f *Feature) handleOnboardingStart(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	userID := i.Member.User.ID

	// Get config
	config, err := f.getWelcomeConfig(ctx, guildID)
	if err != nil {
		return f.respondErrorMessage(ctx, s, i, guildID, "welcome.config_not_found")
	}

	// Check if user already has active session
	sessionKey := fmt.Sprintf("%s%s:%s", sessionKeyPrefix, guildID, userID)
	var existingSession OnboardingSession
	if err := f.cache.GetJSON(ctx, sessionKey, &existingSession); err == nil {
		return f.respondErrorMessage(ctx, s, i, guildID, "welcome.session_already_active")
	}

	// Find available slave
	slaveID, err := f.findAvailableSlave(ctx)
	if err != nil || slaveID == "" {
		return f.respondErrorMessage(ctx, s, i, guildID, "welcome.no_slaves_available")
	}

	// Get age range, voice type, and other roles configs
	ageRangeConfig, _ := f.getAgeRangeConfig(ctx, guildID)
	voiceTypeConfig, _ := f.getVoiceTypeConfig(ctx, guildID)
	otherRolesConfig, _ := f.getOtherRolesConfig(ctx, guildID)

	// Create onboarding task with all role configurations
	payload := map[string]interface{}{
		"user_id":            userID,
		"category_id":        config.VCCategoryID,
		"slave_id":           slaveID,
		"in_progress_role":   config.InProgressRoleID,
		"completed_role":     config.CompletedRoleID,
		"entrance_role":      config.EntranceRoleID,
		"nyukai_role":        config.NyukaiRoleID,
		"setsumeikai_1_role": config.Setsumeikai1RoleID,
		"setsumeikai_2_role": config.Setsumeikai2RoleID,
		"setsumeikai_3_role": config.Setsumeikai3RoleID,
		"member_role":        config.MemberRoleID,
	}

	// Add age range roles if configured
	if ageRangeConfig != nil {
		payload["age_20_early_role"] = ageRangeConfig.Age20EarlyRoleID
		payload["age_20_late_role"] = ageRangeConfig.Age20LateRoleID
		payload["age_30_early_role"] = ageRangeConfig.Age30EarlyRoleID
		payload["age_30_late_role"] = ageRangeConfig.Age30LateRoleID
		payload["age_40_early_role"] = ageRangeConfig.Age40EarlyRoleID
		payload["age_40_late_role"] = ageRangeConfig.Age40LateRoleID
	}

	// Add voice type roles if configured
	if voiceTypeConfig != nil {
		payload["high_voice_role"] = voiceTypeConfig.HighRoleID
		payload["mid_high_voice_role"] = voiceTypeConfig.MidHighRoleID
		payload["mid_voice_role"] = voiceTypeConfig.MidRoleID
		payload["mid_low_voice_role"] = voiceTypeConfig.MidLowRoleID
		payload["low_voice_role"] = voiceTypeConfig.LowRoleID
	}

	// Add other roles if configured
	if otherRolesConfig != nil {
		payload["ero_ok_role"] = otherRolesConfig.EroOkRoleID
		payload["ero_ng_role"] = otherRolesConfig.EroNgRoleID
		payload["neochi_ok_role"] = otherRolesConfig.NeochiOkRoleID
		payload["neochi_ng_role"] = otherRolesConfig.NeochiNgRoleID
		payload["neochi_disconnect_role"] = otherRolesConfig.NeochiDisconnectRoleID
		payload["dm_ok_role"] = otherRolesConfig.DmOkRoleID
		payload["dm_ng_role"] = otherRolesConfig.DmNgRoleID
		payload["friend_ok_role"] = otherRolesConfig.FriendOkRoleID
		payload["friend_ng_role"] = otherRolesConfig.FriendNgRoleID
		payload["bunnyclub_event_role"] = otherRolesConfig.BunnyclubEventRoleID
		payload["user_event_role"] = otherRolesConfig.UserEventRoleID
	}

	task := queue.Task{
		ID:        fmt.Sprintf("onboard-%s-%s-%d", guildID, userID, time.Now().Unix()),
		Type:      "onboarding_start",
		GuildID:   guildID,
		Payload:   payload,
		CreatedAt: time.Now(),
	}

	// Enqueue task
	if err := f.queue.Enqueue(ctx, task); err != nil {
		f.logger.Error("failed to enqueue onboarding task", "error", err)
		return f.respondErrorMessage(ctx, s, i, guildID, "welcome.enqueue_failed")
	}

	// Mark slave as busy
	if err := f.setSlaveStatus(ctx, slaveID, SlaveStatusBusy); err != nil {
		f.logger.Warn("failed to mark slave as busy", "error", err)
	}

	// Create session record
	session := OnboardingSession{
		GuildID:   guildID,
		UserID:    userID,
		SlaveID:   slaveID,
		StartedAt: time.Now(),
	}
	if err := f.cache.SetJSON(ctx, sessionKey, session, 15*time.Minute); err != nil {
		f.logger.Warn("failed to cache session", "error", err)
	}

	f.logger.Info("onboarding started",
		"guild_id", guildID,
		"user_id", userID,
		"slave_id", slaveID,
	)

	// Respond to user
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "welcome.starting_title"),
		Description: f.i18n.T(ctx, guildID, "welcome.starting_description"),
		Color:       int(shared.ColorSuccess),
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

// findAvailableSlave finds an available slave bot.
func (f *Feature) findAvailableSlave(ctx context.Context) (string, error) {
	for _, slaveID := range SlaveIDs {
		status, err := f.getSlaveStatus(ctx, slaveID)
		if err != nil {
			continue
		}
		if status == SlaveStatusAvailable {
			return slaveID, nil
		}
	}
	return "", fmt.Errorf("no available slaves")
}

// getSlaveStatus gets the status of a slave bot.
func (f *Feature) getSlaveStatus(ctx context.Context, slaveID string) (SlaveStatus, error) {
	key := slaveStatusKey + slaveID
	status, err := f.cache.Get(ctx, key)
	if err != nil {
		// Default to offline if not found
		return SlaveStatusOffline, err
	}
	return SlaveStatus(status), nil
}

// setSlaveStatus sets the status of a slave bot.
func (f *Feature) setSlaveStatus(ctx context.Context, slaveID string, status SlaveStatus) error {
	key := slaveStatusKey + slaveID
	return f.cache.Set(ctx, key, string(status), 30*time.Minute)
}

// extractChannelID extracts channel ID from CustomID.
func extractChannelID(customID string) (string, error) {
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
func (f *Feature) respondSuccess(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID, channelID, categoryID string) error {
	desc := f.i18n.TWithArgs(ctx, guildID, "welcome.success",
		map[string]string{
			"channel":  fmt.Sprintf("<#%s>", channelID),
			"category": fmt.Sprintf("<#%s>", categoryID),
		})

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "common.success"),
		Description: desc,
		Color:       int(shared.ColorSuccess),
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

	f.logger.Error("welcome configuration error",
		"guild_id", guildID,
		"error", err,
	)

	return respond(s, i, embed, []discordgo.MessageComponent{})
}

// respondErrorMessage sends error message with specific translation key.
func (f *Feature) respondErrorMessage(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID, messageKey string) error {
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "common.error"),
		Description: f.i18n.T(ctx, guildID, messageKey),
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
		Description: f.i18n.T(ctx, guildID, "welcome.cancelled"),
		Color:       int(shared.ColorInfo),
	}

	return respond(s, i, embed, []discordgo.MessageComponent{})
}

// showStep3 shows Entrance role selection.
func (f *Feature) showStep3(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "welcome.step3_title"),
		Description: f.i18n.T(ctx, guildID, "welcome.step3_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "welcome:entrance_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "welcome.select_entrance_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep4 shows Nyukai role selection.
func (f *Feature) showStep4(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "welcome.step4_title"),
		Description: f.i18n.T(ctx, guildID, "welcome.step4_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "welcome:nyukai_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "welcome.select_nyukai_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep5 shows Setsumeikai 1 role selection.
func (f *Feature) showStep5(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "welcome.step5_title"),
		Description: f.i18n.T(ctx, guildID, "welcome.step5_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "welcome:setsumeikai1_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "welcome.select_setsumeikai1_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep6 shows Setsumeikai 2 role selection.
func (f *Feature) showStep6(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "welcome.step6_title"),
		Description: f.i18n.T(ctx, guildID, "welcome.step6_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "welcome:setsumeikai2_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "welcome.select_setsumeikai2_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep7 shows Setsumeikai 3 role selection.
func (f *Feature) showStep7(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "welcome.step7_title"),
		Description: f.i18n.T(ctx, guildID, "welcome.step7_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "welcome:setsumeikai3_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "welcome.select_setsumeikai3_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep8 shows Member role selection.
func (f *Feature) showStep8(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "welcome.step8_title"),
		Description: f.i18n.T(ctx, guildID, "welcome.step8_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "welcome:member_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "welcome.select_member_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// showStep9 shows Visitor role selection.
func (f *Feature) showStep9(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "welcome.step9_title"),
		Description: f.i18n.T(ctx, guildID, "welcome.step9_description"),
		Color:       int(shared.ColorInfo),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "welcome:visitor_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "welcome.select_visitor_role"),
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

// handleEntranceRoleSelection processes Entrance role selection.
func (f *Feature) handleEntranceRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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
	state.EntranceRoleID = roleID
	state.CurrentStep = 4
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep4(ctx, s, i)
}

// handleNyukaiRoleSelection processes Nyukai role selection.
func (f *Feature) handleNyukaiRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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
	state.NyukaiRoleID = roleID
	state.CurrentStep = 5
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep5(ctx, s, i)
}

// handleSetsumeikai1RoleSelection processes Setsumeikai 1 role selection.
func (f *Feature) handleSetsumeikai1RoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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
	state.Setsumeikai1RoleID = roleID
	state.CurrentStep = 6
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep6(ctx, s, i)
}

// handleSetsumeikai2RoleSelection processes Setsumeikai 2 role selection.
func (f *Feature) handleSetsumeikai2RoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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
	state.Setsumeikai2RoleID = roleID
	state.CurrentStep = 7
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep7(ctx, s, i)
}

// handleSetsumeikai3RoleSelection processes Setsumeikai 3 role selection.
func (f *Feature) handleSetsumeikai3RoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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
	state.Setsumeikai3RoleID = roleID
	state.CurrentStep = 8
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep8(ctx, s, i)
}

// handleMemberRoleSelection processes Member role selection.
func (f *Feature) handleMemberRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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
	state.MemberRoleID = roleID
	state.CurrentStep = 9
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep9(ctx, s, i)
}

// handleVisitorRoleSelection processes Visitor role selection and completes wizard.
func (f *Feature) handleVisitorRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
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
	state.VisitorRoleID = roleID

	// Convert wizard state to config and save
	config := &WelcomeConfig{
		GuildID:             guildID,
		WelcomeChannelID:    state.WelcomeChannelID,
		VCCategoryID:        state.VCCategoryID,
		EntranceRoleID:      state.EntranceRoleID,
		NyukaiRoleID:        state.NyukaiRoleID,
		Setsumeikai1RoleID:  state.Setsumeikai1RoleID,
		Setsumeikai2RoleID:  state.Setsumeikai2RoleID,
		Setsumeikai3RoleID:  state.Setsumeikai3RoleID,
		MemberRoleID:        state.MemberRoleID,
		VisitorRoleID:       state.VisitorRoleID,
	}

	if err := f.saveWelcomeConfig(ctx, config); err != nil {
		return f.respondError(ctx, s, i, guildID, err)
	}

	// Post welcome button
	if err := f.postWelcomeButton(ctx, guildID, state.WelcomeChannelID); err != nil {
		f.logger.Error("failed to post welcome button", "error", err)
	}

	// Delete wizard state
	if err := f.deleteWizardState(ctx, guildID); err != nil {
		f.logger.Error("failed to delete wizard state", "error", err)
	}

	return f.respondSuccess(ctx, s, i, guildID, state.WelcomeChannelID, state.VCCategoryID)
}

// getAgeRangeConfig retrieves age range configuration.
func (f *Feature) getAgeRangeConfig(ctx context.Context, guildID string) (*AgeRangeConfig, error) {
	query := `
		SELECT guild_id, age_20_early_role_id, age_20_late_role_id,
		       age_30_early_role_id, age_30_late_role_id,
		       age_40_early_role_id, age_40_late_role_id
		FROM guild_age_range_config 
		WHERE guild_id = $1
	`
	row := f.db.QueryRow(ctx, query, guildID)

	var config AgeRangeConfig
	var age20Early, age20Late, age30Early, age30Late, age40Early, age40Late *string
	err := row.Scan(&config.GuildID, &age20Early, &age20Late,
		&age30Early, &age30Late, &age40Early, &age40Late)
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

	return &config, nil
}

// getVoiceTypeConfig retrieves voice type configuration.
func (f *Feature) getVoiceTypeConfig(ctx context.Context, guildID string) (*VoiceTypeConfig, error) {
	query := `
		SELECT guild_id, high_role_id, mid_high_role_id,
		       mid_role_id, mid_low_role_id, low_role_id
		FROM guild_voice_type_config 
		WHERE guild_id = $1
	`
	row := f.db.QueryRow(ctx, query, guildID)

	var config VoiceTypeConfig
	var high, midHigh, mid, midLow, low *string
	err := row.Scan(&config.GuildID, &high, &midHigh, &mid, &midLow, &low)
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

	return &config, nil
}

// getOtherRolesConfig retrieves other roles configuration.
func (f *Feature) getOtherRolesConfig(ctx context.Context, guildID string) (*OtherRolesConfig, error) {
	query := `
		SELECT guild_id, ero_ok_role_id, ero_ng_role_id,
		       neochi_ok_role_id, neochi_ng_role_id, neochi_disconnect_role_id,
		       dm_ok_role_id, dm_ng_role_id,
		       friend_ok_role_id, friend_ng_role_id,
		       bunnyclub_event_role_id, user_event_role_id
		FROM guild_other_roles_config 
		WHERE guild_id = $1
	`
	row := f.db.QueryRow(ctx, query, guildID)

	var config OtherRolesConfig
	var eroOk, eroNg, neochiOk, neochiNg, neochiDisconnect *string
	var dmOk, dmNg, friendOk, friendNg, bunnyclubEvent, userEvent *string
	err := row.Scan(&config.GuildID, &eroOk, &eroNg,
		&neochiOk, &neochiNg, &neochiDisconnect,
		&dmOk, &dmNg, &friendOk, &friendNg,
		&bunnyclubEvent, &userEvent)
	if err != nil {
		return nil, err
	}

	if eroOk != nil {
		config.EroOkRoleID = *eroOk
	}
	if eroNg != nil {
		config.EroNgRoleID = *eroNg
	}
	if neochiOk != nil {
		config.NeochiOkRoleID = *neochiOk
	}
	if neochiNg != nil {
		config.NeochiNgRoleID = *neochiNg
	}
	if neochiDisconnect != nil {
		config.NeochiDisconnectRoleID = *neochiDisconnect
	}
	if dmOk != nil {
		config.DmOkRoleID = *dmOk
	}
	if dmNg != nil {
		config.DmNgRoleID = *dmNg
	}
	if friendOk != nil {
		config.FriendOkRoleID = *friendOk
	}
	if friendNg != nil {
		config.FriendNgRoleID = *friendNg
	}
	if bunnyclubEvent != nil {
		config.BunnyclubEventRoleID = *bunnyclubEvent
	}
	if userEvent != nil {
		config.UserEventRoleID = *userEvent
	}

	return &config, nil
}

