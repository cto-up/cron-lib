package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cto-up/core-lib/api/helpers"
	"github.com/cto-up/core-lib/pkg/shared/auth"
	"github.com/cto-up/core-lib/pkg/shared/util"
	api "github.com/cto-up/cron-lib/api/openapi"
	"github.com/cto-up/cron-lib/pkg/db"
	"github.com/cto-up/cron-lib/pkg/db/repository"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/oapi-codegen/runtime/types"
)

// registeredJobLookupError tells a missing row apart from a failed query: a
// database outage must not be reported to the admin as "job not found".
func registeredJobLookupError(c *gin.Context, jobID types.UUID, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("registered job %s not found in this tenant", jobID)})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("loading registered job %s: %v", jobID, err)})
}

type RegisteredJobHandler struct {
	store          *db.Store
	authClientPool auth.AuthClientPool
}

func newRegisteredJobHandler(store *db.Store, authClientPool auth.AuthClientPool) *RegisteredJobHandler {
	return &RegisteredJobHandler{
		store:          store,
		authClientPool: authClientPool,
	}
}

// ListRegisteredJobs godoc
func (h *RegisteredJobHandler) ListRegisteredJobs(c *gin.Context, params api.ListRegisteredJobsParams) {
	tenantID, exists := c.Get(auth.AUTH_TENANT_ID_KEY)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tenant ID missing from the request context: the tenant middleware did not run"})
		return
	}

	pagingRequest := helpers.PagingRequest{
		MaxPageSize:     50,
		DefaultPage:     1,
		DefaultPageSize: 10,
		DefaultSortBy:   "job_name",
		DefaultOrder:    "asc",
		Page:            params.Page,
		PageSize:        params.PageSize,
		SortBy:          (*string)(params.SortBy),
		Order:           (*string)(params.Order),
	}

	pagingSql := helpers.GetPagingSQL(pagingRequest)

	searchTerm := ""
	if params.Q != nil {
		searchTerm = "%" + *params.Q + "%"
	}

	// Get jobs from database
	jobs, err := h.store.ListRegisteredJobs(c, repository.ListRegisteredJobsParams{
		TenantID:   tenantID.(string),
		Limit:      pagingSql.PageSize,
		Offset:     pagingSql.Offset,
		SortBy:     pagingSql.SortBy,
		Order:      pagingSql.Order,
		SearchTerm: searchTerm,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to API response format
	apiJobs := []api.RegisteredJob{}
	for _, job := range jobs {
		apiJob := api.RegisteredJob{
			Id:               job.ID,
			JobName:          job.JobName,
			Schedule:         job.Schedule,
			IsLongRunning:    job.IsLongRunning,
			IsEnabled:        job.IsEnabled,
			LastRegisteredAt: job.LastRegisteredAt,
			InstanceId:       job.InstanceID,
			TenantId:         job.TenantID,
			CreatedAt:        job.CreatedAt,
			UpdatedAt:        job.UpdatedAt,
		}

		apiJobs = append(apiJobs, apiJob)
	}

	c.JSON(http.StatusOK, apiJobs)
}

// GetRegisteredJob godoc
func (h *RegisteredJobHandler) GetRegisteredJob(c *gin.Context, jobID types.UUID) {
	tenantID, exists := c.Get(auth.AUTH_TENANT_ID_KEY)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tenant ID missing from the request context: the tenant middleware did not run"})
		return
	}

	// Get job from database
	job, err := h.store.GetRegisteredJobByID(c, repository.GetRegisteredJobByIDParams{
		ID:       jobID,
		TenantID: tenantID.(string),
	})
	if err != nil {
		registeredJobLookupError(c, jobID, err)
		return
	}

	// Convert to API response format
	apiJob := api.RegisteredJob{
		Id:               job.ID,
		JobName:          job.JobName,
		Schedule:         job.Schedule,
		IsLongRunning:    job.IsLongRunning,
		IsEnabled:        job.IsEnabled,
		LastRegisteredAt: job.LastRegisteredAt,
		InstanceId:       job.InstanceID,
		TenantId:         job.TenantID,
		CreatedAt:        job.CreatedAt,
		UpdatedAt:        job.UpdatedAt,
	}

	c.JSON(http.StatusOK, apiJob)
}

// UpdateRegisteredJob godoc
func (h *RegisteredJobHandler) UpdateRegisteredJob(c *gin.Context, jobID types.UUID) {
	tenantID, exists := c.Get(auth.AUTH_TENANT_ID_KEY)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tenant ID missing from the request context: the tenant middleware did not run"})
		return
	}

	// Parse request body
	var req struct {
		IsEnabled *bool `json:"is_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update job in database
	if req.IsEnabled != nil {
		tag, err := h.store.UpdateRegisteredJobEnabled(c, repository.UpdateRegisteredJobEnabledParams{
			ID:        jobID,
			TenantID:  tenantID.(string),
			IsEnabled: *req.IsEnabled,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("setting is_enabled on registered job %s: %v", jobID, err)})
			return
		}
		if tag.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("registered job %s not found in this tenant; is_enabled was not changed", jobID)})
			return
		}
	}

	// Get updated job
	job, err := h.store.GetRegisteredJobByID(c, repository.GetRegisteredJobByIDParams{
		ID:       jobID,
		TenantID: tenantID.(string),
	})
	if err != nil {
		registeredJobLookupError(c, jobID, err)
		return
	}

	// Convert to API response format
	apiJob := api.RegisteredJob{
		Id:               job.ID,
		JobName:          job.JobName,
		Schedule:         job.Schedule,
		IsLongRunning:    job.IsLongRunning,
		IsEnabled:        job.IsEnabled,
		LastRegisteredAt: job.LastRegisteredAt,
		InstanceId:       job.InstanceID,
		TenantId:         job.TenantID,
		CreatedAt:        job.CreatedAt,
		UpdatedAt:        job.UpdatedAt,
	}

	c.JSON(http.StatusOK, apiJob)
}

// GetJobAuditLogs godoc
func (h *RegisteredJobHandler) GetJobAuditLogs(c *gin.Context, jobID types.UUID, params api.GetJobAuditLogsParams) {
	tenantID, exists := c.Get(auth.AUTH_TENANT_ID_KEY)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tenant ID missing from the request context: the tenant middleware did not run"})
		return
	}

	// Get job from database to get job name
	job, err := h.store.GetRegisteredJobByID(c, repository.GetRegisteredJobByIDParams{
		ID:       jobID,
		TenantID: tenantID.(string),
	})
	if err != nil {
		registeredJobLookupError(c, jobID, err)
		return
	}

	// Parse pagination parameters
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("page must be a positive integer, got %q", c.Query("page"))})
		return
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if err != nil || pageSize < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("pageSize must be a positive integer, got %q", c.Query("pageSize"))})
		return
	}
	sortBy := c.DefaultQuery("sortBy", "start_time")
	order := c.DefaultQuery("order", "desc")

	// Get audit logs from database
	auditLogs, err := h.store.ListJobAuditLogsByJobName(c, repository.ListJobAuditLogsByJobNameParams{
		JobName:  job.JobName,
		TenantID: tenantID.(string),
		Limit:    int32(pageSize),
		Offset:   int32((page - 1) * pageSize),
		SortBy:   sortBy,
		Order:    order,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get total count
	totalCount, err := h.store.CountJobAuditLogsByJobName(c, repository.CountJobAuditLogsByJobNameParams{
		JobName:  job.JobName,
		TenantID: tenantID.(string),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = totalCount

	// Convert to API response format
	var apiAuditLogs []api.JobAuditLog
	for _, auditLog := range auditLogs {
		apiAuditLog := api.JobAuditLog{
			Id:            auditLog.ID,
			AppId:         auditLog.AppID,
			RequestId:     auditLog.RequestID,
			JobName:       auditLog.JobName,
			ScheduledTime: *util.FromNullableTimestamp(auditLog.ScheduledTime),
			StartTime:     util.FromNullableTimestamp(auditLog.StartTime),
			EndTime:       util.FromNullableTimestamp(auditLog.EndTime),
			Status:        auditLog.Status,
			Output:        util.FromNullableText(auditLog.Output),
			Error:         util.FromNullableText(auditLog.Error),
			TenantID:      auditLog.TenantID,
			CreatedAt:     auditLog.CreatedAt.Time,
			UpdatedAt:     util.FromNullableTimestamp(auditLog.UpdatedAt),
		}

		apiAuditLogs = append(apiAuditLogs, apiAuditLog)
	}

	c.JSON(http.StatusOK, apiAuditLogs)
}
