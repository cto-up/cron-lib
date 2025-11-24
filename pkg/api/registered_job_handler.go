package api

import (
	"errors"
	"net/http"
	"strconv"

	"ctoup.com/coreapp/api/helpers"
	access "ctoup.com/coreapp/pkg/shared/service"
	"ctoup.com/coreapp/pkg/shared/util"
	api "github.com/cto-up/cron-lib/api/openapi"
	"github.com/cto-up/cron-lib/pkg/db"
	"github.com/cto-up/cron-lib/pkg/db/repository"
	"github.com/gin-gonic/gin"
	"github.com/oapi-codegen/runtime/types"
)

type RegisteredJobHandler struct {
	store          *db.Store
	authClientPool *access.FirebaseTenantClientConnectionPool
}

func newRegisteredJobHandler(store *db.Store, authClientPool *access.FirebaseTenantClientConnectionPool) *RegisteredJobHandler {
	return &RegisteredJobHandler{
		store:          store,
		authClientPool: authClientPool,
	}
}

// ListRegisteredJobs godoc
func (h *RegisteredJobHandler) ListRegisteredJobs(c *gin.Context, params api.ListRegisteredJobsParams) {
	tenantID, exists := c.Get(access.AUTH_TENANT_ID_KEY)
	if !exists {
		c.JSON(http.StatusInternalServerError, errors.New("TenantID not found"))
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
	tenantID, exists := c.Get(access.AUTH_TENANT_ID_KEY)
	if !exists {
		c.JSON(http.StatusInternalServerError, errors.New("TenantID not found"))
		return
	}

	// Get job from database
	job, err := h.store.GetRegisteredJobByID(c, repository.GetRegisteredJobByIDParams{
		ID:       jobID,
		TenantID: tenantID.(string),
	})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
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
	tenantID, exists := c.Get(access.AUTH_TENANT_ID_KEY)
	if !exists {
		c.JSON(http.StatusInternalServerError, errors.New("TenantID not found"))
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
		_, err := h.store.UpdateRegisteredJobEnabled(c, repository.UpdateRegisteredJobEnabledParams{
			ID:        jobID,
			TenantID:  tenantID.(string),
			IsEnabled: *req.IsEnabled,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Get updated job
	job, err := h.store.GetRegisteredJobByID(c, repository.GetRegisteredJobByIDParams{
		ID:       jobID,
		TenantID: tenantID.(string),
	})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
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
	tenantID, exists := c.Get(access.AUTH_TENANT_ID_KEY)
	if !exists {
		c.JSON(http.StatusInternalServerError, errors.New("TenantID not found"))
		return
	}

	// Get job from database to get job name
	job, err := h.store.GetRegisteredJobByID(c, repository.GetRegisteredJobByIDParams{
		ID:       jobID,
		TenantID: tenantID.(string),
	})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
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
	for _, log := range auditLogs {
		apiAuditLog := api.JobAuditLog{
			Id:            log.ID,
			AppId:         log.AppID,
			RequestId:     log.RequestID,
			JobName:       log.JobName,
			ScheduledTime: *util.FromNullableTimestamp(log.ScheduledTime),
			StartTime:     util.FromNullableTimestamp(log.StartTime),
			EndTime:       util.FromNullableTimestamp(log.EndTime),
			Status:        log.Status,
			Output:        util.FromNullableText(log.Output),
			Error:         util.FromNullableText(log.Error),
			TenantID:      log.TenantID,
			CreatedAt:     log.CreatedAt.Time,
			UpdatedAt:     util.FromNullableTimestamp(log.UpdatedAt),
		}

		apiAuditLogs = append(apiAuditLogs, apiAuditLog)
	}

	c.JSON(http.StatusOK, apiAuditLogs)
}
