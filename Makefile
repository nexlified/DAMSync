.PHONY: dev build migrate-up migrate-down lint test docker-up docker-down tidy startup seed seed-minio seed-dam

# Development
dev:
	cp -n .env.example .env 2>/dev/null || true
	go run ./cmd/server

build:
	CGO_ENABLED=1 go build -o bin/dam ./cmd/server

# Docker
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Migrations (requires migrate CLI: https://github.com/golang-migrate/migrate)
migrate-up:
	migrate -database "$${DAM_DATABASE_DSN}" -path migrations up

migrate-down:
	migrate -database "$${DAM_DATABASE_DSN}" -path migrations down 1

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

# Testing
test:
	go test ./...

test-verbose:
	go test -v ./...

# Linting
lint:
	golangci-lint run ./...

# Dependency management
tidy:
	go mod tidy

# Run with env file
run-local:
	export $$(cat .env | xargs) && go run ./cmd/server

# Startup & seeding
startup:           ## Start Docker services + seed MinIO + seed DAM data
	bash scripts/startup.sh

startup-docker:    ## Start Docker services only (no seeding)
	bash scripts/startup.sh --no-seed

seed:              ## Seed both MinIO and DAM data
	bash scripts/seed-minio.sh
	bash scripts/seed-dam.sh

seed-minio:        ## Set up MinIO buckets/policies from seed/minio.yml
	bash scripts/seed-minio.sh

seed-dam:          ## Seed DAM orgs/users/styles from seed/data.yml
	bash scripts/seed-dam.sh
