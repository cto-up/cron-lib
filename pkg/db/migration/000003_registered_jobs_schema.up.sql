-- cron_registered_jobs definition
CREATE UNLOGGED TABLE cron_registered_jobs (
    id uuid NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    job_name VARCHAR NOT NULL,
    schedule VARCHAR NOT NULL,
    is_long_running BOOLEAN NOT NULL DEFAULT false,
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    last_registered_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    instance_id VARCHAR(128) NOT NULL,
    tenant_id varchar(64) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Add a unique constraint to prevent duplicate jobs
ALTER TABLE cron_registered_jobs ADD CONSTRAINT cron_registered_jobs_uniq UNIQUE (tenant_id, job_name);

-- Add index for better performance
CREATE INDEX idx_cron_registered_jobs_tenant_id ON cron_registered_jobs ("tenant_id");

-- Add trigger for updated_at
CREATE TRIGGER update_cron_registered_jobs_modtime
BEFORE UPDATE ON cron_registered_jobs
FOR EACH ROW
EXECUTE FUNCTION update_modified_column();