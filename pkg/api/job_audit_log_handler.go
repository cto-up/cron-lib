package api

import (
	"errors"
	"net/http"

	"ctoup.com/coreapp/pkg/shared/repository/subentity"
	access "ctoup.com/coreapp/pkg/shared/service"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"ctoup.com/coreapp/api/helpers"
	api "github.com/cto-up/cron-lib/api/openapi"
	"github.com/cto-up/cron-lib/pkg/db"
	"github.com/cto-up/cron-lib/pkg/db/repository"
	"github.com/oapi-codegen/runtime/types"
)

// https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-One_Of
type JobAuditLogHandler struct {
	authClientPool *access.FirebaseTenantClientConnectionPool
	store          *db.Store
}

// DeleteJobAuditLog implements api.ServerInterface.
func (h *JobAuditLogHandler) DeleteJobAuditLog(c *gin.Context, id types.UUID) {
	tenantID, exists := c.Get(access.AUTH_TENANT_ID_KEY)
	if !exists {
		c.JSON(http.StatusInternalServerError, errors.New("TenantID not found"))
		return
	}
	_, err := h.store.DeleteJobAuditLog(c, repository.DeleteJobAuditLogParams{
		ID:       id,
		TenantID: tenantID.(string),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, helpers.ErrorResponse(err))
		return
	}
	c.Status(http.StatusNoContent)
}

// FindJobAuditLogByID implements api.ServerInterface.
func (h *JobAuditLogHandler) GetJobAuditLogByID(c *gin.Context, id types.UUID) {
	tenantID, exists := c.Get(access.AUTH_TENANT_ID_KEY)
	if !exists {
		c.JSON(http.StatusInternalServerError, errors.New("TenantID not found"))
		return
	}
	jobAuditLog, err := h.store.GetJobAuditLogByID(c, repository.GetJobAuditLogByIDParams{
		ID:       id,
		TenantID: tenantID.(string),
	})
	if err != nil {
		if err.Error() == pgx.ErrNoRows.Error() {
			c.JSON(http.StatusNotFound, helpers.ErrorResponse(err))
			return
		}
		c.JSON(http.StatusInternalServerError, helpers.ErrorResponse(err))
		return
	}

	c.JSON(http.StatusOK, jobAuditLog)
}

// ListJobAuditLogs implements api.ServerInterface.
func (h *JobAuditLogHandler) ListJobAuditLogs(c *gin.Context, params api.ListJobAuditLogsParams) {
	tenantID, exists := c.Get(access.AUTH_TENANT_ID_KEY)
	if !exists {
		c.JSON(http.StatusInternalServerError, errors.New("TenantID not found"))
		return
	}
	pagingRequest := helpers.PagingRequest{
		MaxPageSize:     50,
		DefaultPage:     1,
		DefaultPageSize: 10,
		DefaultSortBy:   "scheduled_time",
		DefaultOrder:    "desc",
		Page:            params.Page,
		PageSize:        params.PageSize,
		SortBy:          params.SortBy,
		Order:           (*string)(params.Order),
	}

	pagingSql := helpers.GetPagingSQL(pagingRequest)

	like := pgtype.Text{
		Valid: false,
	}

	if params.Q != nil {
		like.String = *params.Q + "%"
		like.Valid = true
	}

	query := repository.ListJobAuditLogsParams{
		Limit:    pagingSql.PageSize,
		Offset:   pagingSql.Offset,
		Like:     like,
		SortBy:   pagingSql.SortBy,
		Order:    pagingSql.Order,
		TenantID: tenantID.(string),
	}

	jobAuditLogs, err := h.store.ListJobAuditLogs(c, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, helpers.ErrorResponse(err))
		return
	}

	if params.Detail != nil && *params.Detail == "basic" {
		basicEntities := make([]subentity.BasicEntity, 0)
		for _, jobAuditLog := range jobAuditLogs {
			basicEntity := subentity.BasicEntity{
				ID:   jobAuditLog.ID.String(),
				Name: jobAuditLog.JobName,
			}
			basicEntities = append(basicEntities, basicEntity)
		}
		c.JSON(http.StatusOK, basicEntities)
	} else {
		c.JSON(http.StatusOK, jobAuditLogs)
	}
}

func newJobAuditLogHandler(store *db.Store, authClientPool *access.FirebaseTenantClientConnectionPool) *JobAuditLogHandler {
	return &JobAuditLogHandler{
		store:          store,
		authClientPool: authClientPool,
	}
}
