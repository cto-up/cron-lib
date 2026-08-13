package cron

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cto-up/cron-lib/pkg/db/migration"
)

// testAppName scopes the pg_locks assertions to this test's own backends, so a
// concurrently running app on the same database cannot influence the result.
const testAppName = "cron_lib_job_manager_test"

// testJob is a minimal Job whose Run body is supplied per test.
type testJob struct {
	name        string
	tenantID    string
	longRunning bool
	run         func(ctx context.Context) error
}

func (j *testJob) Name() string                  { return j.name }
func (j *testJob) Lock() string                  { return j.name }
func (j *testJob) TenantID() string              { return j.tenantID }
func (j *testJob) Schedule() string              { return "0 */5 * * * *" }
func (j *testJob) NextRunTime() time.Time        { return time.Now().Add(5 * time.Minute) }
func (j *testJob) IsLongRunning() bool           { return j.longRunning }
func (j *testJob) Run(ctx context.Context) error { return j.run(ctx) }

// newTestManager brings up an isolated schema with the real migrations applied
// and returns a JobManager wired to a pool over it. Skips when no database is
// configured, so `go test ./...` still passes on a machine without Postgres.
func newTestManager(t *testing.T, ctx context.Context, schema string) *JobManager {
	t.Helper()

	dsn := os.Getenv("CRON_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("CRON_TEST_DATABASE_URL not set; skipping database-backed test")
	}

	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Skipf("cannot reach CRON_TEST_DATABASE_URL: %v", err)
	}
	defer admin.Close(ctx)

	if _, err := admin.Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE; CREATE SCHEMA %s", schema, schema)); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.ConnConfig.RuntimeParams["application_name"] = testAppName
	// More than one connection, always warm: this is what makes a session-scoped
	// lock unreleasable through the pool, and so what the regression needs.
	cfg.MinConns = 4
	cfg.MaxConns = 8

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		conn, err := pgx.Connect(cleanupCtx, dsn)
		if err != nil {
			return
		}
		defer conn.Close(cleanupCtx)
		conn.Exec(cleanupCtx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema))
	})

	applyMigrations(t, ctx, pool)

	return newJobManager(ctx, pool)
}

// applyMigrations runs the goose "Up" half of each embedded migration. cron-lib
// does not depend on goose, so the directives are stripped here rather than
// duplicating the schema in the test.
func applyMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	entries, err := migration.FS.ReadDir(".")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		body, err := migration.FS.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}

		sql := string(body)
		if i := strings.Index(sql, "-- +goose Down"); i >= 0 {
			sql = sql[:i]
		}
		sql = strings.NewReplacer(
			"-- +goose Up", "",
			"-- +goose StatementBegin", "",
			"-- +goose StatementEnd", "",
		).Replace(sql)

		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}
}

// advisoryLockCount reports advisory locks held by this test's own connections.
func advisoryLockCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int {
	t.Helper()

	var n int
	err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM pg_locks l
		JOIN pg_stat_activity a USING (pid)
		WHERE l.locktype = 'advisory' AND a.application_name = $1`, testAppName).Scan(&n)
	if err != nil {
		t.Fatalf("count advisory locks: %v", err)
	}
	return n
}

func auditStatuses(t *testing.T, ctx context.Context, pool *pgxpool.Pool, jobName string) []string {
	t.Helper()

	rows, err := pool.Query(ctx,
		`SELECT status FROM cron_job_audit_logs WHERE job_name = $1 ORDER BY start_time`, jobName)
	if err != nil {
		t.Fatalf("query audit logs: %v", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatalf("scan audit log: %v", err)
		}
		out = append(out, s)
	}
	return out
}

// Regression for the advisory-lock leak: pg_try_advisory_lock binds to the
// connection that ran it, so acquiring it through a pool left it held forever
// whenever the deferred pg_advisory_unlock was handed a different connection —
// which pg_advisory_unlock reports by returning false, not by erroring. Every
// later tick then recorded "Job already running in another instance" and the
// job never ran again.
//
// Contending pool traffic runs throughout, which is what forces the release
// onto a foreign connection.
func TestExecuteJobWithLockSurvivesRepeatedRunsUnderPoolContention(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jm := newTestManager(t, ctx, "cron_lib_test_contention")

	noise, stopNoise := context.WithCancel(ctx)
	var noiseWG sync.WaitGroup
	for i := 0; i < 4; i++ {
		noiseWG.Add(1)
		go func() {
			defer noiseWG.Done()
			for noise.Err() == nil {
				conn, err := jm.store.ConnPool.Acquire(noise)
				if err != nil {
					return
				}
				conn.Exec(noise, "SELECT 1")
				conn.Release()
			}
		}()
	}
	defer func() {
		stopNoise()
		noiseWG.Wait()
	}()

	const runs = 10
	job := &testJob{
		name:     "test.contention",
		tenantID: "tenant-a",
		run:      func(ctx context.Context) error { return nil },
	}

	for i := 0; i < runs; i++ {
		jm.executeJobWithLock(job)
	}

	stopNoise()
	noiseWG.Wait()

	statuses := auditStatuses(t, ctx, jm.store.ConnPool, job.Name())
	if len(statuses) != runs {
		t.Fatalf("expected %d audit entries, got %d: %v", runs, len(statuses), statuses)
	}
	for i, s := range statuses {
		if s != "completed" {
			t.Errorf("run %d: status = %q, want \"completed\" (all %v)", i+1, s, statuses)
		}
	}

	if n := advisoryLockCount(t, ctx, jm.store.ConnPool); n != 0 {
		t.Errorf("%d advisory lock(s) still held by the pool; the scheduler leaks session locks", n)
	}
}

// A run that overran the staleness window has already lost its lease to another
// instance. Its status write used to match on job id alone, so it would clear
// locked_by and stamp 'completed' over the run that superseded it — erasing the
// live lease and letting a third instance start yet another copy.
//
// Run steals the lease here, standing in for the takeover that a >10 minute
// overrun would cause.
func TestExecuteJobWithLockDoesNotClobberSupersededRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jm := newTestManager(t, ctx, "cron_lib_test_fence")

	job := &testJob{
		name:     "test.superseded",
		tenantID: "tenant-a",
		run: func(ctx context.Context) error {
			_, err := jm.store.ConnPool.Exec(ctx,
				`UPDATE cron_jobs SET locked_by = 'other-instance' WHERE lock = $1 AND tenant_id = $2`,
				"test.superseded", "tenant-a")
			return err
		},
	}

	jm.executeJobWithLock(job)

	var status string
	var lockedBy *string
	err := jm.store.ConnPool.QueryRow(ctx,
		`SELECT status, locked_by FROM cron_jobs WHERE lock = $1 AND tenant_id = $2`,
		job.Lock(), job.TenantID()).Scan(&status, &lockedBy)
	if err != nil {
		t.Fatalf("read cron_jobs row: %v", err)
	}

	if status != "running" {
		t.Errorf("status = %q, want \"running\"; the superseded run overwrote the live lease", status)
	}
	if lockedBy == nil {
		t.Errorf("locked_by = NULL, want \"other-instance\"; the superseded run released someone else's lease")
	} else if *lockedBy != "other-instance" {
		t.Errorf("locked_by = %q, want \"other-instance\"", *lockedBy)
	}
}

// The status write must not inherit a context that the job itself can outlive.
// It used to reuse the 60s lock-acquisition context derived from jm.context, so
// a job running longer than that deadline — or any run overlapping shutdown —
// failed to clear the row, stranding it at 'running' until the stale-lock sweep
// 10 minutes later.
//
// Cancelling jm.context from inside Run reproduces that without a slow test.
func TestExecuteJobWithLockRecordsOutcomeAfterParentContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jm := newTestManager(t, ctx, "cron_lib_test_ctx")

	job := &testJob{
		name:     "test.outlives-context",
		tenantID: "tenant-a",
		run: func(ctx context.Context) error {
			cancel() // the job outlives the context its lock was acquired under
			return nil
		},
	}

	jm.executeJobWithLock(job)

	readCtx, readCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer readCancel()

	var status string
	var lockedAt *time.Time
	err := jm.store.ConnPool.QueryRow(readCtx,
		`SELECT status, locked_at FROM cron_jobs WHERE lock = $1 AND tenant_id = $2`,
		job.Lock(), job.TenantID()).Scan(&status, &lockedAt)
	if err != nil {
		t.Fatalf("read cron_jobs row: %v", err)
	}

	if status != "completed" {
		t.Errorf("status = %q, want \"completed\"; the row is stranded and blocks the next tick", status)
	}
	if lockedAt != nil {
		t.Errorf("locked_at = %v, want NULL; the lock was never released", *lockedAt)
	}
}
