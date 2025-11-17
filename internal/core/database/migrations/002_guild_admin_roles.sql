-- Migration: Guild admin role configuration
-- Created: 2025-10-28

CREATE TABLE IF NOT EXISTS guild_admin_roles (
    guild_id VARCHAR(20) PRIMARY KEY,
    role_name VARCHAR(100) NOT NULL,
    created_by VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Index for faster lookups
CREATE INDEX IF NOT EXISTS idx_guild_admin_roles_guild ON guild_admin_roles(guild_id);

-- Comments
COMMENT ON TABLE guild_admin_roles IS 'Stores custom admin role per Discord guild';
COMMENT ON COLUMN guild_admin_roles.guild_id IS 'Discord guild (server) ID';
COMMENT ON COLUMN guild_admin_roles.role_name IS 'Custom admin role name for this guild';
COMMENT ON COLUMN guild_admin_roles.created_by IS 'User ID who set this role';

