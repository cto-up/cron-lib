-- +goose Up
-- is_enabled is an admin decision, not a cache: an UNLOGGED table is truncated
-- by crash recovery, which silently re-enabled every disabled job.
ALTER TABLE cron_registered_jobs SET LOGGED;

-- Why this job is not running, when registration refused it (e.g. a schedule
-- the cron parser rejects). NULL when the last registration succeeded.
ALTER TABLE cron_registered_jobs ADD COLUMN registration_error TEXT NULL;

-- +goose Down
ALTER TABLE cron_registered_jobs DROP COLUMN IF EXISTS registration_error;
ALTER TABLE cron_registered_jobs SET UNLOGGED;
