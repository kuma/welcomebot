-- Migration: Guild self-introduction channels configuration
-- Created: 2025-10-28

CREATE TABLE IF NOT EXISTS guild_selfintro_channels (
    guild_id VARCHAR(20) PRIMARY KEY,
    male_channel_id VARCHAR(20) NOT NULL,
    female_channel_id VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Index for faster lookups
CREATE INDEX IF NOT EXISTS idx_guild_selfintro_channels_guild ON guild_selfintro_channels(guild_id);

-- Comments
COMMENT ON TABLE guild_selfintro_channels IS 'Stores self-introduction channel configuration per Discord guild';
COMMENT ON COLUMN guild_selfintro_channels.guild_id IS 'Discord guild (server) ID';
COMMENT ON COLUMN guild_selfintro_channels.male_channel_id IS 'Text channel ID for male self-introductions';
COMMENT ON COLUMN guild_selfintro_channels.female_channel_id IS 'Text channel ID for female self-introductions';

