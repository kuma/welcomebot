-- Create table for guild welcome configuration
CREATE TABLE IF NOT EXISTS guild_welcome_config (
    guild_id VARCHAR(20) PRIMARY KEY,
    welcome_channel_id VARCHAR(20) NOT NULL,
    vc_category_id VARCHAR(20) NOT NULL,
    button_message_id VARCHAR(20),
    in_progress_role_id VARCHAR(20),
    completed_role_id VARCHAR(20),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create index on guild_id for faster lookups
CREATE INDEX IF NOT EXISTS idx_guild_welcome_config_guild_id ON guild_welcome_config(guild_id);

