package otherroles2

import "time"

const (
	cacheKeyPrefix = "welcomebot:otherroles:config:" // Shared cache key with otherroles1
)

// OtherRolesConfig represents the FULL other roles configuration for a guild (shared table).
// This feature only manages the "Other Roles 2" subset.
type OtherRolesConfig struct {
	GuildID string `json:"guild_id"`
	
	// Other Roles 1 (not managed by this feature, but part of same table)
	EroOKRoleID            string `json:"ero_ok_role_id,omitempty"`
	EroNGRoleID            string `json:"ero_ng_role_id,omitempty"`
	NeochiOKRoleID         string `json:"neochi_ok_role_id,omitempty"`
	NeochiNGRoleID         string `json:"neochi_ng_role_id,omitempty"`
	NeochiDisconnectRoleID string `json:"neochi_disconnect_role_id,omitempty"`
	
	// Other Roles 2 (managed by this feature)
	DMOKRoleID           string `json:"dm_ok_role_id,omitempty"`
	DMNGRoleID           string `json:"dm_ng_role_id,omitempty"`
	FriendOKRoleID       string `json:"friend_ok_role_id,omitempty"`
	FriendNGRoleID       string `json:"friend_ng_role_id,omitempty"`
	BunnyclubEventRoleID string `json:"bunnyclub_event_role_id,omitempty"`
	UserEventRoleID      string `json:"user_event_role_id,omitempty"`
	
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WizardState tracks the configuration wizard progress for Other Roles 2.
type WizardState struct {
	GuildID              string `json:"guild_id"`
	DMOKRoleID           string `json:"dm_ok_role_id"`
	DMNGRoleID           string `json:"dm_ng_role_id"`
	FriendOKRoleID       string `json:"friend_ok_role_id"`
	FriendNGRoleID       string `json:"friend_ng_role_id"`
	BunnyclubEventRoleID string `json:"bunnyclub_event_role_id"`
	UserEventRoleID      string `json:"user_event_role_id"`
	CurrentStep          int    `json:"current_step"`
}

