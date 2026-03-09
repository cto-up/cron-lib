package api

import (
	"errors"
	"net/http"

	"ctoup.com/coreapp/pkg/shared/auth"
	access "ctoup.com/coreapp/pkg/shared/service"
	"ctoup.com/coreapp/pkg/shared/util"
	"github.com/cto-up/cron-lib/pkg/service"
	"github.com/gin-gonic/gin"
)

type SeedHandler struct {
	seedService *service.SeedService
}

func newSeedHandler(seedService *service.SeedService) *SeedHandler {
	return &SeedHandler{
		seedService: seedService,
	}
}

// SeedSkeellsCoachData implements the OpenAPI endpoint for seeding reference data
func (h *SeedHandler) SeedReferenceData(c *gin.Context) {
	logger := util.GetLoggerFromCtx(c.Request.Context())
	// Check if user has admin privileges
	if !access.IsAdmin(c) {
		logger.Error().Msg("User does not have admin privileges")
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}
	tenantID, exists := c.Get(auth.AUTH_TENANT_ID_KEY)
	if !exists {
		c.JSON(http.StatusInternalServerError, errors.New("TenantID not found"))
		return
	}

	err := h.seedService.SeedReferenceData(c.Request.Context(), tenantID.(string))
	if err != nil {
		logger.Err(err).Msg("Error seeding reference data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

func (h *SeedHandler) SeedSampleData(c *gin.Context) {
	logger := util.GetLoggerFromCtx(c.Request.Context())
	// Check if user has admin privileges
	if !access.IsAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}
	tenantID, exists := c.Get(auth.AUTH_TENANT_ID_KEY)
	if !exists {
		logger.Error().Msg("TenantID not found")
		c.JSON(http.StatusInternalServerError, errors.New("TenantID not found"))
		return
	}

	err := h.seedService.SeedSampleData(c.Request.Context(), tenantID.(string))
	if err != nil {
		logger.Err(err).Msg("Error seeding sample data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}
