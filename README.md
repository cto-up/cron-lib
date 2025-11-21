## Scheduler

The scheduler is a background task scheduler that uses cron expressions to schedule tasks.

It uses "github.com/robfig/cron/v3"

It is globally available and can be used by any module and any tenant.

To use the scheduler, you need to create a new scheduler instance and add tasks to it.

```go


// ScheduledPostJob handles scheduled LinkedIn posts
type ScheduledPostJob struct {
	store           *db.Store
	linkedInService *service.LinkedInService
	tenantID        string
	nextRunTime     time.Time
	cronParser      *cron.Parser
}

// Name implements cron.Job
func (j *ScheduledPostJob) Name() string {
	return "scheduled_linkedin_posts"
}

// Lock implements cron.Job. This prevents 2 processes to run at the same job at the same time
func (j *ScheduledPostJob) Lock() string {
	return fmt.Sprintf("scheduled_posts_%s", j.tenantID)
}

// TenantID implements cron.Job
func (j *ScheduledPostJob) TenantID() string {
	return j.tenantID
}

// Schedule implements cron.Job - run every minute to check for scheduled posts
func (j *ScheduledPostJob) Schedule() string {
	return "0 * * * * *" // Every minute
}

// IsLongRunning implements cron.Job
func (j *ScheduledPostJob) IsLongRunning() bool {
	return false
}

// NextRunTime implements cron.Job
func (j *ScheduledPostJob) NextRunTime() time.Time {
	if j.nextRunTime.IsZero() {
		j.nextRunTime = time.Now().Add(1 * time.Minute)
	}
	return j.nextRunTime
}

// Run implements cron.Job
func (j *ScheduledPostJob) Run(ctx context.Context) error {
	log.Info().
		Str("tenant_id", j.tenantID).
		Msg("Running scheduled posts job")

	// Update next run time
	j.nextRunTime = time.Now().Add(1 * time.Minute)

	// Get posts that are scheduled and due
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	posts, err := j.store.ListScheduledPosts(ctx, repository.ListScheduledPostsParams{
		ScheduledAt: now,
		UserID:      "", // Get all users' posts for this tenant
		Limit:       100,
		TenantID:    j.tenantID,
	})
	if err != nil {
		log.Error().Err(err).Str("tenant_id", j.tenantID).Msg("Failed to get scheduled posts")
		return err
	}

	if len(posts) == 0 {
		log.Debug().Str("tenant_id", j.tenantID).Msg("No scheduled posts found")
		return nil
	}

	log.Info().
		Int("post_count", len(posts)).
		Str("tenant_id", j.tenantID).
		Msg("Processing scheduled posts")

	successCount := 0
	errorCount := 0

	for _, post := range posts {
		if err := j.processScheduledPost(ctx, post); err != nil {
			log.Error().
				Err(err).
				Str("post_id", post.ID.String()).
				Str("user_id", post.UserID).
				Msg("Failed to process scheduled post")
			errorCount++
		} else {
			successCount++
		}
	}

	log.Info().
		Int("success_count", successCount).
		Int("error_count", errorCount).
		Str("tenant_id", j.tenantID).
		Msg("Completed scheduled posts processing")

	return nil
}

// processScheduledPost processes a single scheduled post
func (j *ScheduledPostJob) processScheduledPost(ctx context.Context, post repository.SociPost) error {
	// Implement the execution
	return nil
}

// NewScheduledPostJob creates a new scheduled post job
func NewScheduledPostJob(store *db.Store, linkedInService *service.LinkedInService, tenantID string) *ScheduledPostJob {
	cronParser := cron.NewParser(
		cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
	)
	return &ScheduledPostJob{
		store:           store,
		linkedInService: linkedInService,
		tenantID:        tenantID,
		cronParser:      &cronParser,
	}
}

// PostScheduler manages scheduled post jobs
type PostScheduler struct {
	store           *db.Store
	linkedInService *service.LinkedInService
	scheduler       *hubcron.JobManager
}

// NewPostScheduler creates a new post scheduler
func NewPostScheduler(store *db.Store, linkedInService *service.LinkedInService) *PostScheduler {
	scheduler := hubcron.InitJobManager(context.Background(), store.ConnPool)

	return &PostScheduler{
		store:           store,
		linkedInService: linkedInService,
		scheduler:       scheduler,
	}
}

// AddTenantPostJob adds a scheduled post job for a tenant
func (ps *PostScheduler) AddTenantPostJob(ctx context.Context, tenantID string) {
	job := NewScheduledPostJob(ps.store, ps.linkedInService, tenantID)
	ps.scheduler.RegisterJob(job)
	log.Info().Str("tenant_id", tenantID).Msg("Registered scheduled post job for tenant")
}

// RemoveTenantPostJob removes a scheduled post job for a tenant
func (ps *PostScheduler) RemoveTenantPostJob(ctx context.Context, tenantID string) {
	ps.scheduler.UnregisterJob("scheduled_linkedin_posts", tenantID)
	log.Info().Str("tenant_id", tenantID).Msg("Unregistered scheduled post job for tenant")
}
```

```go
// do it once
scheduler := cron.InitJobManager(ctx, connPool)

// jon has a name and tenantID
scheduler.RegisterJob(job)


// Unregister using job.Name() and job.TenantID()
scheduler.UnregisterJob("scheduled_linkedin_posts", tenantID)

```

The scheduler will then run the task daily. You can also add tasks with interval-based scheduling:

Task name must be unique. If you try to add a task with the same name, it will not be added.
