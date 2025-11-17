-- Create table for guild age range configuration
CREATE TABLE IF NOT EXISTS guild_age_range_config (
    guild_id VARCHAR(20) PRIMARY KEY,
    age_20_early_role_id VARCHAR(20),
    age_20_late_role_id VARCHAR(20),
    age_30_early_role_id VARCHAR(20),
    age_30_late_role_id VARCHAR(20),
    age_40_early_role_id VARCHAR(20),
    age_40_late_role_id VARCHAR(20),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create index on guild_id for faster lookups
CREATE INDEX IF NOT EXISTS idx_guild_age_range_config_guild_id ON guild_age_range_config(guild_id);

