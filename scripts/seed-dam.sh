#!/usr/bin/env bash
# scripts/seed-dam.sh — seeds DAM initial data (orgs, users, styles) from seed/data.yml
#
# Requirements:  curl, python3
# Usage:         ./scripts/seed-dam.sh [path/to/data.yml]
#
# Idempotent: skips records that already exist.
set -euo pipefail

SEED_FILE="${1:-seed/data.yml}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SEED_FILE="$PROJECT_ROOT/${SEED_FILE#"$PROJECT_ROOT/"}"
PARSER="$SCRIPT_DIR/_parse-yaml.py"

if [[ ! -f "$SEED_FILE" ]]; then
  echo "ERROR: Seed file not found: $SEED_FILE" >&2; exit 1
fi

# Parse YAML → JSON
SEED_JSON=$(python3 "$PARSER" "$SEED_FILE")

DAM_URL=$(echo "$SEED_JSON"  | python3 -c "import sys,json; print(json.load(sys.stdin).get('dam_url','http://localhost:8080'))")
ORG_COUNT=$(echo "$SEED_JSON" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('organizations',[])))")

# ---------------------------------------------------------------------------
# Wait for DAM server to be ready (up to 60 s)
# ---------------------------------------------------------------------------
echo "→ Waiting for DAM server at $DAM_URL ..."
for i in $(seq 1 30); do
  if curl -sf -X POST "$DAM_URL/api/v1/auth/login" \
      -H "Content-Type: application/json" -d '{}' -o /dev/null 2>/dev/null; then
    break
  fi
  sleep 2
done
echo "  DAM is up."

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
dam_post() {
  local path="$1" body="$2" token="${3:-}"
  if [[ -n "$token" ]]; then
    curl -s -X POST "$DAM_URL$path" \
      -H "Content-Type: application/json" -H "Authorization: Bearer $token" -d "$body"
  else
    curl -s -X POST "$DAM_URL$path" -H "Content-Type: application/json" -d "$body"
  fi
}

json_val() {
  # Extract a field from a JSON string: json_val "$json" field
  echo "$1" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('$2',''))"
}

# ---------------------------------------------------------------------------
# Seed each organisation
# ---------------------------------------------------------------------------
for i in $(seq 0 $((ORG_COUNT - 1))); do
  ORG=$(echo "$SEED_JSON" | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin)['organizations'][$i]))")

  ORG_NAME=$(json_val "$ORG" name)
  ORG_SLUG=$(json_val "$ORG" slug)
  ORG_PLAN=$(echo "$ORG" | python3 -c "import sys,json; print(json.load(sys.stdin).get('plan','free'))")
  OWNER_EMAIL=$(echo "$ORG" | python3 -c "import sys,json; print(json.load(sys.stdin).get('owner',{}).get('email',''))")
  OWNER_PASS=$(echo "$ORG"  | python3 -c "import sys,json; print(json.load(sys.stdin).get('owner',{}).get('password',''))")
  STYLE_COUNT=$(echo "$ORG" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('styles',[])))")

  echo ""
  echo "==> Org: $ORG_NAME ($ORG_SLUG)"

  # Register org + owner user
  REG_BODY=$(python3 -c "
import json, sys
print(json.dumps({
  'org_name':  sys.argv[1],
  'org_slug':  sys.argv[2],
  'email':     sys.argv[3],
  'password':  sys.argv[4],
}))
" "$ORG_NAME" "$ORG_SLUG" "$OWNER_EMAIL" "$OWNER_PASS")

  REG_RESP=$(dam_post "/api/v1/auth/register" "$REG_BODY")
  REG_ERR=$(json_val "$REG_RESP" error)

  if [[ -z "$REG_ERR" ]]; then
    echo "  ✓ Org + owner created"
  else
    echo "  ℹ Skipped ($REG_ERR)"
  fi

  # Login to get bearer token
  LOGIN_BODY=$(python3 -c "
import json, sys
print(json.dumps({'org_slug': sys.argv[1], 'email': sys.argv[2], 'password': sys.argv[3]}))
" "$ORG_SLUG" "$OWNER_EMAIL" "$OWNER_PASS")

  LOGIN_RESP=$(dam_post "/api/v1/auth/login" "$LOGIN_BODY")
  TOKEN=$(json_val "$LOGIN_RESP" access_token)

  if [[ -z "$TOKEN" ]]; then
    echo "  ERROR: login failed for $OWNER_EMAIL — skipping styles" >&2
    continue
  fi
  echo "  ✓ Authenticated as $OWNER_EMAIL"

  # Seed image styles
  for j in $(seq 0 $((STYLE_COUNT - 1))); do
    STYLE=$(echo "$ORG" | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin)['styles'][$j]))")

    STYLE_NAME=$(json_val "$STYLE" name)
    STYLE_SLUG=$(json_val "$STYLE" slug)
    STYLE_FMT=$(echo "$STYLE"  | python3 -c "import sys,json; print(json.load(sys.stdin).get('output_format','webp'))")
    STYLE_QUAL=$(echo "$STYLE" | python3 -c "import sys,json; print(json.load(sys.stdin).get('quality',85))")
    STYLE_OPS=$(echo "$STYLE"  | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin).get('operations',[])))")

    STYLE_BODY=$(python3 -c "
import json, sys
ops_raw = json.loads(sys.argv[1])
ops = [{k: v for k, v in op.items()} for op in ops_raw]
print(json.dumps({
  'name':          sys.argv[2],
  'slug':          sys.argv[3],
  'output_format': sys.argv[4],
  'quality':       int(sys.argv[5]),
  'operations':    ops,
}))
" "$STYLE_OPS" "$STYLE_NAME" "$STYLE_SLUG" "$STYLE_FMT" "$STYLE_QUAL")

    STYLE_RESP=$(dam_post "/api/v1/styles" "$STYLE_BODY" "$TOKEN")
    STYLE_ERR=$(json_val "$STYLE_RESP" error)

    if [[ -z "$STYLE_ERR" ]]; then
      echo "  ✓ Style: $STYLE_SLUG"
    else
      echo "  ℹ Style '$STYLE_SLUG' skipped ($STYLE_ERR)"
    fi
  done
done

echo ""
echo "✓ DAM seeding complete"
