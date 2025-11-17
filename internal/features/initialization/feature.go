package initialization

import (
	"context"
	"fmt"

	"welcomebot/internal/bot"
	"welcomebot/internal/core/i18n"
	"welcomebot/internal/core/logger"

	"github.com/bwmarrin/discordgo"
)

const featureName = "initialization"

// LanguageSetupHandler defines interface for language setup feature.
type LanguageSetupHandler interface {
	ShowLanguagePicker(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error
}

// Feature orchestrates guild initialization.
type Feature struct {
	i18n            i18n.I18n
	logger          logger.Logger
	languageFeature LanguageSetupHandler
}

// New creates a new init feature.
func New(deps Dependencies) (*Feature, error) {
	if err := deps.Validate(); err != nil {
		return nil, fmt.Errorf("validate dependencies: %w", err)
	}

	return &Feature{
		i18n:   deps.I18n,
		logger: deps.Logger,
	}, nil
}

// SetLanguageFeature sets the language feature for delegation.
func (f *Feature) SetLanguageFeature(languageFeature LanguageSetupHandler) {
	f.languageFeature = languageFeature
}

// Name returns the feature name.
func (f *Feature) Name() string {
	return featureName
}

// HandleInteraction handles init-related interactions.
func (f *Feature) HandleInteraction(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Init doesn't handle interactions directly, only orchestrates
	return bot.ErrNotHandled
}

// RegisterCommands returns slash commands for this feature.
func (f *Feature) RegisterCommands() []*discordgo.ApplicationCommand {
	return nil // No commands, only orchestration
}

// GetMenuButton returns the menu button for this feature.
func (f *Feature) GetMenuButton() *bot.MenuButton {
	return nil // Init doesn't appear in menu
}

// CheckRequired checks if guild has all required settings.
func (f *Feature) CheckRequired(ctx context.Context, guildID string) (bool, []string) {
	missing := []string{}

	if !f.hasLanguage(ctx, guildID) {
		missing = append(missing, "language")
	}

	// Future: Add more required settings here
	// if !f.hasTimezone(ctx, guildID) {
	//     missing = append(missing, "timezone")
	// }

	return len(missing) == 0, missing
}

// StartInitWizard starts the initialization wizard for missing settings.
func (f *Feature) StartInitWizard(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, missing []string) error {
	guildID := i.GuildID

	f.logger.Info("starting init wizard",
		"guild_id", guildID,
		"missing", missing,
	)

	if contains(missing, "language") {
		return f.delegateToLanguage(ctx, s, i)
	}

	// Future: Handle other missing settings

	// All done (shouldn't reach here)
	return nil
}

// delegateToLanguage delegates to language feature.
func (f *Feature) delegateToLanguage(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	if f.languageFeature == nil {
		return fmt.Errorf("language feature not configured")
	}

	return f.languageFeature.ShowLanguagePicker(ctx, s, i)
}

// hasLanguage checks if guild has language configured.
func (f *Feature) hasLanguage(ctx context.Context, guildID string) bool {
	return f.i18n.HasGuildLanguage(ctx, guildID)
}

// contains checks if slice contains string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

