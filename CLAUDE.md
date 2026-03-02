# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build (CGO required — bimg links against libvips)
CGO_ENABLED=1 go build -o bin/dam ./cmd/server

# Run (reads .env automatically)
./bin/dam
# or
make dev         # copies .env.example → .env if missing, then go run ./cmd/server

# Docker services (Postgres 16, Redis 7, MinIO)
make docker-up
make docker-down

# Full local bootstrap: Docker + MinIO bucket setup + seed initial data
make startup                     # docker up + seed
bash scripts/seed-minio.sh       # MinIO buckets from seed/minio.yml
bash scripts/seed-dam.sh         # orgs/users/styles from seed/data.yml (idempotent)

# Migrations (golang-migrate)
make migrate-up                  # requires $DAM_DATABASE_DSN set
make migrate-down
make migrate-create name=add_foo # creates new migration pair

# Tests
go test ./...
go test -v ./internal/application/services/...   # specific package

# Linting
golangci-lint run ./...

# Dependency management
go mod tidy
```

**System requirement:** `libvips` must be installed (`brew install vips` on macOS).

## Configuration

Config loads from (in priority order): OS env vars → `.env` file → `config.yml` → defaults.

Env var prefix is `DAM_`. Key mappings: `DAM_DATABASE_DSN`, `DAM_REDIS_ADDR`, `DAM_JWT_ACCESS_SECRET`, `DAM_JWT_REFRESH_SECRET`, `DAM_STORAGE_ENDPOINT`, etc.

Copy `.env.example` to `.env` for local development. The `.env` file is parsed by `config.Load()` directly (not by the shell), so values don't need to be exported.

**Viper gotcha:** every config key must have a `v.SetDefault()` call in `config/config.go` for `v.Unmarshal()` to pick it up from `AutomaticEnv()`. This is already done — don't add new config fields without a default.

## Architecture

Hexagonal (Ports & Adapters). Dependencies point inward only.

```
cmd/server/main.go          ← wires everything, starts Fiber
config/                     ← Viper config loading
internal/
  domain/                   ← pure Go structs, sentinel errors, events (no framework imports)
  application/
    ports/inbound/          ← service interfaces (what HTTP handlers call)
    ports/outbound/         ← repository/storage/cache interfaces (what services call)
    services/               ← use-case implementations (imports domain + outbound ports only)
  infrastructure/
    http/                   ← Fiber app, middleware, v1/ handlers, serve/ CDN handlers
    postgres/               ← sqlx repositories implementing outbound.Repository interfaces
    redis/                  ← cache, rate limiter, event publisher
    storage/                ← S3-compatible object storage (AWS SDK v2)
    transform/              ← bimg/libvips image transformer
migrations/                 ← golang-migrate SQL files
seed/                       ← data.yml, minio.yml for local bootstrapping
scripts/                    ← seed-dam.sh, seed-minio.sh, startup.sh, _parse-yaml.py
```

### Request lifecycle

1. All requests hit global middleware: recover → request ID → security headers → dev logger → gzip → **domain resolver** (maps `Host:` header → org via Redis cache / DB).
2. CDN routes (`/files/*`, `/styles/:style/*`, `/secure/:token/:expires/*`) are unauthenticated; org is resolved from the domain resolver.
3. API routes (`/api/v1/`) apply a rate limiter (200 req/60s per IP via Redis sorted sets).
4. Protected routes require `RequireAuth` middleware which accepts either `Authorization: Bearer <jwt>` or `X-API-Key: <key>` (also `?api_key=` query param). Both set `org_id` in Fiber locals.
5. `RequireRole("owner","admin")` checks JWT claims role. `RequireScope("assets:write")` checks API key scopes (JWT users bypass scope checks).

### Image style transform flow

`GET /styles/{slug}/{asset-path}` → `serve.Handler.ServeStyled` → `StyleService.ServeStyled`:
1. Check Redis cache (`dam:transform:{hash}`) → serve if hit.
2. Check `transform_cache` DB table → download from S3, serve, and warm Redis.
3. Download original from S3 → bimg transform → upload to `orgs/{org}/transforms/{style}/{asset}.{fmt}` → save DB record → cache in Redis → serve.

### Authentication

- **JWT**: HMAC-SHA256, separate access (default 15m) and refresh (168h) secrets. Refresh token revocation stored in Redis as `revoked_token:{jti}`.
- **API Keys**: raw key = `{8-char-prefix}.{secret}`. Prefix stored plaintext for lookup; full key bcrypt-hashed. Created via `POST /api/v1/api-keys`.
- **Signed URLs**: `HMAC-SHA256(org_secret, "{assetID}:{timestamp}")`, verified by `AssetService.ValidateSignedURL`.

### Multi-tenancy

Every entity is org-scoped. The org ID comes from one of three sources (set in Fiber locals under `org_id`):
1. JWT claims (`OrgID` field)
2. API key's `OrgID`
3. Domain resolver (maps custom `Host:` header → org)

Handlers call `mustOrgID(c)` (defined in `internal/infrastructure/http/v1/`) to extract it.

### Error handling

`domain/errors.go` defines sentinel errors (`ErrNotFound`, `ErrUnauthorized`, etc.) and a `DomainError` wrapper. The Fiber `errorHandler` in `server.go` maps error message substrings to HTTP status codes. Wrap domain errors with `domain.NewError(sentinel, "message")` to propagate correctly.

### Database

Single migration file `migrations/000001_initial_schema.up.sql`. Run automatically on server startup. Notable details:
- `assets` table has a `search_vector TSVECTOR` column auto-updated via trigger for PostgreSQL FTS.
- `audit_logs` is partitioned by year (`PARTITION BY RANGE (created_at)`); PRIMARY KEY includes `created_at` (required by PostgreSQL).
- `transform_cache` uses `ON CONFLICT (asset_id, params_hash) DO UPDATE` for upsert.

### Known incomplete stubs

- `ServeSigned` handler validates expiry but skips HMAC verification (TODO comment in `serve/handler.go`).
- Webhook async dispatch worker not yet started (see `main.go` `// TODO: subscribe to event stream`).
- ACME/certmagic TLS automation is scaffolded in config but not wired into the server.
