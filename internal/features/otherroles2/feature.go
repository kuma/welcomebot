package otherroles2

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

const featureName = "otherroles2"

// Feature implements other roles 2 configuration.
type Feature struct {
	db     database.Client
	cache  cache.Client
	i18n   i18n.I18n
	logger logger.Logger
}

// New creates a new other roles 2 feature.
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

// HandleInteraction handles other roles 2 configuration interactions.
func (f *Feature) HandleInteraction(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	customID := extractCustomID(i)
	guildID := i.GuildID

	if customID == "menu:otherroles2:setup" {
		return f.startWizard(ctx, s, i)
	}

	if customID == "otherroles2:confirm_overwrite" {
		return f.showStep1(ctx, s, i)
	}

	if customID == "otherroles2:cancel" {
		return f.respondCancelled(ctx, s, i, guildID)
	}

	if strings.HasPrefix(customID, "otherroles2:dm_ok_role:") {
		return f.handleDMOKRoleSelection(ctx, s, i)
	}

	if strings.HasPrefix(customID, "otherroles2:dm_ng_role:") {
		return f.handleDMNGRoleSelection(ctx, s, i)
	}

	if strings.HasPrefix(customID, "otherroles2:friend_ok_role:") {
		return f.handleFriendOKRoleSelection(ctx, s, i)
	}

	if strings.HasPrefix(customID, "otherroles2:friend_ng_role:") {
		return f.handleFriendNGRoleSelection(ctx, s, i)
	}

	if strings.HasPrefix(customID, "otherroles2:bunnyclub_event_role:") {
		return f.handleBunnyclubEventRoleSelection(ctx, s, i)
	}

	if strings.HasPrefix(customID, "otherroles2:user_event_role:") {
		return f.handleUserEventRoleSelection(ctx, s, i)
	}

	return bot.ErrNotHandled
}

// RegisterCommands returns slash commands for this feature.
func (f *Feature) RegisterCommands() []*discordgo.ApplicationCommand {
	return nil
}

// GetMenuButton returns the menu button for this feature.
func (f *Feature) GetMenuButton() *bot.MenuButton {
	return &bot.MenuButton{
		Label:       "📝 Set Other Roles 2",
		CustomID:    "menu:otherroles2:setup",
		Tier:        3,
		Category:    "admin",
		SubCategory: "configuration",
		AdminOnly:   true,
		IsCategory:  false,
	}
}

func (f *Feature) getWizardState(ctx context.Context, guildID string) (*WizardState, error) {
	key := fmt.Sprintf("welcomebot:otherroles2:wizard:%s", guildID)
	var state WizardState
	if err := f.cache.GetJSON(ctx, key, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (f *Feature) saveWizardState(ctx context.Context, state *WizardState) error {
	key := fmt.Sprintf("welcomebot:otherroles2:wizard:%s", state.GuildID)
	return f.cache.SetJSON(ctx, key, state, 30*time.Minute)
}

func (f *Feature) deleteWizardState(ctx context.Context, guildID string) error {
	key := fmt.Sprintf("welcomebot:otherroles2:wizard:%s", guildID)
	return f.cache.Delete(ctx, key)
}

func (f *Feature) startWizard(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	config, err := f.getOtherRolesConfig(ctx, guildID)
	if err == nil && config != nil && config.DMOKRoleID != "" {
		return f.showOverwriteConfirmation(ctx, s, i, config)
	}

	state := &WizardState{GuildID: guildID, CurrentStep: 1}
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}

	return f.showStep1(ctx, s, i)
}

func (f *Feature) showOverwriteConfirmation(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, config *OtherRolesConfig) error {
	guildID := i.GuildID

	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "otherroles2.overwrite_title"),
		Description: f.i18n.T(ctx, guildID, "otherroles2.current_config"),
		Color:       int(shared.ColorWarning),
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    f.i18n.T(ctx, guildID, "otherroles2.reconfigure"),
					Style:    discordgo.DangerButton,
					CustomID: "otherroles2:confirm_overwrite",
				},
				discordgo.Button{
					Label:    f.i18n.T(ctx, guildID, "common.cancel"),
					Style:    discordgo.SecondaryButton,
					CustomID: "otherroles2:cancel",
				},
			},
		},
	}

	return respond(s, i, embed, components)
}

func (f *Feature) showStep1(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "otherroles2.step1_title"),
		Description: f.i18n.T(ctx, guildID, "otherroles2.step1_description"),
		Color:       int(shared.ColorInfo),
	}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "otherroles2:dm_ok_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "otherroles2.select_dm_ok_role"),
				},
			},
		},
	}
	return respond(s, i, embed, components)
}

func (f *Feature) showStep2(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "otherroles2.step2_title"),
		Description: f.i18n.T(ctx, guildID, "otherroles2.step2_description"),
		Color:       int(shared.ColorInfo),
	}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "otherroles2:dm_ng_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "otherroles2.select_dm_ng_role"),
				},
			},
		},
	}
	return respond(s, i, embed, components)
}

func (f *Feature) showStep3(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "otherroles2.step3_title"),
		Description: f.i18n.T(ctx, guildID, "otherroles2.step3_description"),
		Color:       int(shared.ColorInfo),
	}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "otherroles2:friend_ok_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "otherroles2.select_friend_ok_role"),
				},
			},
		},
	}
	return respond(s, i, embed, components)
}

func (f *Feature) showStep4(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "otherroles2.step4_title"),
		Description: f.i18n.T(ctx, guildID, "otherroles2.step4_description"),
		Color:       int(shared.ColorInfo),
	}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "otherroles2:friend_ng_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "otherroles2.select_friend_ng_role"),
				},
			},
		},
	}
	return respond(s, i, embed, components)
}

func (f *Feature) showStep5(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "otherroles2.step5_title"),
		Description: f.i18n.T(ctx, guildID, "otherroles2.step5_description"),
		Color:       int(shared.ColorInfo),
	}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "otherroles2:bunnyclub_event_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "otherroles2.select_bunnyclub_event_role"),
				},
			},
		},
	}
	return respond(s, i, embed, components)
}

func (f *Feature) showStep6(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "otherroles2.step6_title"),
		Description: f.i18n.T(ctx, guildID, "otherroles2.step6_description"),
		Color:       int(shared.ColorInfo),
	}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.RoleSelectMenu,
					CustomID:    "otherroles2:user_event_role:select",
					Placeholder: f.i18n.T(ctx, guildID, "otherroles2.select_user_event_role"),
				},
			},
		},
	}
	return respond(s, i, embed, components)
}

func (f *Feature) handleDMOKRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return fmt.Errorf("no role selected")
	}
	state, err := f.getWizardState(ctx, guildID)
	if err != nil {
		state = &WizardState{GuildID: guildID}
	}
	state.DMOKRoleID = values[0]
	state.CurrentStep = 2
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}
	return f.showStep2(ctx, s, i)
}

func (f *Feature) handleDMNGRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return fmt.Errorf("no role selected")
	}
	state, err := f.getWizardState(ctx, guildID)
	if err != nil {
		return fmt.Errorf("get wizard state: %w", err)
	}
	state.DMNGRoleID = values[0]
	state.CurrentStep = 3
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}
	return f.showStep3(ctx, s, i)
}

func (f *Feature) handleFriendOKRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return fmt.Errorf("no role selected")
	}
	state, err := f.getWizardState(ctx, guildID)
	if err != nil {
		return fmt.Errorf("get wizard state: %w", err)
	}
	state.FriendOKRoleID = values[0]
	state.CurrentStep = 4
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}
	return f.showStep4(ctx, s, i)
}

func (f *Feature) handleFriendNGRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return fmt.Errorf("no role selected")
	}
	state, err := f.getWizardState(ctx, guildID)
	if err != nil {
		return fmt.Errorf("get wizard state: %w", err)
	}
	state.FriendNGRoleID = values[0]
	state.CurrentStep = 5
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}
	return f.showStep5(ctx, s, i)
}

func (f *Feature) handleBunnyclubEventRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return fmt.Errorf("no role selected")
	}
	state, err := f.getWizardState(ctx, guildID)
	if err != nil {
		return fmt.Errorf("get wizard state: %w", err)
	}
	state.BunnyclubEventRoleID = values[0]
	state.CurrentStep = 6
	if err := f.saveWizardState(ctx, state); err != nil {
		f.logger.Error("failed to save wizard state", "error", err)
	}
	return f.showStep6(ctx, s, i)
}

func (f *Feature) handleUserEventRoleSelection(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return fmt.Errorf("no role selected")
	}
	state, err := f.getWizardState(ctx, guildID)
	if err != nil {
		return fmt.Errorf("get wizard state: %w", err)
	}
	state.UserEventRoleID = values[0]

	// Get existing config to preserve Other Roles 1 values
	existing, _ := f.getOtherRolesConfig(ctx, guildID)
	
	config := &OtherRolesConfig{
		GuildID:              guildID,
		DMOKRoleID:           state.DMOKRoleID,
		DMNGRoleID:           state.DMNGRoleID,
		FriendOKRoleID:       state.FriendOKRoleID,
		FriendNGRoleID:       state.FriendNGRoleID,
		BunnyclubEventRoleID: state.BunnyclubEventRoleID,
		UserEventRoleID:      state.UserEventRoleID,
	}

	// Preserve Other Roles 1 values if they exist
	if existing != nil {
		config.EroOKRoleID = existing.EroOKRoleID
		config.EroNGRoleID = existing.EroNGRoleID
		config.NeochiOKRoleID = existing.NeochiOKRoleID
		config.NeochiNGRoleID = existing.NeochiNGRoleID
		config.NeochiDisconnectRoleID = existing.NeochiDisconnectRoleID
	}

	if err := f.saveOtherRolesConfig(ctx, config); err != nil {
		return f.respondError(ctx, s, i, guildID, err)
	}

	if err := f.deleteWizardState(ctx, guildID); err != nil {
		f.logger.Error("failed to delete wizard state", "error", err)
	}

	return f.respondSuccess(ctx, s, i, guildID)
}

func (f *Feature) saveOtherRolesConfig(ctx context.Context, config *OtherRolesConfig) error {
	query := `
		INSERT INTO guild_other_roles_config (
			guild_id, ero_ok_role_id, ero_ng_role_id,
			neochi_ok_role_id, neochi_ng_role_id, neochi_disconnect_role_id,
			dm_ok_role_id, dm_ng_role_id, friend_ok_role_id, friend_ng_role_id,
			bunnyclub_event_role_id, user_event_role_id, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		ON CONFLICT (guild_id)
		DO UPDATE SET 
			ero_ok_role_id = $2,
			ero_ng_role_id = $3,
			neochi_ok_role_id = $4,
			neochi_ng_role_id = $5,
			neochi_disconnect_role_id = $6,
			dm_ok_role_id = $7,
			dm_ng_role_id = $8,
			friend_ok_role_id = $9,
			friend_ng_role_id = $10,
			bunnyclub_event_role_id = $11,
			user_event_role_id = $12,
			updated_at = NOW()
	`

	_, err := f.db.Exec(ctx, query,
		config.GuildID,
		config.EroOKRoleID, config.EroNGRoleID,
		config.NeochiOKRoleID, config.NeochiNGRoleID, config.NeochiDisconnectRoleID,
		config.DMOKRoleID, config.DMNGRoleID,
		config.FriendOKRoleID, config.FriendNGRoleID,
		config.BunnyclubEventRoleID, config.UserEventRoleID,
	)
	if err != nil {
		return fmt.Errorf("save to database: %w", err)
	}

	config.UpdatedAt = time.Now()
	cacheKey := cacheKeyPrefix + config.GuildID
	if err := f.cache.SetJSON(ctx, cacheKey, config, 0); err != nil {
		f.logger.Warn("failed to cache other roles config", "error", err)
	}

	f.logger.Info("other roles 2 config saved", "guild_id", config.GuildID)
	return nil
}

func (f *Feature) getOtherRolesConfig(ctx context.Context, guildID string) (*OtherRolesConfig, error) {
	cacheKey := cacheKeyPrefix + guildID

	var config OtherRolesConfig
	if err := f.cache.GetJSON(ctx, cacheKey, &config); err == nil {
		return &config, nil
	}

	query := `
		SELECT guild_id, ero_ok_role_id, ero_ng_role_id,
		       neochi_ok_role_id, neochi_ng_role_id, neochi_disconnect_role_id,
		       dm_ok_role_id, dm_ng_role_id, friend_ok_role_id, friend_ng_role_id,
		       bunnyclub_event_role_id, user_event_role_id,
		       created_at, updated_at
		FROM guild_other_roles_config 
		WHERE guild_id = $1
	`
	row := f.db.QueryRow(ctx, query, guildID)

	var eroOK, eroNG, neochiOK, neochiNG, neochiDC *string
	var dmOK, dmNG, friendOK, friendNG, bunnyEvent, userEvent *string
	err := row.Scan(&config.GuildID,
		&eroOK, &eroNG, &neochiOK, &neochiNG, &neochiDC,
		&dmOK, &dmNG, &friendOK, &friendNG, &bunnyEvent, &userEvent,
		&config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if eroOK != nil {
		config.EroOKRoleID = *eroOK
	}
	if eroNG != nil {
		config.EroNGRoleID = *eroNG
	}
	if neochiOK != nil {
		config.NeochiOKRoleID = *neochiOK
	}
	if neochiNG != nil {
		config.NeochiNGRoleID = *neochiNG
	}
	if neochiDC != nil {
		config.NeochiDisconnectRoleID = *neochiDC
	}
	if dmOK != nil {
		config.DMOKRoleID = *dmOK
	}
	if dmNG != nil {
		config.DMNGRoleID = *dmNG
	}
	if friendOK != nil {
		config.FriendOKRoleID = *friendOK
	}
	if friendNG != nil {
		config.FriendNGRoleID = *friendNG
	}
	if bunnyEvent != nil {
		config.BunnyclubEventRoleID = *bunnyEvent
	}
	if userEvent != nil {
		config.UserEventRoleID = *userEvent
	}

	f.cache.SetJSON(ctx, cacheKey, &config, 0)
	return &config, nil
}

func (f *Feature) respondSuccess(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) error {
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "common.success"),
		Description: f.i18n.T(ctx, guildID, "otherroles2.success"),
		Color:       int(shared.ColorSuccess),
	}
	return respond(s, i, embed, []discordgo.MessageComponent{})
}

func (f *Feature) respondError(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID string, err error) error {
	f.logger.Error("other roles 2 configuration error", "error", err)
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "common.error"),
		Description: f.i18n.T(ctx, guildID, "otherroles2.error_save"),
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

func (f *Feature) respondCancelled(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) error {
	embed := &discordgo.MessageEmbed{
		Title:       f.i18n.T(ctx, guildID, "common.cancelled"),
		Description: f.i18n.T(ctx, guildID, "otherroles2.cancelled"),
		Color:       int(shared.ColorInfo),
	}
	return respond(s, i, embed, []discordgo.MessageComponent{})
}

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

func extractCustomID(i *discordgo.InteractionCreate) string {
	if i.Type == discordgo.InteractionMessageComponent {
		return i.MessageComponentData().CustomID
	}
	return ""
}
