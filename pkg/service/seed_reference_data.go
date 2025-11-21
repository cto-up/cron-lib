package service

import (
	"context"

	coreRepo "ctoup.com/coreapp/pkg/core/db/repository"
	"github.com/cto-up/cron-lib/pkg/db/repository"
)

func (s *SeedService) seedCoreRefenceData(ctx context.Context, tenantID string) error {
	return nil
}

func (s *SeedService) seedRefenceData(ctx context.Context, qtx *repository.Queries, coreTx *coreRepo.Queries, tenantID string) error {
	return s.seedCoreRefenceData(ctx, tenantID)
}
