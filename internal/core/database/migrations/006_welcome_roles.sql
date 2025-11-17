-- Add role columns to guild_welcome_config table
ALTER TABLE guild_welcome_config
    ADD COLUMN entrance_role_id VARCHAR(20),
    ADD COLUMN nyukai_role_id VARCHAR(20),
    ADD COLUMN setsumeikai_1_role_id VARCHAR(20),
    ADD COLUMN setsumeikai_2_role_id VARCHAR(20),
    ADD COLUMN setsumeikai_3_role_id VARCHAR(20),
    ADD COLUMN member_role_id VARCHAR(20);
