-- Create table for guild other roles configuration (shared by other roles 1 and 2)
CREATE TABLE IF NOT EXISTS guild_other_roles_config (
    guild_id VARCHAR(20) PRIMARY KEY,
    -- Other Roles 1
    ero_ok_role_id VARCHAR(20),
    ero_ng_role_id VARCHAR(20),
    neochi_ok_role_id VARCHAR(20),
    neochi_ng_role_id VARCHAR(20),
    neochi_disconnect_role_id VARCHAR(20),
    -- Other Roles 2
    dm_ok_role_id VARCHAR(20),
    dm_ng_role_id VARCHAR(20),
    friend_ok_role_id VARCHAR(20),
    friend_ng_role_id VARCHAR(20),
    bunnyclub_event_role_id VARCHAR(20),
    user_event_role_id VARCHAR(20),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create index on guild_id for faster lookups
CREATE INDEX IF NOT EXISTS idx_guild_other_roles_config_guild_id ON guild_other_roles_config(guild_id);

