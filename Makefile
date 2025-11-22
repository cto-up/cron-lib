include .env
export $(shell sed 's/=.*//' .env)
DB_CONNECTION = postgres://${DATABASE_USERNAME}:${DATABASE_PASSWORD}@${DATABASE_URL}
COMMAND ?= new # new:front_views
FILE ?= entity.json

testme:
	env

postgresup:
	docker compose -f docker/postgresql.yml up

postgresdown:
	docker compose -f docker/postgresql.yml down

migrateup:
	@$(MAKE) migrate-module DIRECTION=up
migrateup1:
	@$(MAKE) migrate-module DIRECTION=up STEP=1
migratedown:
	@$(MAKE) migrate-module DIRECTION=down
migratedown1:
	@$(MAKE) migrate-module DIRECTION=down STEP=1

migrate-module:
	cd pkg/db; \
	echo "I'm in pkg/db and $(DIRECTION) and $(STEP)"; \
	migrate -path migration -database "${DB_CONNECTION}&x-migrations-table=cron_migrations" -verbose $(DIRECTION) $(STEP)

sqlc:
	cd pkg/db; echo "I'm in backend cron"; \
	sqlc generate


BASE_API_BE_DIR := api/openapi
BASE_API_FE_DIR := ../cron-fe-lib

# Define the pattern to search for and replace
SEARCH_STRING_1 := from \'./core
REPLACE_STRING_1 := from \'core-fe-lib/openapi/core/core

SEARCH_STRING_2 := from \'../core
REPLACE_STRING_2 := from \'core-fe-lib/openapi/core/core

BASE_OPENAPI_CRON_DIR := pkg/api/openapi

build:
	go build ./...

openapi:
	@echo "Generating OpenAPI code"
	@find $(BASE_API_FE_DIR) -type f -name "*.ts" -delete
	openapi --input $(BASE_OPENAPI_CRON_DIR)/cron-api.yaml --output $(BASE_API_FE_DIR) --client axios
	@rm -rf $(BASE_API_FE_DIR)/$(MODULE)/core
	@find $(BASE_API_FE_DIR)/$(MODULE) -name "*.ts" -type f -exec sed -i '' "s|$(SEARCH_STRING_1)|$(REPLACE_STRING_1)|g" {} +
	@find $(BASE_API_FE_DIR)/$(MODULE) -name "*.ts" -type f -exec sed -i '' "s|$(SEARCH_STRING_2)|$(REPLACE_STRING_2)|g" {} +
	@echo "Replacement complete."
	
	oapi-codegen -config $(BASE_OPENAPI_CRON_DIR)/_oapi-schema-config.yaml $(BASE_OPENAPI_CRON_DIR)/cron-schema.yaml > api/openapi/cron-schema.go
	oapi-codegen -config $(BASE_OPENAPI_CRON_DIR)/_oapi-service-config.yaml $(BASE_OPENAPI_CRON_DIR)/cron-api.yaml > api/openapi/cron-service.go

release:
	@echo "Creating release"
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION parameter is required. Use 'vx.x.x' format."; \
		exit 1; \
	fi; \
	gh release create $(VERSION) --title "$(VERSION)" --notes "$(NOTES)"

include .env
export $(shell sed 's/=.*//' .env)
DB_CONNECTION = postgres://${DATABASE_USERNAME}:${DATABASE_PASSWORD}@${DATABASE_URL}

.PHONY: postgresup postgresdown migratecreate migrateup migratedown sqlc test openapi build
