package api

import (
	"context"

	api "github.com/cto-up/cron-lib/api/openapi"

	"ctoup.com/coreapp/api/openapi/core"
	access "ctoup.com/coreapp/pkg/shared/service"
	"github.com/cto-up/cron-lib/pkg"
	"github.com/cto-up/cron-lib/pkg/db"
	"github.com/cto-up/cron-lib/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CronHandler struct {
	authClientPool *access.FirebaseTenantClientConnectionPool
	store          *db.Store
	*JobAuditLogHandler
	*MigrationHandler
	*SeedHandler
	*RegisteredJobHandler
}

func RegisterHandler(connPool *pgxpool.Pool, firebaseTenantClientPool *access.FirebaseTenantClientConnectionPool, openaiOptions core.GinServerOptions, router *gin.Engine) {

	// Create job manager
	jobManager := cron.InitJobManager(context.Background(), connPool)

	// Start scheduler
	jobManager.StartScheduler()

	store := db.NewStore(connPool, false)
	var middlewares []api.MiddlewareFunc
	for _, mw := range openaiOptions.Middlewares {
		middlewares = append(middlewares, api.MiddlewareFunc(mw))
	}
	options := api.GinServerOptions{
		BaseURL:     "",
		Middlewares: middlewares,
	}

	handler := &CronHandler{
		store:                store,
		authClientPool:       firebaseTenantClientPool,
		JobAuditLogHandler:   newJobAuditLogHandler(store, firebaseTenantClientPool),
		MigrationHandler:     newMigrationHandler(store),
		SeedHandler:          newSeedHandler(service.NewSeedService(connPool)),
		RegisteredJobHandler: newRegisteredJobHandler(store, firebaseTenantClientPool),
	}
	api.RegisterHandlersWithOptions(router, handler, options)
}
