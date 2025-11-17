package shared

import "time"

// Common Discord-related types used across features.

// GuildID represents a Discord guild (server) ID.
type GuildID string

// ChannelID represents a Discord channel ID.
type ChannelID string

// UserID represents a Discord user ID.
type UserID string

// RoleID represents a Discord role ID.
type RoleID string

// MessageID represents a Discord message ID.
type MessageID string

// EmbedColor represents colors for Discord embeds.
type EmbedColor int

const (
	// ColorDefault is light blue/info color.
	ColorDefault EmbedColor = 0x00AAFF
	// ColorSuccess is green color.
	ColorSuccess EmbedColor = 0x2ECC71
	// ColorWarning is yellow color.
	ColorWarning EmbedColor = 0xFEE75C
	// ColorError is red color.
	ColorError EmbedColor = 0xED4245
	// ColorInfo is Discord blurple color.
	ColorInfo EmbedColor = 0x7289DA
)

// Common cache TTL durations.
const (
	// TTLShort is for frequently changing data (5 minutes).
	TTLShort = 5 * time.Minute
	// TTLMedium is for moderately stable data (30 minutes).
	TTLMedium = 30 * time.Minute
	// TTLLong is for stable data (2 hours).
	TTLLong = 2 * time.Hour
	// TTLDay is for very stable data (24 hours).
	TTLDay = 24 * time.Hour
)

// Redis key prefixes for consistent naming.
const (
	RedisKeyPrefix  = "welcomebot:"
	RedisKeyGuild   = RedisKeyPrefix + "guild:"
	RedisKeyUser    = RedisKeyPrefix + "user:"
	RedisKeyChannel = RedisKeyPrefix + "channel:"
	RedisKeyConfig  = RedisKeyPrefix + "config:"
	RedisKeyFeature = RedisKeyPrefix + "feature:"
)
