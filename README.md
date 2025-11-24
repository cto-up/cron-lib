## Scheduler

The scheduler is a background task scheduler that uses cron expressions to schedule tasks.

It aims:

- to track registered jobs, job status and job history
- to avoid running the same job in a multi-server environment

It uses "github.com/robfig/cron/v3"

It is globally available and can be used by any module and any tenant.

To use the scheduler, you need to create a new scheduler instance and add tasks to it.

For Global Tenant use tenantID = ""

```go


package commoncron

import (
	"context"
	"fmt"
	"time"

	hubcron "github.com/cto-up/cron-lib/pkg"
	"github.com/cto-up/cron-lib/pkg/utils"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// ScheduledEchoJob handles scheduled LinkedIn posts
type ScheduledEchoJob struct {
	connPool       *pgxpool.Pool
	tenantID        string
	nextRunTime     time.Time
	msg 		  string
}

// Name implements cron.Job
func (j *ScheduledEchoJob) Name() string {
	return "scheduled_echos"
}

// Lock implements cron.Job. This prevents 2 processes to run at the same job at the same time
func (j *ScheduledEchoJob) Lock() string {
	return fmt.Sprintf("scheduled_echos_%s_%s",  j.tenantID, j.msg)
}

// TenantID implements cron.Job
func (j *ScheduledEchoJob) TenantID() string {
	return j.tenantID
}

// Schedule implements cron.Job - run every minute to check for scheduled posts
func (j *ScheduledEchoJob) Schedule() string {
	return "0 * * * * *" // Every minute
}

// IsLongRunning implements cron.Job
func (j *ScheduledEchoJob) IsLongRunning() bool {
	return false
}

// NextRunTime implements cron.Job
func (j *ScheduledEchoJob) NextRunTime() time.Time {
	nextTime, err := utils.NextRunTime(j.Schedule())
	if err != nil {
		log.Error().
			Str("tenant_id", j.tenantID).
			Err(err).
			Msg("Failed to parse cron schedule for scheduled posts job")
		return time.Time{}
	}
	j.nextRunTime = nextTime
	return j.nextRunTime
}

// Run implements cron.Job
func (j *ScheduledEchoJob) Run(ctx context.Context) error {
	log.Info().

		Str("tenant_id", j.tenantID).
		Str("msg", j.msg).
		Msg("Running scheduled echos job")

	return nil
}


// NewScheduledEchoJob creates a new scheduled post job
func NewScheduledEchoJob(connPool *pgxpool.Pool, msg, tenantID string) *ScheduledEchoJob {

	return &ScheduledEchoJob{
		connPool:           connPool,
		msg: 		  msg,
		tenantID:        tenantID,
	}
}

// EchoScheduler manages scheduled post jobs
type EchoScheduler struct {
	connPool       *pgxpool.Pool
	scheduler       *hubcron.JobManager
}

// NewEchoScheduler creates a new post scheduler
func NewEchoScheduler(connPool *pgxpool.Pool) *EchoScheduler {
	scheduler := hubcron.InitJobManager(context.Background(), connPool)

	return &EchoScheduler{
		connPool:           connPool,
		scheduler:       scheduler,
	}
}

// AddTenantEchoJob adds a scheduled post job for a tenant
func (ps *EchoScheduler) AddTenantEchoJob(ctx context.Context, msg string, tenantID string) {
	job := NewScheduledEchoJob(ps.connPool, msg, tenantID)
	ps.scheduler.RegisterJob(job)
	log.Info().Str("tenant_id", tenantID).Msg("Registered scheduled echo job for tenant")
}

// RemoveTenantEchoJob removes a scheduled post job for a tenant
func (ps *EchoScheduler) RemoveTenantEchoJob(ctx context.Context, tenantID string) {
	ps.scheduler.UnregisterJob("scheduled_echos", tenantID)
	log.Info().Str("tenant_id", tenantID).Msg("Unregistered scheduled echo job for tenant")
}

```

The scheduler will then run the task daily. You can also add tasks with interval-based scheduling:

Task name must be unique. If you try to add a task with the same name, it will not be added.
