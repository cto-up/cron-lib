-- +goose Up
-- Up Migration: Remove time zone from relevant columns
ALTER TABLE cron_job_audit_logs
    ALTER COLUMN scheduled_time TYPE TIMESTAMP WITHOUT TIME ZONE,
    ALTER COLUMN start_time TYPE TIMESTAMP WITHOUT TIME ZONE,
    ALTER COLUMN end_time TYPE TIMESTAMP WITHOUT TIME ZONE,
    ALTER COLUMN created_at TYPE TIMESTAMP WITHOUT TIME ZONE,
    ALTER COLUMN updated_at TYPE TIMESTAMP WITHOUT TIME ZONE;
-- +goose Down
-- Down Migration: Restore time zone awareness
ALTER TABLE cron_job_audit_logs
    ALTER COLUMN scheduled_time TYPE TIMESTAMPTZ,
    ALTER COLUMN start_time TYPE TIMESTAMPTZ,
    ALTER COLUMN end_time TYPE TIMESTAMPTZ,
    ALTER COLUMN created_at TYPE TIMESTAMPTZ,
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ;