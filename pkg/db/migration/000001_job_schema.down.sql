-- Drop trigger
DROP TRIGGER IF EXISTS update_cron_jobs_modtime ON cron_jobs;

-- Drop indexes
DROP INDEX IF EXISTS idx_cron_jobs_tenant_id;

-- Drop table
DROP TABLE IF EXISTS cron_jobs;