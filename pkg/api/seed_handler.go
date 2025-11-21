package api

import (
	"errors"
	"net/http"

	access "ctoup.com/coreapp/pkg/shared/service"
	"github.com/cto-up/cron-lib/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
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
	// Check if user has admin privileges
	if !access.IsAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}
	tenantID, exists := c.Get(access.AUTH_TENANT_ID_KEY)
	if !exists {
		c.JSON(http.StatusInternalServerError, errors.New("TenantID not found"))
		return
	}

	err := h.seedService.SeedReferenceData(c.Request.Context(), tenantID.(string))
	if err != nil {
		log.Error().Err(err).Msg("Error seeding reference data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

func (h *SeedHandler) SeedSampleData(c *gin.Context) {
	// Check if user has admin privileges
	if !access.IsAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}
	tenantID, exists := c.Get(access.AUTH_TENANT_ID_KEY)
	if !exists {
		c.JSON(http.StatusInternalServerError, errors.New("TenantID not found"))
		return
	}

	err := h.seedService.SeedSampleData(c.Request.Context(), tenantID.(string))
	if err != nil {
		log.Error().Err(err).Msg("Error seeding sample data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}
