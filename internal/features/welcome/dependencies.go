package welcome

import (
	"errors"

	"welcomebot/internal/core/cache"
	"welcomebot/internal/core/database"
	"welcomebot/internal/core/i18n"
	"welcomebot/internal/core/logger"
	"welcomebot/internal/core/queue"

	"github.com/bwmarrin/discordgo"
)

// Dependencies contains all required dependencies for the welcome feature.
type Dependencies struct {
	DB      database.Client
	Cache   cache.Client
	Queue   queue.Client
	I18n    i18n.I18n
	Logger  logger.Logger
	Session *discordgo.Session
}

// Validate ensures all required dependencies are present.
func (d Dependencies) Validate() error {
	if d.DB == nil {
		return errors.New("database is required")
	}
	if d.Cache == nil {
		return errors.New("cache is required")
	}
	if d.Queue == nil {
		return errors.New("queue is required")
	}
	if d.I18n == nil {
		return errors.New("i18n is required")
	}
	if d.Logger == nil {
		return errors.New("logger is required")
	}
	if d.Session == nil {
		return errors.New("discord session is required")
	}
	return nil
}

