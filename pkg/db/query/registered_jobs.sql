
-- name: UpsertRegisteredJob :one
INSERT INTO cron_registered_jobs (
  job_name, schedule, is_long_running, is_enabled, 
  last_registered_at, instance_id, tenant_id
) VALUES (
  sqlc.arg('job_name')::text,
  sqlc.arg('schedule')::text,
  sqlc.arg('is_long_running')::boolean,
  sqlc.arg('is_enabled')::boolean,
  NOW(),
  sqlc.arg('instance_id')::text,
  sqlc.arg('tenant_id')::text
)
ON CONFLICT (tenant_id, job_name) DO UPDATE
SET 
  schedule = EXCLUDED.schedule,
  is_long_running = EXCLUDED.is_long_running,
  is_enabled = EXCLUDED.is_enabled,
  last_registered_at = NOW(),
  instance_id = EXCLUDED.instance_id,
  updated_at = NOW()
RETURNING *;

-- name: ListRegisteredJobs :many
SELECT rj.*, 
  (SELECT COUNT(*) FROM cron_job_audit_logs al 
   WHERE al.job_name = rj.job_name AND al.tenant_id = rj.tenant_id) as execution_count
FROM cron_registered_jobs rj
WHERE rj.tenant_id = sqlc.arg('tenant_id')::text
  AND (UPPER(rj.job_name) LIKE UPPER(sqlc.arg('search_term')) OR sqlc.arg('search_term') IS NULL)
ORDER BY
  CASE WHEN sqlc.arg('sort_by')::TEXT = 'job_name' AND sqlc.arg('order')::TEXT = 'asc' THEN rj.job_name END ASC,
  CASE WHEN sqlc.arg('sort_by')::TEXT = 'job_name' AND sqlc.arg('order')::TEXT != 'asc' THEN rj.job_name END DESC,
  CASE WHEN sqlc.arg('sort_by')::TEXT = 'schedule' AND sqlc.arg('order')::TEXT = 'asc' THEN rj.schedule END ASC,
  CASE WHEN sqlc.arg('sort_by')::TEXT = 'schedule' AND sqlc.arg('order')::TEXT != 'asc' THEN rj.schedule END DESC,
  CASE WHEN sqlc.arg('sort_by')::TEXT = 'last_registered_at' AND sqlc.arg('order')::TEXT = 'asc' THEN rj.last_registered_at END ASC,
  CASE WHEN sqlc.arg('sort_by')::TEXT = 'last_registered_at' AND sqlc.arg('order')::TEXT != 'asc' THEN rj.last_registered_at END DESC,
  CASE WHEN sqlc.arg('sort_by')::TEXT = 'is_enabled' AND sqlc.arg('order')::TEXT = 'asc' THEN rj.is_enabled END ASC,
  CASE WHEN sqlc.arg('sort_by')::TEXT = 'is_enabled' AND sqlc.arg('order')::TEXT != 'asc' THEN rj.is_enabled END DESC
LIMIT sqlc.arg('limit')::int
OFFSET sqlc.arg('offset')::int;

-- name: CountRegisteredJobs :one
SELECT COUNT(*) 
FROM cron_registered_jobs
WHERE tenant_id = sqlc.arg('tenant_id')::text
  AND (UPPER(job_name) LIKE UPPER(sqlc.arg('search_term')) OR sqlc.arg('search_term') IS NULL);

-- name: GetRegisteredJobByID :one
SELECT * 
FROM cron_registered_jobs
WHERE id = sqlc.arg('id')::uuid
  AND tenant_id = sqlc.arg('tenant_id')::text;

-- name: UpdateRegisteredJobEnabled :execresult
UPDATE cron_registered_jobs
SET is_enabled = sqlc.arg('is_enabled')::boolean,
    updated_at = NOW()
WHERE id = sqlc.arg('id')::uuid
  AND tenant_id = sqlc.arg('tenant_id')::text;

-- name: ListJobAuditLogsByJobName :many
SELECT * 
FROM cron_job_audit_logs
WHERE job_name = sqlc.arg('job_name')::text
  AND tenant_id = sqlc.arg('tenant_id')::text
ORDER BY
  CASE WHEN sqlc.arg('sort_by')::TEXT = 'scheduled_time' AND sqlc.arg('order')::TEXT = 'asc' THEN scheduled_time END ASC,
  CASE WHEN sqlc.arg('sort_by')::TEXT = 'scheduled_time' AND sqlc.arg('order')::TEXT != 'asc' THEN scheduled_time END DESC,
  CASE WHEN sqlc.arg('sort_by')::TEXT = 'start_time' AND sqlc.arg('order')::TEXT = 'asc' THEN start_time END ASC,
  CASE WHEN sqlc.arg('sort_by')::TEXT = 'start_time' AND sqlc.arg('order')::TEXT != 'asc' THEN start_time END DESC,
  CASE WHEN sqlc.arg('sort_by')::TEXT = 'end_time' AND sqlc.arg('order')::TEXT = 'asc' THEN end_time END ASC,
  CASE WHEN sqlc.arg('sort_by')::TEXT = 'end_time' AND sqlc.arg('order')::TEXT != 'asc' THEN end_time END DESC,
  CASE WHEN sqlc.arg('sort_by')::TEXT = 'status' AND sqlc.arg('order')::TEXT = 'asc' THEN status END ASC,
  CASE WHEN sqlc.arg('sort_by')::TEXT = 'status' AND sqlc.arg('order')::TEXT != 'asc' THEN status END DESC
LIMIT sqlc.arg('limit')::int
OFFSET sqlc.arg('offset')::int;

-- name: CountJobAuditLogsByJobName :one
SELECT COUNT(*) 
FROM cron_job_audit_logs
WHERE job_name = sqlc.arg('job_name')::text
  AND tenant_id = sqlc.arg('tenant_id')::text;

-- name: CleanupStaleRegisteredJobs :execresult
DELETE FROM cron_registered_jobs
WHERE last_registered_at < NOW() - INTERVAL '24 hours'
  AND tenant_id = sqlc.arg('tenant_id')::text;

-- name: DeleteRegisteredJob :exec
DELETE FROM cron_registered_jobs
WHERE job_name = sqlc.arg('job_name')::text
  AND tenant_id = sqlc.arg('tenant_id')::text;
