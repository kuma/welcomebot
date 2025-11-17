package menu

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

const featureName = "menu"

// Feature implements the menu system.
type Feature struct {
	registry FeatureRegistry
	init     InitChecker
	i18n     i18n.I18n
	logger   logger.Logger
}

// New creates a new menu feature.
func New(deps Dependencies) (*Feature, error) {
	if err := deps.Validate(); err != nil {
		return nil, fmt.Errorf("validate dependencies: %w", err)
	}

	return &Feature{
		registry: deps.Registry,
		init:     deps.Init,
		i18n:     deps.I18n,
		logger:   deps.Logger,
	}, nil
}

// Name returns the feature name.
func (f *Feature) Name() string {
	return featureName
}

// HandleInteraction handles menu command and navigation interactions.
func (f *Feature) HandleInteraction(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	if guildID == "" {
		return f.respondError(s, i, "This command must be used in a server")
	}

	// Handle /welcomebot command
	if i.Type == discordgo.InteractionApplicationCommand {
		data := i.ApplicationCommandData()
		if data.Name != "welcomebot" {
			return bot.ErrNotHandled
		}
		return f.showMainMenu(ctx, s, i)
	}

	// Handle button clicks
	if i.Type == discordgo.InteractionMessageComponent {
		customID := i.MessageComponentData().CustomID
		
		// Category navigation
		if strings.HasPrefix(customID, "menu:category:") {
			return f.showCategorySubMenu(ctx, s, i)
		}
		
		// Sub-category navigation
		if strings.HasPrefix(customID, "menu:subcategory:") {
			return f.showSubCategoryMenu(ctx, s, i)
		}
		
		// Back navigation
		if strings.HasPrefix(customID, "menu:back:") {
			return f.handleBackNavigation(ctx, s, i)
		}
	}

	return bot.ErrNotHandled
}

// RegisterCommands returns slash commands for this feature.
func (f *Feature) RegisterCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "welcomebot",
			Description: "Show welcome bot features menu",
		},
	}
}

// GetMenuButton returns the menu button for this feature.
func (f *Feature) GetMenuButton() *bot.MenuButton {
	return nil // Menu doesn't appear in itself
}

// showMainMenu shows Tier 1 main menu or delegates to init.
func (f *Feature) showMainMenu(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID

	// Check if guild is initialized
	isReady, missing := f.init.CheckRequired(ctx, guildID)
	if !isReady {
		return f.init.StartInitWizard(ctx, s, i, missing)
	}

	// Show main menu (Tier 1)
	return f.displayMainMenu(ctx, s, i)
}

// showCategorySubMenu shows Tier 2 sub-menu for a category.
func (f *Feature) showCategorySubMenu(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	customID := i.MessageComponentData().CustomID
	// Parse: "menu:category:admin" → category = "admin"
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		return fmt.Errorf("invalid category customID")
	}
	
	category := parts[2]
	return f.displayCategoryMenu(ctx, s, i, category)
}

// showSubCategoryMenu shows Tier 3 feature list for a sub-category.
func (f *Feature) showSubCategoryMenu(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	customID := i.MessageComponentData().CustomID
	// Parse: "menu:subcategory:admin:configuration" → category="admin", sub="configuration"
	parts := strings.Split(customID, ":")
	if len(parts) < 4 {
		return fmt.Errorf("invalid subcategory customID")
	}
	
	category := parts[2]
	subCategory := parts[3]
	return f.displayFeatureList(ctx, s, i, category, subCategory)
}

// handleBackNavigation handles back button clicks.
func (f *Feature) handleBackNavigation(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	customID := i.MessageComponentData().CustomID
	// Parse: "menu:back:main" or "menu:back:admin"
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		return fmt.Errorf("invalid back customID")
	}
	
	target := parts[2]
	
	if target == "main" {
		return f.displayMainMenu(ctx, s, i)
	}
	
	// Back to category (e.g., "menu:back:admin")
	return f.displayCategoryMenu(ctx, s, i, target)
}

// displayMainMenu shows Tier 1 main categories.
func (f *Feature) displayMainMenu(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	guildID := i.GuildID
	userID := i.Member.User.ID
	isAdmin := f.checkAdminPermission(s, guildID, userID)

	embed := &discordgo.MessageEmbed{
		Title:       "🤖 " + f.i18n.T(ctx, guildID, "menu.title"),
		Description: f.i18n.T(ctx, guildID, "menu.main_description"),
		Color:       int(shared.ColorInfo),
	}

	components := f.buildMainMenuButtons(ctx, guildID, isAdmin)

	return f.respond(s, i, embed, components)
}

// displayCategoryMenu shows Tier 2 sub-categories for a main category.
func (f *Feature) displayCategoryMenu(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, category string) error {
	guildID := i.GuildID
	userID := i.Member.User.ID
	isAdmin := f.checkAdminPermission(s, guildID, userID)

	embed := &discordgo.MessageEmbed{
		Title:       f.getCategoryEmoji(category) + " " + f.getCategoryTitle(ctx, guildID, category),
		Description: f.i18n.T(ctx, guildID, "menu.category_description"),
		Color:       int(shared.ColorInfo),
	}

	components := f.buildCategoryButtons(ctx, category, guildID, isAdmin)

	return f.respond(s, i, embed, components)
}

// displayFeatureList shows Tier 3 features for a sub-category.
func (f *Feature) displayFeatureList(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, category, subCategory string) error {
	guildID := i.GuildID
	userID := i.Member.User.ID
	isAdmin := f.checkAdminPermission(s, guildID, userID)

	embed := &discordgo.MessageEmbed{
		Title:       f.getSubCategoryTitle(ctx, guildID, subCategory),
		Description: f.i18n.T(ctx, guildID, "menu.select_feature"),
		Color:       int(shared.ColorInfo),
	}

	components := f.buildFeatureButtons(ctx, guildID, category, subCategory, isAdmin)

	return f.respond(s, i, embed, components)
}

// buildMainMenuButtons builds Tier 1 category buttons.
func (f *Feature) buildMainMenuButtons(ctx context.Context, guildID string, isAdmin bool) []discordgo.MessageComponent {
	components := []discordgo.MessageComponent{}
	
	// Admin category (Tier 1)
	if isAdmin {
		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    f.i18n.T(ctx, guildID, "menu.buttons.admin"),
					Style:    discordgo.PrimaryButton,
					CustomID: "menu:category:admin",
				},
			},
		})
	}
	
	// Information category (Tier 1, public)
	components = append(components, discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    f.i18n.T(ctx, guildID, "menu.buttons.information"),
				Style:    discordgo.PrimaryButton,
				CustomID: "menu:category:information",
			},
		},
	})
	
	return components
}

// buildCategoryButtons builds Tier 2 sub-category buttons or feature buttons.
func (f *Feature) buildCategoryButtons(ctx context.Context, category, guildID string, isAdmin bool) []discordgo.MessageComponent {
	components := []discordgo.MessageComponent{}
	
	if category == "admin" {
		// Admin has sub-categories
		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    f.i18n.T(ctx, guildID, "menu.buttons.configuration"),
					Style:    discordgo.PrimaryButton,
					CustomID: "menu:subcategory:admin:configuration",
				},
			},
		})
	} else if category == "information" {
		// Information shows features directly (no sub-categories)
		return f.buildFeatureButtons(ctx, guildID, category, "", isAdmin)
	}
	
	// Back button
	components = append(components, discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    f.i18n.T(ctx, guildID, "menu.buttons.back"),
				Style:    discordgo.SecondaryButton,
				CustomID: "menu:back:main",
			},
		},
	})
	
	return components
}

// buildFeatureButtons builds Tier 3 feature buttons.
func (f *Feature) buildFeatureButtons(ctx context.Context, guildID, category, subCategory string, isAdmin bool) []discordgo.MessageComponent {
	components := []discordgo.MessageComponent{}
	buttons := []discordgo.MessageComponent{}
	
	// Collect features for this sub-category
	for _, feature := range f.registry.GetAllFeatures() {
		btn := feature.GetMenuButton()
		if btn == nil {
			continue
		}
		
		// Filter by category and subcategory
		if btn.Category != category || btn.SubCategory != subCategory {
			continue
		}
		
		// Filter by permission
		if btn.AdminOnly && !isAdmin {
			continue
		}
		
		// Translate button label
		label := f.translateFeatureLabel(ctx, guildID, feature.Name(), btn.Label)
		
		buttons = append(buttons, discordgo.Button{
			Label:    label,
			Style:    discordgo.PrimaryButton,
			CustomID: btn.CustomID,
		})
	}
	
	// Add feature buttons (max 5 per row)
	// Discord allows a maximum of 5 buttons per ActionRow
	for i := 0; i < len(buttons); i += 5 {
		end := i + 5
		if end > len(buttons) {
			end = len(buttons)
		}
		components = append(components, discordgo.ActionsRow{
			Components: buttons[i:end],
		})
	}
	
	// Back button
	// If we're in a subcategory (under admin), go back to parent category
	// If we're at top level (like information), go back to main
	backTarget := "main"
	if subCategory != "" {
		// We're in a subcategory, go back to parent category
		backTarget = category
	}
	
	components = append(components, discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    f.i18n.T(ctx, guildID, "menu.buttons.back"),
				Style:    discordgo.SecondaryButton,
				CustomID: fmt.Sprintf("menu:back:%s", backTarget),
			},
		},
	})
	
	return components
}

// translateFeatureLabel translates feature button label.
func (f *Feature) translateFeatureLabel(ctx context.Context, guildID, featureName, fallback string) string {
	key := fmt.Sprintf("menu.features.%s", featureName)
	translated := f.i18n.T(ctx, guildID, key)
	
	// If translation returns the key itself (not found), use fallback
	if translated == key {
		return fallback
	}
	
	return translated
}

// checkAdminPermission checks if user has admin rights.
func (f *Feature) checkAdminPermission(s *discordgo.Session, guildID, userID string) bool {
	guild, err := s.Guild(guildID)
	if err != nil {
		return false
	}

	// Check 1: Server owner always has admin
	if guild.OwnerID == userID {
		return true
	}

	member, err := s.GuildMember(guildID, userID)
	if err != nil {
		return false
	}

	// Check 2: User has Administrator permission directly
	if member.Permissions&discordgo.PermissionAdministrator != 0 {
		return true
	}

	// Check 3: Check roles for Administrator permission or admin role name
	for _, roleID := range member.Roles {
		for _, guildRole := range guild.Roles {
			if guildRole.ID == roleID {
				if guildRole.Permissions&discordgo.PermissionAdministrator != 0 {
					return true
				}
				if guildRole.Name == shared.DefaultAdminRole {
					return true
				}
			}
		}
	}

	return false
}

// respond sends an interaction response.
func (f *Feature) respond(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
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

// getCategoryEmoji returns emoji for category.
func (f *Feature) getCategoryEmoji(category string) string {
	switch category {
	case "admin":
		return "👑"
	case "information":
		return "📊"
	default:
		return "📁"
	}
}

// getCategoryTitle returns translated title for category.
func (f *Feature) getCategoryTitle(ctx context.Context, guildID, category string) string {
	key := fmt.Sprintf("menu.category.%s", category)
	return f.i18n.T(ctx, guildID, key)
}

// getSubCategoryTitle returns translated title for sub-category.
func (f *Feature) getSubCategoryTitle(ctx context.Context, guildID, subCategory string) string {
	key := fmt.Sprintf("menu.subcategory.%s", subCategory)
	return f.i18n.T(ctx, guildID, key)
}

// respondError sends an error response.
func (f *Feature) respondError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) error {
	embed := &discordgo.MessageEmbed{
		Title:       "Error / エラー",
		Description: message,
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

