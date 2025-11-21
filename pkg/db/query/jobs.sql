-- name: GetJobByID :one
SELECT * FROM cron_jobs
WHERE id = $1 AND tenant_id = sqlc.arg('tenant_id')::text LIMIT 1;

-- name: ListJobs :many
SELECT * FROM cron_jobs
WHERE tenant_id = sqlc.arg('tenant_id')::text
  AND (UPPER(job_name) LIKE UPPER(sqlc.narg('like')) OR sqlc.narg('like') IS NULL)
ORDER BY
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'lock' AND sqlc.arg('order')::TEXT = 'asc' THEN lock END ASC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'lock' AND sqlc.arg('order')::TEXT != 'asc' THEN lock END DESC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'job_name' AND sqlc.arg('order')::TEXT = 'asc' THEN job_name END ASC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'job_name' AND sqlc.arg('order')::TEXT != 'asc' THEN job_name END DESC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'status' AND sqlc.arg('order')::TEXT = 'asc' THEN status END ASC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'status' AND sqlc.arg('order')::TEXT != 'asc' THEN status END DESC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'last_execution_time' AND sqlc.arg('order')::TEXT = 'asc' THEN last_execution_time END ASC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'last_execution_time' AND sqlc.arg('order')::TEXT != 'asc' THEN last_execution_time END DESC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'next_execution_time' AND sqlc.arg('order')::TEXT = 'asc' THEN next_execution_time END ASC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'next_execution_time' AND sqlc.arg('order')::TEXT != 'asc' THEN next_execution_time END DESC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'locked_by' AND sqlc.arg('order')::TEXT = 'asc' THEN locked_by END ASC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'locked_by' AND sqlc.arg('order')::TEXT != 'asc' THEN locked_by END DESC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'locked_at' AND sqlc.arg('order')::TEXT = 'asc' THEN locked_at END ASC,
  CASE WHEN sqlc.arg('sortBy')::TEXT = 'locked_at' AND sqlc.arg('order')::TEXT != 'asc' THEN locked_at END DESC
LIMIT $1
OFFSET $2;

-- name: CreateJob :one
INSERT INTO cron_jobs (
  "lock", "job_name", tenant_id, "status", "last_execution_time", "next_execution_time", "locked_by", "locked_at"
) VALUES (
  $1, $2, sqlc.arg('tenant_id')::text, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: UpdateJob :one
UPDATE cron_jobs 
SET "lock" = sqlc.arg('lock'),
    "job_name" = sqlc.arg('job_name'),
    "status" = sqlc.arg('status'),
    "last_execution_time" = COALESCE(sqlc.narg('last_execution_time')::timestamptz, last_execution_time),
    "next_execution_time" = COALESCE(sqlc.narg('next_execution_time')::timestamptz, next_execution_time),
    "locked_by" = COALESCE(sqlc.narg('locked_by'), locked_by),
    "locked_at" = COALESCE(sqlc.narg('locked_at')::timestamptz, locked_at),
    updated_at = NOW()
WHERE id = $1 AND tenant_id = sqlc.arg('tenant_id')::text
RETURNING *;

-- name: DeleteJob :one
DELETE FROM cron_jobs
WHERE id = $1 and tenant_id = sqlc.arg('tenant_id')::text
RETURNING id;

-- name: ReleaseTaskLock :exec
UPDATE cron_jobs
SET status = 'completed', 
    updated_at = NOW(),
    locked_by = NULL,
    locked_at = NULL
WHERE id = sqlc.arg('id')::uuid 
  AND tenant_id = sqlc.arg('tenant_id')::text;

-- name: CleanupOldTasks :execresult
DELETE FROM cron_jobs
WHERE status IN ('completed', 'failed') 
  AND updated_at < NOW() - INTERVAL '7 days'
  AND tenant_id = sqlc.arg('tenant_id')::text;

-- name: TryAdvisoryLock :one
SELECT pg_try_advisory_lock(sqlc.arg('lock_id')::bigint) as lock_acquired;

-- name: ReleaseAdvisoryLock :exec
SELECT pg_advisory_unlock(sqlc.arg('lock_id')::bigint);

-- name: UpdateJobStatusToFailed :exec
UPDATE cron_jobs 
SET status = 'failed', 
    updated_at = NOW(),
    locked_by = NULL,
    locked_at = NULL
WHERE id = sqlc.arg('job_id')::uuid;

-- name: UpdateJobStatusToCompleted :exec
UPDATE cron_jobs 
SET status = 'completed', 
    updated_at = NOW(),
    locked_by = NULL,
    locked_at = NULL
WHERE id = sqlc.arg('job_id')::uuid;

-- name: AcquireJobLockInDB :one
INSERT INTO cron_jobs (
  tenant_id, "lock", "job_name", "status", "last_execution_time", "next_execution_time", 
  "locked_by", "locked_at"
) VALUES (
  sqlc.arg('tenant_id')::text,
  sqlc.arg('lock')::text,
  sqlc.arg('job_name')::text,
  'running',
  sqlc.arg('now')::timestamptz,
  sqlc.arg('next_run_time')::timestamptz,
  sqlc.arg('instance_id')::text,
  sqlc.arg('now')::timestamptz
)
ON CONFLICT (tenant_id, lock) DO UPDATE
SET 
  status = 'running',
  last_execution_time = sqlc.arg('now')::timestamptz,
  next_execution_time = sqlc.arg('next_run_time')::timestamptz,
  locked_by = sqlc.arg('instance_id')::text,
  locked_at = sqlc.arg('now')::timestamptz,
  updated_at = sqlc.arg('now')::timestamptz
WHERE cron_jobs.locked_at IS NULL 
   OR cron_jobs.locked_at < sqlc.arg('now')::timestamptz - INTERVAL '10 minutes'
   OR cron_jobs.status != 'running'
RETURNING id;

-- NEW: Clean up stale locks from crashed instances
-- name: CleanupStaleLocks :execresult
UPDATE cron_jobs 
SET status = 'failed',
    locked_by = NULL,
    locked_at = NULL,
    updated_at = NOW()
WHERE status = 'running' 
  AND locked_at < NOW() - INTERVAL '15 minutes'
  AND tenant_id = sqlc.arg('tenant_id')::text;

-- NEW: Get jobs that are potentially stuck
-- name: GetStaleJobs :many
SELECT id, "lock", job_name, tenant_id, locked_by, locked_at, status
FROM cron_jobs
WHERE status = 'running' 
  AND locked_at < NOW() - INTERVAL '15 minutes'
  AND tenant_id = sqlc.arg('tenant_id')::text;

-- NEW: Force unlock a specific job (for admin operations)
-- name: ForceUnlockJob :exec
UPDATE cron_jobs
SET status = 'failed',
    locked_by = NULL,
    locked_at = NULL,
    updated_at = NOW()
WHERE id = sqlc.arg('job_id')::uuid
  AND tenant_id = sqlc.arg('tenant_id')::text;

-- NEW: Check if a job is currently locked by any instance
-- name: IsJobLocked :one
SELECT 
  id,
  CASE 
    WHEN status = 'running' AND locked_at > NOW() - INTERVAL '10 minutes' THEN true
    ELSE false
  END as is_locked,
  locked_by,
  locked_at
FROM cron_jobs
WHERE tenant_id = sqlc.arg('tenant_id')::text 
  AND lock = sqlc.arg('lock')::text
LIMIT 1;

-- NEW: Heartbeat update for long-running jobs
-- name: UpdateJobHeartbeat :exec
UPDATE cron_jobs
SET locked_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg('job_id')::uuid
  AND locked_by = sqlc.arg('instance_id')::text
  AND status = 'running';