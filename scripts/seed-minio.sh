#!/usr/bin/env bash
# scripts/seed-minio.sh — creates MinIO buckets and policies from seed/minio.yml
#
# Requirements:  python3, plus mc (MinIO client) or Docker
# Usage:         ./scripts/seed-minio.sh [path/to/minio.yml]
set -euo pipefail

SEED_FILE="${1:-seed/minio.yml}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SEED_FILE="$PROJECT_ROOT/${SEED_FILE#"$PROJECT_ROOT/"}"
PARSER="$SCRIPT_DIR/_parse-yaml.py"

if [[ ! -f "$SEED_FILE" ]]; then
  echo "ERROR: Seed file not found: $SEED_FILE" >&2; exit 1
fi

# Parse YAML → JSON
SEED_JSON=$(python3 "$PARSER" "$SEED_FILE")

ALIAS=$(echo "$SEED_JSON"  | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('minio',{}).get('alias','damlocal'))")
URL=$(echo "$SEED_JSON"    | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('minio',{}).get('url','http://localhost:9000'))")
AK=$(echo "$SEED_JSON"     | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('minio',{}).get('access_key','minioadmin'))")
SK=$(echo "$SEED_JSON"     | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('minio',{}).get('secret_key','minioadmin123'))")
BUCKET_COUNT=$(echo "$SEED_JSON" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('buckets',[])))")

echo "→ MinIO: $URL (alias: $ALIAS)"

# Build all mc commands to run in a single container invocation
MC_SCRIPT="mc alias set $ALIAS $URL $AK $SK"

for i in $(seq 0 $((BUCKET_COUNT - 1))); do
  BUCKET=$(echo "$SEED_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['buckets'][$i]['name'])")
  POLICY=$(echo "$SEED_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['buckets'][$i].get('policy','none'))")

  MC_SCRIPT="$MC_SCRIPT && mc mb --ignore-existing $ALIAS/$BUCKET"

  case "$POLICY" in
    download)                MC_SCRIPT="$MC_SCRIPT && mc anonymous set download $ALIAS/$BUCKET" ;;
    upload)                  MC_SCRIPT="$MC_SCRIPT && mc anonymous set upload   $ALIAS/$BUCKET" ;;
    public|public-read-write) MC_SCRIPT="$MC_SCRIPT && mc anonymous set public  $ALIAS/$BUCKET" ;;
    none|private|"")         ;;
    *)                       echo "  WARNING: unknown policy '$POLICY' for $BUCKET" ;;
  esac

  echo "  → Bucket: $BUCKET (policy: $POLICY)"
done

# Run all commands in a single mc invocation (preserves alias state)
if command -v mc &>/dev/null; then
  eval "$MC_SCRIPT"
elif docker info &>/dev/null 2>&1; then
  echo "  (using Docker mc image)"
  docker run --rm --network host --entrypoint sh minio/mc -c "$MC_SCRIPT"
else
  echo "ERROR: 'mc' not found and Docker unavailable." >&2
  echo "Install mc: https://min.io/docs/minio/linux/reference/minio-mc.html" >&2
  exit 1
fi

echo "✓ MinIO setup complete"
