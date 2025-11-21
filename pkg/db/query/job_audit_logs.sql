-- name: GetJobAuditLogByID :one
SELECT * FROM cron_job_audit_logs
WHERE id = $1 AND tenant_id = sqlc.arg('tenant_id')::text LIMIT 1;

-- name: ListJobAuditLogs :many
SELECT * FROM cron_job_audit_logs
WHERE tenant_id = sqlc.arg('tenant_id')::text
  AND (UPPER(job_name) LIKE UPPER(sqlc.narg('like')) OR sqlc.narg('like') IS NULL)
ORDER BY
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'app_id' AND sqlc.arg('order')::TEXT = 'asc' THEN app_id END ASC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'app_id' AND sqlc.arg('order')::TEXT != 'asc' THEN app_id END DESC
  ,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'request_id' AND sqlc.arg('order')::TEXT = 'asc' THEN request_id END ASC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'request_id' AND sqlc.arg('order')::TEXT != 'asc' THEN request_id END DESC
  ,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'job_name' AND sqlc.arg('order')::TEXT = 'asc' THEN job_name END ASC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'job_name' AND sqlc.arg('order')::TEXT != 'asc' THEN job_name END DESC
  ,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'scheduled_time' AND sqlc.arg('order')::TEXT = 'asc' THEN scheduled_time END ASC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'scheduled_time' AND sqlc.arg('order')::TEXT != 'asc' THEN scheduled_time END DESC
  ,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'start_time' AND sqlc.arg('order')::TEXT = 'asc' THEN start_time END ASC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'start_time' AND sqlc.arg('order')::TEXT != 'asc' THEN start_time END DESC
  ,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'end_time' AND sqlc.arg('order')::TEXT = 'asc' THEN end_time END ASC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'end_time' AND sqlc.arg('order')::TEXT != 'asc' THEN end_time END DESC
  ,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'status' AND sqlc.arg('order')::TEXT = 'asc' THEN status END ASC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'status' AND sqlc.arg('order')::TEXT != 'asc' THEN status END DESC
  ,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'output' AND sqlc.arg('order')::TEXT = 'asc' THEN output END ASC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'output' AND sqlc.arg('order')::TEXT != 'asc' THEN output END DESC
  ,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'error' AND sqlc.arg('order')::TEXT = 'asc' THEN error END ASC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'error' AND sqlc.arg('order')::TEXT != 'asc' THEN error END DESC
  
LIMIT $1
OFFSET $2;

-- name: CreateJobAuditLog :one
INSERT INTO cron_job_audit_logs (
  user_id, tenant_id, "app_id", "request_id", "job_name", "scheduled_time", "start_time", "end_time", "status", "output", "error"
) VALUES (
  $1, sqlc.arg('tenant_id')::text, 
  $2, 
  $3, 
  $4, 
  $5, 
  $6, 
  $7, 
  $8, 
  $9, 
  $10
)
RETURNING *;

-- name: UpdateJobAuditLog :one
UPDATE cron_job_audit_logs 
SET 
    "end_time" =  COALESCE(sqlc.narg('end_time'), end_time),
    "status" =  sqlc.arg('status'),
    "output" =  COALESCE(sqlc.narg('output'), output),
    "error" =  COALESCE(sqlc.narg('error'), error)

WHERE id = $1 AND tenant_id = sqlc.arg('tenant_id')::text
RETURNING *;

-- name: DeleteJobAuditLog :one
DELETE FROM cron_job_audit_logs
WHERE id = $1 and tenant_id = sqlc.arg('tenant_id')::text
RETURNING id
;
