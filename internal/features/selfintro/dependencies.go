package selfintro

import (
	"errors"

	"welcomebot/internal/core/cache"
	"welcomebot/internal/core/database"
	"welcomebot/internal/core/i18n"
	"welcomebot/internal/core/logger"
)

// Dependencies contains all required dependencies for the selfintro feature.
type Dependencies struct {
	DB     database.Client
	Cache  cache.Client
	I18n   i18n.I18n
	Logger logger.Logger
}

// Validate ensures all required dependencies are present.
func (d Dependencies) Validate() error {
	if d.DB == nil {
		return errors.New("database is required")
	}
	if d.Cache == nil {
		return errors.New("cache is required")
	}
	if d.I18n == nil {
		return errors.New("i18n is required")
	}
	if d.Logger == nil {
		return errors.New("logger is required")
	}
	return nil
}

