package voicetype

import "time"

const (
	cacheKeyPrefix = "welcomebot:voicetype:config:"
)

// VoiceTypeConfig represents voice type role configuration for a guild.
type VoiceTypeConfig struct {
	GuildID      string    `json:"guild_id"`
	HighRoleID   string    `json:"high_role_id,omitempty"`
	MidHighRoleID string   `json:"mid_high_role_id,omitempty"`
	MidRoleID    string    `json:"mid_role_id,omitempty"`
	MidLowRoleID string    `json:"mid_low_role_id,omitempty"`
	LowRoleID    string    `json:"low_role_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// WizardState tracks the configuration wizard progress.
type WizardState struct {
	GuildID       string `json:"guild_id"`
	HighRoleID    string `json:"high_role_id"`
	MidHighRoleID string `json:"mid_high_role_id"`
	MidRoleID     string `json:"mid_role_id"`
	MidLowRoleID  string `json:"mid_low_role_id"`
	LowRoleID     string `json:"low_role_id"`
	CurrentStep   int    `json:"current_step"`
}

