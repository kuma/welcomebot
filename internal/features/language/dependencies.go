package language

import (
	"errors"

	"welcomebot/internal/core/i18n"
	"welcomebot/internal/core/logger"
)

// Dependencies contains all required dependencies for the language feature.
type Dependencies struct {
	I18n   i18n.I18n
	Logger logger.Logger
}

// Validate ensures all required dependencies are present.
func (d Dependencies) Validate() error {
	if d.I18n == nil {
		return errors.New("i18n is required")
	}
	if d.Logger == nil {
		return errors.New("logger is required")
	}
	return nil
}

