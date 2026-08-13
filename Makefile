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

sqlc:
	cd pkg/db; echo "I'm in backend cron"; \
	sqlc generate


BASE_API_BE_DIR := api/openapi
# Frontend codegen lands in cron-fe-lib/lib/ — everything outside lib/ is
# hand-authored (Vue feature module) and must not be wiped by this target.
BASE_API_FE_DIR := ../cron-fe-lib/lib

# Define the pattern to search for and replace
SEARCH_STRING_1 := from \'./core
REPLACE_STRING_1 := from \'core-fe-lib/openapi/core/core

SEARCH_STRING_2 := from \'../core
REPLACE_STRING_2 := from \'core-fe-lib/openapi/core/core

BASE_OPENAPI_DIR := pkg/api/openapi

build:
	go build ./...

openapi:
	@echo "Generating OpenAPI code"
	@rm -rf $(BASE_API_FE_DIR)
	openapi --input $(BASE_OPENAPI_DIR)/cron-api.yaml --output $(BASE_API_FE_DIR) --client axios
	@rm -rf $(BASE_API_FE_DIR)/core
	@find $(BASE_API_FE_DIR) -name "*.ts" -type f -exec sed -i '' "s|$(SEARCH_STRING_1)|$(REPLACE_STRING_1)|g" {} +
	@find $(BASE_API_FE_DIR) -name "*.ts" -type f -exec sed -i '' "s|$(SEARCH_STRING_2)|$(REPLACE_STRING_2)|g" {} +
	@echo "Replacement complete."
	
	oapi-codegen -config $(BASE_OPENAPI_DIR)/_oapi-schema-config.yaml $(BASE_OPENAPI_DIR)/cron-schema.yaml > api/openapi/cron-schema.go
	oapi-codegen -config $(BASE_OPENAPI_DIR)/_oapi-service-config.yaml $(BASE_OPENAPI_DIR)/cron-api.yaml > api/openapi/cron-service.go

update-core-backend:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION parameter is required. Use 'vx.x.x' format."; \
		exit 1; \
	fi
	go get -u ctoup.com/coreapp@$(VERSION)


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


.PHONY: postgresup postgresdown sqlc test openapi build update-core-backend

# Library migrations are 16 digits: YYYYMMDDHHMMSS + source id 02. Consumers
# flatten every library's migrations and their own modules into ONE goose
# namespace, so a bare timestamp can collide with an app migration — which is
# exactly how coreapp v0.2.29 collided with skeells on 20260810120000. The
# 2-digit suffix makes a cross-repo collision arithmetically impossible.
new-migration: ## Create a goose migration (NAME=<snake_case>)
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Example: make new-migration NAME=add_user_index"; \
		exit 1; \
	fi
	@VERSION="$$(date -u +%Y%m%d%H%M%S)02"; \
	FILE="pkg/db/migration/$${VERSION}_$(NAME).sql"; \
	if [ -e "$$FILE" ]; then echo "Error: $$FILE already exists."; exit 1; fi; \
	printf -- '-- +goose Up\n\n\n-- +goose Down\n\n' > "$$FILE"; \
	echo "Created $$FILE"
	@./scripts/check-migration-versions.sh
