package selfintro

import "time"

// SelfIntroConfig represents self-intro channel configuration for a guild.
type SelfIntroConfig struct {
	GuildID         string    `json:"guild_id"`
	MaleChannelID   string    `json:"male_channel_id"`
	FemaleChannelID string    `json:"female_channel_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

const (
	cacheKeyPrefix = "welcomebot:selfintro:"
)

