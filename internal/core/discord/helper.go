package discord

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// Helper provides Discord API operations.
type Helper interface {
	CreateChannel(ctx context.Context, guildID string, cfg ChannelConfig) (string, error)
	DeleteChannel(ctx context.Context, channelID string) error
	SendMessage(ctx context.Context, channelID string, msg Message) error
	SendEmbed(ctx context.Context, channelID string, embed *discordgo.MessageEmbed) error
	EditMessage(ctx context.Context, channelID, messageID string, content string) error
	DeleteMessage(ctx context.Context, channelID, messageID string) error
	AddRole(ctx context.Context, guildID, userID, roleID string) error
	RemoveRole(ctx context.Context, guildID, userID, roleID string) error
}

// ChannelConfig contains channel creation configuration.
type ChannelConfig struct {
	Name       string
	Type       discordgo.ChannelType
	CategoryID string
	Position   int
}

// Message represents a Discord message to send.
type Message struct {
	Content string
	Embed   *discordgo.MessageEmbed
}

// discordHelper implements Helper using discordgo.
type discordHelper struct {
	session *discordgo.Session
}

// New creates a new Discord helper with the given session.
func New(session *discordgo.Session) Helper {
	return &discordHelper{session: session}
}

// CreateChannel creates a new Discord channel.
func (h *discordHelper) CreateChannel(ctx context.Context, guildID string, cfg ChannelConfig) (string, error) {
	channel, err := h.session.GuildChannelCreateComplex(guildID, discordgo.GuildChannelCreateData{
		Name:     cfg.Name,
		Type:     cfg.Type,
		ParentID: cfg.CategoryID,
		Position: cfg.Position,
	})
	if err != nil {
		return "", fmt.Errorf("create channel: %w", err)
	}

	return channel.ID, nil
}

// DeleteChannel deletes a Discord channel.
func (h *discordHelper) DeleteChannel(ctx context.Context, channelID string) error {
	_, err := h.session.ChannelDelete(channelID)
	if err != nil {
		return fmt.Errorf("delete channel %s: %w", channelID, err)
	}
	return nil
}

// SendMessage sends a message to a Discord channel.
func (h *discordHelper) SendMessage(ctx context.Context, channelID string, msg Message) error {
	_, err := h.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: msg.Content,
		Embed:   msg.Embed,
	})
	if err != nil {
		return fmt.Errorf("send message to %s: %w", channelID, err)
	}
	return nil
}

// SendEmbed sends an embed to a Discord channel.
func (h *discordHelper) SendEmbed(ctx context.Context, channelID string, embed *discordgo.MessageEmbed) error {
	return h.SendMessage(ctx, channelID, Message{Embed: embed})
}

// EditMessage edits an existing Discord message.
func (h *discordHelper) EditMessage(ctx context.Context, channelID, messageID string, content string) error {
	_, err := h.session.ChannelMessageEdit(channelID, messageID, content)
	if err != nil {
		return fmt.Errorf("edit message %s: %w", messageID, err)
	}
	return nil
}

// DeleteMessage deletes a Discord message.
func (h *discordHelper) DeleteMessage(ctx context.Context, channelID, messageID string) error {
	err := h.session.ChannelMessageDelete(channelID, messageID)
	if err != nil {
		return fmt.Errorf("delete message %s: %w", messageID, err)
	}
	return nil
}

// AddRole adds a role to a guild member.
func (h *discordHelper) AddRole(ctx context.Context, guildID, userID, roleID string) error {
	err := h.session.GuildMemberRoleAdd(guildID, userID, roleID)
	if err != nil {
		return fmt.Errorf("add role %s to user %s: %w", roleID, userID, err)
	}
	return nil
}

// RemoveRole removes a role from a guild member.
func (h *discordHelper) RemoveRole(ctx context.Context, guildID, userID, roleID string) error {
	err := h.session.GuildMemberRoleRemove(guildID, userID, roleID)
	if err != nil {
		return fmt.Errorf("remove role %s from user %s: %w", roleID, userID, err)
	}
	return nil
}
