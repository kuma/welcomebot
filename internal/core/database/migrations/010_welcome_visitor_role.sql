-- Add visitor role column to guild_welcome_config table
ALTER TABLE guild_welcome_config
    ADD COLUMN visitor_role_id VARCHAR(20);

