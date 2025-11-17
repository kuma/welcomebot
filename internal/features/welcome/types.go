package welcome

import "time"

const (
	cacheKeyPrefix = "welcomebot:config:"
	slaveStatusKey = "welcomebot:slaves:status:"
	sessionKeyPrefix = "welcomebot:session:"
)

// WelcomeConfig represents welcome configuration for a guild.
type WelcomeConfig struct {
	GuildID             string    `json:"guild_id"`
	WelcomeChannelID    string    `json:"welcome_channel_id"`
	VCCategoryID        string    `json:"vc_category_id"`
	ButtonMessageID     string    `json:"button_message_id"`
	InProgressRoleID    string    `json:"in_progress_role_id,omitempty"`
	CompletedRoleID     string    `json:"completed_role_id,omitempty"`
	EntranceRoleID      string    `json:"entrance_role_id,omitempty"`
	NyukaiRoleID        string    `json:"nyukai_role_id,omitempty"`
	Setsumeikai1RoleID  string    `json:"setsumeikai_1_role_id,omitempty"`
	Setsumeikai2RoleID  string    `json:"setsumeikai_2_role_id,omitempty"`
	Setsumeikai3RoleID  string    `json:"setsumeikai_3_role_id,omitempty"`
	MemberRoleID        string    `json:"member_role_id,omitempty"`
	VisitorRoleID       string    `json:"visitor_role_id,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// SlaveStatus represents the current status of a slave bot.
type SlaveStatus string

const (
	SlaveStatusAvailable SlaveStatus = "available"
	SlaveStatusBusy      SlaveStatus = "busy"
	SlaveStatusOffline   SlaveStatus = "offline"
)

// OnboardingSession represents an active onboarding session.
type OnboardingSession struct {
	GuildID      string    `json:"guild_id"`
	UserID       string    `json:"user_id"`
	SlaveID      string    `json:"slave_id"`
	VoiceChannel string    `json:"voice_channel_id"`
	StartedAt    time.Time `json:"started_at"`
}

// SlaveInfo represents information about a slave bot.
type SlaveInfo struct {
	ID         string      `json:"id"`
	Status     SlaveStatus `json:"status"`
	LastUpdate time.Time   `json:"last_update"`
}

// WizardState tracks the configuration wizard progress.
type WizardState struct {
	GuildID             string `json:"guild_id"`
	WelcomeChannelID    string `json:"welcome_channel_id"`
	VCCategoryID        string `json:"vc_category_id"`
	EntranceRoleID      string `json:"entrance_role_id"`
	NyukaiRoleID        string `json:"nyukai_role_id"`
	Setsumeikai1RoleID  string `json:"setsumeikai_1_role_id"`
	Setsumeikai2RoleID  string `json:"setsumeikai_2_role_id"`
	Setsumeikai3RoleID  string `json:"setsumeikai_3_role_id"`
	MemberRoleID        string `json:"member_role_id"`
	VisitorRoleID       string `json:"visitor_role_id"`
	CurrentStep         int    `json:"current_step"`
}

var (
	// SlaveIDs represents the three slave bot instances
	SlaveIDs = []string{"slave-1", "slave-2", "slave-3"}
)

// AgeRangeConfig represents age range role configuration for a guild.
type AgeRangeConfig struct {
	GuildID          string `json:"guild_id"`
	Age20EarlyRoleID string `json:"age_20_early_role_id,omitempty"`
	Age20LateRoleID  string `json:"age_20_late_role_id,omitempty"`
	Age30EarlyRoleID string `json:"age_30_early_role_id,omitempty"`
	Age30LateRoleID  string `json:"age_30_late_role_id,omitempty"`
	Age40EarlyRoleID string `json:"age_40_early_role_id,omitempty"`
	Age40LateRoleID  string `json:"age_40_late_role_id,omitempty"`
}

// VoiceTypeConfig represents voice type role configuration for a guild.
type VoiceTypeConfig struct {
	GuildID       string `json:"guild_id"`
	HighRoleID    string `json:"high_role_id,omitempty"`
	MidHighRoleID string `json:"mid_high_role_id,omitempty"`
	MidRoleID     string `json:"mid_role_id,omitempty"`
	MidLowRoleID  string `json:"mid_low_role_id,omitempty"`
	LowRoleID     string `json:"low_role_id,omitempty"`
}

// OtherRolesConfig represents other roles configuration for a guild.
type OtherRolesConfig struct {
	GuildID                string `json:"guild_id"`
	EroOkRoleID            string `json:"ero_ok_role_id,omitempty"`
	EroNgRoleID            string `json:"ero_ng_role_id,omitempty"`
	NeochiOkRoleID         string `json:"neochi_ok_role_id,omitempty"`
	NeochiNgRoleID         string `json:"neochi_ng_role_id,omitempty"`
	NeochiDisconnectRoleID string `json:"neochi_disconnect_role_id,omitempty"`
	DmOkRoleID             string `json:"dm_ok_role_id,omitempty"`
	DmNgRoleID             string `json:"dm_ng_role_id,omitempty"`
	FriendOkRoleID         string `json:"friend_ok_role_id,omitempty"`
	FriendNgRoleID         string `json:"friend_ng_role_id,omitempty"`
	BunnyclubEventRoleID   string `json:"bunnyclub_event_role_id,omitempty"`
	UserEventRoleID        string `json:"user_event_role_id,omitempty"`
}

