package cron

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"

	"github.com/cto-up/cron-lib/pkg/db"
	"github.com/cto-up/cron-lib/pkg/db/repository"
)

// Job defines the interface that all scheduled jobs must implement
type Job interface {
	Name() string

	// Lock returns a unique identifier for the job
	Lock() string

	// TenantID returns the tenant ID for the job
	TenantID() string

	// Schedule returns the cron schedule for the job (e.g., "0 * * * * *" for every minute)
	Schedule() string

	// Run executes the job and returns an error if it fails
	Run(ctx context.Context) error

	// NextRunTime returns when this job should run next (optional)
	// This is used for tracking in the cron_jobs table
	NextRunTime() time.Time

	// IsLongRunning returns true if this job typically runs longer than 5 minutes
	// This enables heartbeat updates during execution
	IsLongRunning() bool
}

type JobManager struct {
	cron          *cron.Cron
	jobs          []Job
	entryIDs      map[string]cron.EntryID // Map job identifiers to cron entry IDs
	mutex         sync.Mutex              // Protect concurrent access to jobs and entryIDs
	context       context.Context
	store         *db.Store
	instanceID    string        // Unique identifier for this application instance
	isRunning     bool          // Track if scheduler is running
	cleanupTicker *time.Ticker  // For periodic cleanup
	stopCleanup   chan struct{} // Signal to stop cleanup routine
}

// Singleton instance and mutex for thread-safe initialization
var (
	instance     *JobManager
	instanceOnce sync.Once
	instanceMu   sync.Mutex
)

// GetJobManager returns the singleton instance of JobManager
func GetJobManager() *JobManager {
	if instance == nil {
		log.Printf("Warning: JobManager singleton accessed before initialization")
	}
	return instance
}

// InitJobManager initializes the singleton JobManager instance
func InitJobManager(ctx context.Context, connPool *pgxpool.Pool) *JobManager {
	instanceMu.Lock()
	defer instanceMu.Unlock()

	instanceOnce.Do(func() {
		instance = newJobManager(ctx, connPool)
		log.Printf("JobManager singleton initialized with instance ID: %s", instance.instanceID)
	})

	return instance
}

// newJobManager creates a new JobManager instance (private constructor)
func newJobManager(ctx context.Context, connPool *pgxpool.Pool) *JobManager {
	instanceID := uuid.New().String() // Generate a unique ID for this instance
	return &JobManager{
		cron:       cron.New(cron.WithSeconds()),
		jobs:       []Job{},
		entryIDs:   make(map[string]cron.EntryID),
		context:    ctx,
		store:      db.NewStore(connPool, true),
		instanceID: instanceID,
		isRunning:  false,
	}
}

// RegisterJob adds a job to the job manager
func (jm *JobManager) RegisterJob(job Job) {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	// Check if job already exists
	key := jobKey(job.Name(), job.TenantID())
	for _, existingJob := range jm.jobs {
		if jobKey(existingJob.Name(), existingJob.TenantID()) == key {
			log.Printf("Job %s for tenant %s already registered", job.Name(), job.TenantID())
			return
		}
	}

	// Add job to slice
	jm.jobs = append(jm.jobs, job)

	// Register job in database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Extract job details
	jobName := job.Name()
	tenantID := job.TenantID()
	schedule := job.Schedule()
	isLongRunning := job.IsLongRunning()

	// Register job in database
	params := repository.UpsertRegisteredJobParams{
		JobName:       jobName,
		Schedule:      schedule,
		IsLongRunning: isLongRunning,
		IsEnabled:     true, // Default to enabled
		InstanceID:    jm.instanceID,
		TenantID:      tenantID,
	}

	_, err := jm.store.UpsertRegisteredJob(ctx, params)
	if err != nil {
		log.Printf("Error registering job %s (tenant %s) in database: %v", jobName, tenantID, err)
		// Continue even if registration fails
	}

	// If scheduler is already running, add job to cron
	if jm.isRunning {
		jm.scheduleJob(job)
	}
}

// scheduleJob adds a job to the cron scheduler
func (jm *JobManager) scheduleJob(job Job) {
	schedule := job.Schedule()
	entryID, err := jm.cron.AddFunc(schedule, func() {
		jm.executeJobWithLock(job)
	})

	if err != nil {
		log.Printf("Failed to schedule job %s: %v", job.Name(), err)
		return
	}

	// Store the entry ID for later removal if needed
	key := jobKey(job.Name(), job.TenantID())
	jm.entryIDs[key] = entryID
	log.Printf("Job %s for tenant %s scheduled with ID %v", job.Name(), job.TenantID(), entryID)
}

// UnregisterJob removes a job from the job manager
func (jm *JobManager) UnregisterJob(jobName string, tenantID string) {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	key := jobKey(jobName, tenantID)

	// Remove from running cron if scheduler is active
	if jm.isRunning {
		if entryID, exists := jm.entryIDs[key]; exists {
			jm.cron.Remove(entryID)
			delete(jm.entryIDs, key)
			log.Printf("Removed job %s for tenant %s from running scheduler", jobName, tenantID)
		}
	}

	// Remove from jobs slice
	for i, job := range jm.jobs {
		if job.Name() == jobName && job.TenantID() == tenantID {
			jm.jobs = append(jm.jobs[:i], jm.jobs[i+1:]...)
			log.Printf("Unregistered job %s for tenant %s", jobName, tenantID)
			break
		}
	}

	// Remove job from database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// We need to add a new query to delete a registered job
	err := jm.store.DeleteRegisteredJob(ctx, repository.DeleteRegisteredJobParams{
		JobName:  jobName,
		TenantID: tenantID,
	})

	if err != nil {
		log.Printf("Error removing job %s (tenant %s) from database: %v", jobName, tenantID, err)
		// Continue even if database removal fails
	} else {
		log.Printf("Removed job %s for tenant %s from database", jobName, tenantID)
	}
}

// StartScheduler starts the cron scheduler with concurrency control
func (jm *JobManager) StartScheduler() {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	// Don't start if already running
	if jm.isRunning {
		log.Printf("Scheduler is already running")
		return
	}

	// Schedule all jobs
	for _, job := range jm.jobs {
		jm.scheduleJob(job)
	}

	// Start the cron scheduler
	jm.cron.Start()
	jm.isRunning = true

	// **START CLEANUP ROUTINE HERE**
	jm.startCleanupRoutine()

	log.Printf("Scheduler started with %d jobs", len(jm.jobs))
}

// StopScheduler stops the cron scheduler
func (jm *JobManager) StopScheduler() {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	if !jm.isRunning {
		return
	}

	// Stop the cleanup routine first
	jm.stopCleanupRoutine()

	// Stop the scheduler and wait for running jobs
	ctx := jm.cron.Stop()

	// Create a timeout for the shutdown
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Wait for either the jobs to complete or the timeout to expire
	select {
	case <-ctx.Done():
		log.Printf("All cron jobs completed gracefully")
	case <-timeoutCtx.Done():
		log.Printf("Warning: Cron job shutdown timed out after 30 seconds. Some jobs may not have completed.")
	}

	// Clear entry IDs as they're no longer valid
	jm.entryIDs = make(map[string]cron.EntryID)
	jm.isRunning = false
	log.Printf("Scheduler stopped")
}

// **NEW: Start the cleanup routine**
func (jm *JobManager) startCleanupRoutine() {
	// Run cleanup every 5 minutes
	jm.cleanupTicker = time.NewTicker(5 * time.Minute)

	go func() {
		// Run initial cleanup on startup
		jm.cleanupStaleJobs()

		for {
			select {
			case <-jm.cleanupTicker.C:
				jm.cleanupStaleJobs()
			case <-jm.stopCleanup:
				return
			case <-jm.context.Done():
				return
			}
		}
	}()

	log.Printf("Cleanup routine started")
}

// **NEW: Stop the cleanup routine**
func (jm *JobManager) stopCleanupRoutine() {
	if jm.cleanupTicker != nil {
		jm.cleanupTicker.Stop()
		close(jm.stopCleanup)
		jm.stopCleanup = make(chan struct{}) // Reset for next start
		log.Printf("Cleanup routine stopped")
	}
}

// **NEW: Cleanup stale jobs periodically**
func (jm *JobManager) cleanupStaleJobs() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get unique tenant IDs from registered jobs
	tenantIDs := make(map[string]bool)
	jm.mutex.Lock()
	for _, job := range jm.jobs {
		tenantIDs[job.TenantID()] = true
	}
	jm.mutex.Unlock()

	// Clean up stale locks for each tenant
	totalCleaned := int64(0)
	for tenantID := range tenantIDs {
		result, err := jm.store.CleanupStaleLocks(ctx, tenantID)
		if err != nil {
			log.Printf("Error cleaning up stale locks for tenant %s: %v", tenantID, err)
			continue
		}

		if rowsAffected := result.RowsAffected(); rowsAffected > 0 {
			totalCleaned += rowsAffected
			log.Printf("Cleaned up %d stale job locks for tenant %s", rowsAffected, tenantID)
		}
	}

	if totalCleaned > 0 {
		log.Printf("Total stale locks cleaned up: %d", totalCleaned)
	}
}

// **NEW: Update job heartbeat for long-running jobs**
func (jm *JobManager) updateJobHeartbeat(jobID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := jm.store.UpdateJobHeartbeat(ctx, repository.UpdateJobHeartbeatParams{
		JobID:      jobID,
		InstanceID: jm.instanceID,
	})
	if err != nil {
		log.Printf("Error updating job heartbeat for job %s: %v", jobID, err)
	}
}

// **NEW: Start heartbeat routine for long-running jobs**
func (jm *JobManager) startHeartbeat(jobID uuid.UUID, stopChan <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Minute) // Update every 2 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			jm.updateJobHeartbeat(jobID)
		case <-stopChan:
			return
		case <-jm.context.Done():
			return
		}
	}
}

// executeJobWithLock handles the concurrency control logic
func (jm *JobManager) executeJobWithLock(job Job) {
	jobName := job.Name()
	lock := job.Lock()
	tenantID := job.TenantID()

	// Generate a request ID for tracking this job execution
	requestID := uuid.New().String()

	// Create initial audit log entry
	auditCtx, auditCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer auditCancel()

	now := time.Now()
	scheduledTime := pgtype.Timestamp{Time: now, Valid: true}
	startTime := pgtype.Timestamp{Time: now, Valid: true}

	// Create audit log with "started" status
	auditParams := repository.CreateJobAuditLogParams{
		UserID:        "system",
		AppID:         jm.instanceID,
		RequestID:     requestID,
		JobName:       jobName,
		ScheduledTime: scheduledTime,
		StartTime:     startTime,
		Status:        "started",
		TenantID:      tenantID,
	}

	auditLog, err := jm.store.CreateJobAuditLog(auditCtx, auditParams)
	if err != nil {
		log.Printf("Error creating audit log for job %s (tenant %s): %v", jobName, tenantID, err)
		// Continue execution even if audit logging fails
	}

	// Use PostgreSQL advisory lock to prevent concurrent execution
	lockID := int64(jobLockToLockID(lock, tenantID))

	// Try to acquire an advisory lock with timeout
	ctx, cancel := context.WithTimeout(jm.context, 60*time.Second)
	defer cancel()

	// Try to acquire advisory lock using sqlc
	lockResult, err := jm.store.TryAdvisoryLock(ctx, lockID)
	if err != nil {
		log.Printf("Error acquiring lock for job %s (tenant %s): %v", jobName, tenantID, err)
		errorMsg := err.Error()
		jm.updateAuditLogStatus(auditLog.ID, "failed", nil, &errorMsg, tenantID)
		return
	}

	if !lockResult {
		log.Printf("Job %s for tenant %s is already running in another instance", jobName, tenantID)
		errorMsg := "Job already running in another instance"
		jm.updateAuditLogStatus(auditLog.ID, "skipped", nil, &errorMsg, tenantID)
		return
	}

	// Ensure advisory lock is released even in case of panic
	defer func() {
		// Release advisory lock using sqlc
		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer releaseCancel()

		err := jm.store.ReleaseAdvisoryLock(releaseCtx, lockID)
		if err != nil {
			log.Printf("Error releasing lock for job %s (tenant %s): %v", jobName, tenantID, err)
		}
	}()

	// Try to acquire the job lock in the database
	nextRunTime := job.NextRunTime()

	// Acquire job lock in DB using sqlc
	acquireParams := repository.AcquireJobLockInDBParams{
		TenantID:    tenantID,
		Lock:        lock,
		JobName:     job.Name(),
		Now:         now,
		NextRunTime: nextRunTime,
		InstanceID:  jm.instanceID,
	}

	jobID, err := jm.store.AcquireJobLockInDB(ctx, acquireParams)
	if err != nil {
		// Check for "no rows" error which indicates the ON CONFLICT WHERE clause wasn't satisfied
		if pgx.ErrNoRows.Error() == err.Error() {
			log.Printf("Job %s for tenant %s is already locked by another instance", jobName, tenantID)
			errorMsg := "Job already locked in database"
			jm.updateAuditLogStatus(auditLog.ID, "skipped", nil, &errorMsg, tenantID)
		} else {
			log.Printf("Database error acquiring lock for job %s (tenant %s): %v", jobName, tenantID, err)
			errorMsg := err.Error()
			jm.updateAuditLogStatus(auditLog.ID, "failed", nil, &errorMsg, tenantID)
		}
		return
	}

	// **START HEARTBEAT FOR LONG-RUNNING JOBS**
	var heartbeatStop chan struct{}
	if job.IsLongRunning() {
		heartbeatStop = make(chan struct{})
		go jm.startHeartbeat(jobID, heartbeatStop)
		log.Printf("Started heartbeat for long-running job %s (tenant %s)", jobName, tenantID)
	}

	// **STOP HEARTBEAT ON COMPLETION**
	defer func() {
		if heartbeatStop != nil {
			close(heartbeatStop)
		}
	}()

	// Add panic recovery to ensure job status is updated even if job panics
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in job %s (tenant %s): %v", jobName, tenantID, r)

			// Update status to failed using sqlc
			updateCtx, updateCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer updateCancel()

			err = jm.store.UpdateJobStatusToFailed(updateCtx, jobID)
			if err != nil {
				log.Printf("Error updating job status to failed after panic: %v", err)
			}

			// Update audit log with panic information
			errorMsg := fmt.Sprintf("Panic: %v", r)
			jm.updateAuditLogStatus(auditLog.ID, "failed", nil, &errorMsg, tenantID)
		}
	}()

	// Execute the job
	var output string
	var jobErr error

	if jobErr = job.Run(jm.context); jobErr != nil {
		log.Printf("Error in job %s (tenant %s): %v", jobName, tenantID, jobErr)

		// Update status to failed using sqlc
		err = jm.store.UpdateJobStatusToFailed(ctx, jobID)
		if err != nil {
			log.Printf("Error updating job status to failed: %v", err)
		}

		// Update audit log with error information
		errorMsg := jobErr.Error()
		jm.updateAuditLogStatus(auditLog.ID, "failed", nil, &errorMsg, tenantID)
	} else {
		log.Printf("Job %s for tenant %s executed successfully", jobName, tenantID)

		// Update status to completed using sqlc
		err = jm.store.UpdateJobStatusToCompleted(ctx, jobID)
		if err != nil {
			log.Printf("Error updating job status to completed: %v", err)
		}

		// Update audit log with success information
		output = "Job completed successfully"
		jm.updateAuditLogStatus(auditLog.ID, "completed", &output, nil, tenantID)
	}

	// Periodically clean up old completed tasks
	if jobName == "system.cleanup" || now.Minute() == 0 { // Run on the hour or with dedicated cleanup job
		_, err := jm.store.CleanupOldTasks(ctx, tenantID)
		if err != nil {
			log.Printf("Error cleaning up old tasks for tenant %s: %v", tenantID, err)
		}

		// Clean up stale locks
		result, err := jm.store.CleanupStaleLocks(ctx, tenantID)
		if err != nil {
			log.Printf("Error cleaning up stale locks for tenant %s: %v", tenantID, err)
		} else if rowsAffected := result.RowsAffected(); rowsAffected > 0 {
			log.Printf("Cleaned up %d stale job locks for tenant %s", rowsAffected, tenantID)
		}

		// Clean up stale registered jobs
		result, err = jm.store.CleanupStaleRegisteredJobs(ctx, tenantID)
		if err != nil {
			log.Printf("Error cleaning up stale registered jobs for tenant %s: %v", tenantID, err)
		} else if rowsAffected := result.RowsAffected(); rowsAffected > 0 {
			log.Printf("Cleaned up %d stale registered jobs for tenant %s", rowsAffected, tenantID)
		}
	}
}

// updateAuditLogStatus updates the job audit log with the final status
func (jm *JobManager) updateAuditLogStatus(auditLogID uuid.UUID, status string, output *string, errorMsg *string, tenantID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	endTime := pgtype.Timestamp{Time: time.Now(), Valid: true}

	// Convert output to pgtype.Text
	var outputText pgtype.Text
	if output != nil {
		outputText = pgtype.Text{String: *output, Valid: true}
	}

	// Convert error to pgtype.Text
	var errorText pgtype.Text
	if errorMsg != nil {
		errorText = pgtype.Text{String: *errorMsg, Valid: true}
	}

	// Update the audit log
	updateParams := repository.UpdateJobAuditLogParams{
		ID:       auditLogID,
		EndTime:  endTime,
		Status:   status,
		Output:   outputText,
		Error:    errorText,
		TenantID: tenantID,
	}

	_, err := jm.store.UpdateJobAuditLog(ctx, updateParams)
	if err != nil {
		log.Printf("Error updating job audit log %s: %v", auditLogID, err)
	}
}

// jobKey creates a unique key for a job based on name and tenant
func jobKey(name, tenantID string) string {
	return fmt.Sprintf("%s:%s", tenantID, name)
}

// jobLockToLockID converts a job name and tenant ID to a consistent lock ID
func jobLockToLockID(jobName, tenantID string) uint32 {
	// Combine job name and tenant ID to create a unique key
	key := jobKey(jobName, tenantID)

	// Simple hash function to convert string to uint32
	var hash uint32 = 5381
	for _, c := range key {
		hash = (hash << 5) + hash + uint32(c)
	}
	return hash
}
