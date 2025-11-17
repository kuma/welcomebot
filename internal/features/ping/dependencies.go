package ping

import (
	"errors"

	"welcomebot/internal/core/logger"
)

// Dependencies contains all required dependencies for the ping feature.
type Dependencies struct {
	Logger logger.Logger
}

// Validate ensures all required dependencies are present.
func (d Dependencies) Validate() error {
	if d.Logger == nil {
		return errors.New("logger is required")
	}
	return nil
}

