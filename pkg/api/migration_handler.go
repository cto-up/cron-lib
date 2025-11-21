package api

import (
	"net/http"
	"strings"

	access "ctoup.com/coreapp/pkg/shared/service"
	"github.com/cto-up/cron-lib/pkg/db"
	"github.com/gin-gonic/gin"
)

type MigrationHandler struct {
	store *db.Store
}

func newMigrationHandler(store *db.Store) *MigrationHandler {
	return &MigrationHandler{
		store: store,
	}
}

// MigrationSkeellsCoachData implements the OpenAPI endpoint for migrationing reference data
func (h *MigrationHandler) MigrateUp(c *gin.Context) {
	// Check if user has admin privileges
	if !access.IsSuperAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Super Admin privileges required"})
		return
	}
	err := db.MigrateUp(h.store.ConnPool.Config().ConnString())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

func (h *MigrationHandler) MigrateDown(c *gin.Context) {
	// Check if user has admin privileges
	if !access.IsSuperAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Super Admin privileges required"})
		return
	}

	err := db.MigrateDown(h.store.ConnPool.Config().ConnString())
	if err != nil {
		if strings.Contains(err.Error(), "TRUNCATE") {
			c.Status(http.StatusNoContent)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
