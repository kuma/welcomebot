-- Migration: Guild gender role configuration
-- Created: 2025-10-28

CREATE TABLE IF NOT EXISTS guild_gender_roles (
    guild_id VARCHAR(20) PRIMARY KEY,
    male_role_id VARCHAR(20) NOT NULL,
    female_role_id VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Index for faster lookups
CREATE INDEX IF NOT EXISTS idx_guild_gender_roles_guild ON guild_gender_roles(guild_id);

-- Comments
COMMENT ON TABLE guild_gender_roles IS 'Stores gender role configuration per Discord guild';
COMMENT ON COLUMN guild_gender_roles.guild_id IS 'Discord guild (server) ID';
COMMENT ON COLUMN guild_gender_roles.male_role_id IS 'Discord role ID for male members';
COMMENT ON COLUMN guild_gender_roles.female_role_id IS 'Discord role ID for female members';


