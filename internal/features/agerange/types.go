package agerange

import "time"

const (
	cacheKeyPrefix = "welcomebot:agerange:config:"
)

// AgeRangeConfig represents age range role configuration for a guild.
type AgeRangeConfig struct {
	GuildID           string    `json:"guild_id"`
	Age20EarlyRoleID  string    `json:"age_20_early_role_id,omitempty"`
	Age20LateRoleID   string    `json:"age_20_late_role_id,omitempty"`
	Age30EarlyRoleID  string    `json:"age_30_early_role_id,omitempty"`
	Age30LateRoleID   string    `json:"age_30_late_role_id,omitempty"`
	Age40EarlyRoleID  string    `json:"age_40_early_role_id,omitempty"`
	Age40LateRoleID   string    `json:"age_40_late_role_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// WizardState tracks the configuration wizard progress.
type WizardState struct {
	GuildID          string `json:"guild_id"`
	Age20EarlyRoleID string `json:"age_20_early_role_id"`
	Age20LateRoleID  string `json:"age_20_late_role_id"`
	Age30EarlyRoleID string `json:"age_30_early_role_id"`
	Age30LateRoleID  string `json:"age_30_late_role_id"`
	Age40EarlyRoleID string `json:"age_40_early_role_id"`
	Age40LateRoleID  string `json:"age_40_late_role_id"`
	CurrentStep      int    `json:"current_step"`
}

