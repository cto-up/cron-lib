-- Drop trigger
DROP TRIGGER IF EXISTS update_cron_job_audit_logs_modtime ON cron_job_audit_logs;

-- Drop indexes
DROP INDEX IF EXISTS idx_cron_job_audit_logs_app_id;
DROP INDEX IF EXISTS idx_cron_job_audit_logs_request_id;
DROP INDEX IF EXISTS idx_cron_job_audit_logs_job_name;
DROP INDEX IF EXISTS idx_cron_job_audit_logs_tenant_id;

-- Drop table
DROP TABLE IF EXISTS cron_job_audit_logs;