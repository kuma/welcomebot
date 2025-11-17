-- Migration: Guild language preferences
-- Created: 2025-10-28

CREATE TABLE IF NOT EXISTS guild_languages (
    guild_id VARCHAR(20) PRIMARY KEY,
    language_code VARCHAR(5) NOT NULL DEFAULT 'en',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Index for faster lookups
CREATE INDEX IF NOT EXISTS idx_guild_languages_guild ON guild_languages(guild_id);

-- Comments
COMMENT ON TABLE guild_languages IS 'Stores language preference per Discord guild';
COMMENT ON COLUMN guild_languages.guild_id IS 'Discord guild (server) ID';
COMMENT ON COLUMN guild_languages.language_code IS 'ISO language code (en, ja, etc)';

