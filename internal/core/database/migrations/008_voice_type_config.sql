-- Create table for guild voice type configuration
CREATE TABLE IF NOT EXISTS guild_voice_type_config (
    guild_id VARCHAR(20) PRIMARY KEY,
    high_role_id VARCHAR(20),
    mid_high_role_id VARCHAR(20),
    mid_role_id VARCHAR(20),
    mid_low_role_id VARCHAR(20),
    low_role_id VARCHAR(20),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create index on guild_id for faster lookups
CREATE INDEX IF NOT EXISTS idx_guild_voice_type_config_guild_id ON guild_voice_type_config(guild_id);

