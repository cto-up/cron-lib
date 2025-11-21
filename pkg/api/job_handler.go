package api

import (
	"errors"
	"net/http"

	"ctoup.com/coreapp/pkg/shared/repository/subentity"
	access "ctoup.com/coreapp/pkg/shared/service"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/oapi-codegen/runtime/types"

	"ctoup.com/coreapp/api/helpers"
	api "github.com/cto-up/cron-lib/api/openapi"
	"github.com/cto-up/cron-lib/pkg/db/repository"
)

// DeleteJob implements api.ServerInterface.
func (h *CronHandler) DeleteJob(c *gin.Context, id types.UUID) {
	tenantID, exists := c.Get(access.AUTH_TENANT_ID_KEY)
	if !exists {
		c.JSON(http.StatusInternalServerError, errors.New("TenantID not found"))
		return
	}
	_, err := h.store.DeleteJob(c, repository.DeleteJobParams{
		ID:       id,
		TenantID: tenantID.(string),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, helpers.ErrorResponse(err))
		return
	}
	c.Status(http.StatusNoContent)
}

// FindJobByID implements api.ServerInterface.
func (h *CronHandler) GetJobByID(c *gin.Context, id types.UUID, params api.GetJobByIDParams) {
	tenantID, exists := c.Get(access.AUTH_TENANT_ID_KEY)
	if !exists {
		c.JSON(http.StatusInternalServerError, errors.New("TenantID not found"))
		return
	}
	job, err := h.store.GetJobByID(c, repository.GetJobByIDParams{
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

	c.JSON(http.StatusOK, job)
}

// ListJobs implements api.ServerInterface.
func (h *CronHandler) ListJobs(c *gin.Context, params api.ListJobsParams) {
	tenantID, exists := c.Get(access.AUTH_TENANT_ID_KEY)
	if !exists {
		c.JSON(http.StatusInternalServerError, errors.New("TenantID not found"))
		return
	}
	pagingRequest := helpers.PagingRequest{
		MaxPageSize:     50,
		DefaultPage:     1,
		DefaultPageSize: 10,
		DefaultSortBy:   "name",
		DefaultOrder:    "asc",
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

	query := repository.ListJobsParams{
		Limit:    pagingSql.PageSize,
		Offset:   pagingSql.Offset,
		Like:     like,
		SortBy:   pagingSql.SortBy,
		Order:    pagingSql.Order,
		TenantID: tenantID.(string),
	}

	jobs, err := h.store.ListJobs(c, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, helpers.ErrorResponse(err))
		return
	}

	if params.Detail != nil && *params.Detail == "basic" {
		basicEntities := make([]subentity.BasicEntity, 0)
		for _, job := range jobs {
			basicEntity := subentity.BasicEntity{
				ID:   job.ID.String(),
				Name: job.JobName,
			}
			basicEntities = append(basicEntities, basicEntity)
		}
		c.JSON(http.StatusOK, basicEntities)
	} else {
		c.JSON(http.StatusOK, jobs)
	}
}
