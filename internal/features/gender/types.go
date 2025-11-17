package gender

import "time"

// GenderConfig represents gender role configuration for a guild.
type GenderConfig struct {
	GuildID      string    `json:"guild_id"`
	MaleRoleID   string    `json:"male_role_id"`
	FemaleRoleID string    `json:"female_role_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

const (
	cacheKeyPrefix = "welcomebot:gender:"
)


