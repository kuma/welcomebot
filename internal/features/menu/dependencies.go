package menu

import (
	"context"
	"errors"

	"welcomebot/internal/bot"
	"welcomebot/internal/core/i18n"
	"welcomebot/internal/core/logger"

	"github.com/bwmarrin/discordgo"
)

// InitChecker checks if guild initialization is complete.
type InitChecker interface {
	CheckRequired(ctx context.Context, guildID string) (bool, []string)
	StartInitWizard(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, missing []string) error
}

// FeatureRegistry provides access to registered features.
type FeatureRegistry interface {
	GetAllFeatures() []bot.Feature
}

// Dependencies contains all required dependencies for the menu feature.
type Dependencies struct {
	Registry FeatureRegistry
	Init     InitChecker
	I18n     i18n.I18n
	Logger   logger.Logger
}

// Validate ensures all required dependencies are present.
func (d Dependencies) Validate() error {
	if d.Registry == nil {
		return errors.New("registry is required")
	}
	if d.Init == nil {
		return errors.New("init is required")
	}
	if d.I18n == nil {
		return errors.New("i18n is required")
	}
	if d.Logger == nil {
		return errors.New("logger is required")
	}
	return nil
}

