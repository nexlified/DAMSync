#!/usr/bin/env bash
# scripts/startup.sh — full local development startup
#
# What it does:
#   1. Starts Docker services (postgres, redis, minio)
#   2. Waits for them to be healthy
#   3. Sets up MinIO buckets/policies from seed/minio.yml
#   4. Seeds DAM initial data (orgs, users, styles) from seed/data.yml
#
# Usage:
#   ./scripts/startup.sh              # start everything + seed
#   ./scripts/startup.sh --no-seed    # start Docker only, skip seeding
#   ./scripts/startup.sh --seed-only  # skip Docker start, just seed
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

NO_SEED=false
SEED_ONLY=false

for arg in "$@"; do
  case "$arg" in
    --no-seed)   NO_SEED=true ;;
    --seed-only) SEED_ONLY=true ;;
  esac
done

cd "$PROJECT_ROOT"

# ---------------------------------------------------------------------------
# 1. Start Docker services
# ---------------------------------------------------------------------------
if [[ "$SEED_ONLY" == "false" ]]; then
  echo "==> Starting Docker services ..."
  docker compose up -d

  echo "==> Waiting for services to become healthy ..."

  wait_healthy() {
    local name="$1"
    local max=30
    for i in $(seq 1 $max); do
      status=$(docker inspect --format='{{.State.Health.Status}}' "$name" 2>/dev/null || echo "missing")
      if [[ "$status" == "healthy" ]]; then
        echo "  ✓ $name is healthy"
        return 0
      fi
      printf "  · $name: $status (%d/%d)\r" "$i" "$max"
      sleep 3
    done
    echo "  ERROR: $name did not become healthy in time" >&2
    return 1
  }

  wait_healthy dam_postgres
  wait_healthy dam_redis
  wait_healthy dam_minio
fi

# ---------------------------------------------------------------------------
# 2. Set up MinIO buckets
# ---------------------------------------------------------------------------
if [[ "$NO_SEED" == "false" ]]; then
  echo ""
  echo "==> Setting up MinIO ..."
  bash "$SCRIPT_DIR/seed-minio.sh" seed/minio.yml
fi

# ---------------------------------------------------------------------------
# 3. Seed DAM data
# ---------------------------------------------------------------------------
if [[ "$NO_SEED" == "false" ]]; then
  echo ""
  echo "==> Seeding DAM initial data ..."
  bash "$SCRIPT_DIR/seed-dam.sh" seed/data.yml
fi

echo ""
echo "============================================================"
echo "  Local environment is ready."
echo "  DAM API:      http://localhost:8080"
echo "  MinIO console: http://localhost:9001  (minioadmin / minioadmin123)"
echo "============================================================"
