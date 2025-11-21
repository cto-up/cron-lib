-- cron_job_audit_logs definition
CREATE TABLE cron_job_audit_logs (
    id uuid NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    app_id VARCHAR(64) NOT NULL,
    request_id VARCHAR(128) NOT NULL,
    job_name VARCHAR(128) NOT NULL,
    scheduled_time TIMESTAMP WITH TIME ZONE NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE,
    end_time TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) NOT NULL,
    output TEXT,
    error TEXT,
    user_id varchar(128) NOT NULL,
    tenant_id varchar(64) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_cron_job_audit_logs_app_id ON cron_job_audit_logs ("app_id");
CREATE INDEX idx_cron_job_audit_logs_request_id ON cron_job_audit_logs ("request_id");
CREATE INDEX idx_cron_job_audit_logs_job_name ON cron_job_audit_logs ("job_name");
CREATE INDEX idx_cron_job_audit_logs_tenant_id ON cron_job_audit_logs ("tenant_id");

CREATE TRIGGER update_cron_job_audit_logs_modtime
BEFORE UPDATE ON cron_job_audit_logs
FOR EACH ROW
EXECUTE FUNCTION update_modified_column();
