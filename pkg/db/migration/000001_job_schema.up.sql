-- Create update_modified_column function if it doesn't exist
-- Use for test containers
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- cron_jobs definition
CREATE UNLOGGED TABLE cron_jobs (
    id uuid NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    "lock" VARCHAR NOT NULL,
    "job_name" VARCHAR NOT NULL,
    status VARCHAR(20) NOT NULL,
    last_execution_time TIMESTAMPTZ NULL,
    next_execution_time TIMESTAMPTZ NULL,
    locked_by VARCHAR(128) NULL,
    locked_at TIMESTAMPTZ NULL,
    tenant_id varchar(64) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Add a unique constraint to prevent duplicate jobs
ALTER TABLE cron_jobs ADD CONSTRAINT cron_jobs_uniq UNIQUE (tenant_id, lock);

-- Add an index for better performance
CREATE INDEX idx_cron_jobs_locked ON cron_jobs (locked_at);

CREATE INDEX idx_cron_jobs_tenant_id ON cron_jobs ("tenant_id");

CREATE TRIGGER update_cron_jobs_modtime
BEFORE UPDATE ON cron_jobs
FOR EACH ROW
EXECUTE FUNCTION update_modified_column();
