#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_URL="${BACKEND_URL:-http://localhost:8082}"
COMPOSE="${COMPOSE:-docker compose}"
RUN_INTEGRATION="${RUN_INTEGRATION:-1}"
INTEGRATION_ONLY="${INTEGRATION_ONLY:-0}"
USE_COMPOSE_SERVICES="${USE_COMPOSE_SERVICES:-1}"
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5436}"
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6382}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}✓${NC} $1"; }
fail() { echo -e "${RED}✗${NC} $1"; exit 1; }
info() { echo -e "${YELLOW}→${NC} $1"; }

echo "=== NoteHub 全体テスト ==="
echo

if [[ "$INTEGRATION_ONLY" == "1" ]]; then
  info "Skipping unit/build steps (INTEGRATION_ONLY=1)"
else
info "1/4 Backend unit tests"
(cd "$ROOT_DIR/backend" && go test ./... -count=1)
pass "Backend unit tests"

info "2/4 Frontend unit tests (Vitest)"
(cd "$ROOT_DIR/frontend" && npm test)
pass "Frontend unit tests"

info "3/4 Frontend build (typecheck + vite build)"
(cd "$ROOT_DIR/frontend" && npm run build)
pass "Frontend build"
fi

if [[ "$RUN_INTEGRATION" != "1" ]]; then
  info "Integration tests skipped (RUN_INTEGRATION=0)"
  echo
  pass "All requested tests passed"
  exit 0
fi

info "4/4 Integration smoke tests (Postgres/Redis + backend local)"
cd "$ROOT_DIR"

if [[ "$USE_COMPOSE_SERVICES" == "1" ]]; then
  $COMPOSE up -d postgres redis
fi

wait_for_tcp() {
  local host="$1"
  local port="$2"
  local attempt
  for attempt in $(seq 1 60); do
    if (echo >/dev/tcp/"$host"/"$port") >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

wait_for_tcp "$POSTGRES_HOST" "$POSTGRES_PORT" || fail "Postgres did not become ready on ${POSTGRES_HOST}:${POSTGRES_PORT}"
wait_for_tcp "$REDIS_HOST" "$REDIS_PORT" || fail "Redis did not become ready on ${REDIS_HOST}:${REDIS_PORT}"
pass "Postgres/Redis ready"

BACKEND_PID=""
BACKEND_LOG="$(mktemp)"
cleanup() {
  if [[ -n "$BACKEND_PID" ]] && kill -0 "$BACKEND_PID" 2>/dev/null; then
    kill "$BACKEND_PID" 2>/dev/null || true
    wait "$BACKEND_PID" 2>/dev/null || true
  fi
  rm -f "$BACKEND_LOG"
}
trap cleanup EXIT

set -a
if [[ -f "$ROOT_DIR/.env.local" ]]; then
  source "$ROOT_DIR/.env.local"
fi
export JWT_SECRET="${JWT_SECRET:-notehub-ci-secret-key-32bytes-minimum}"
export JWT_ISSUER="${JWT_ISSUER:-notehub-api}"
export JWT_AUDIENCE="${JWT_AUDIENCE:-notehub-client}"
export APP_ENV="${APP_ENV:-test}"
set +a

(
  cd "$ROOT_DIR/backend"
  export DATABASE_URL="${DATABASE_URL:-postgres://notehub:notehub@${POSTGRES_HOST}:${POSTGRES_PORT}/notehub?sslmode=disable}"
  export REDIS_URL="${REDIS_URL:-redis://${REDIS_HOST}:${REDIS_PORT}/0}"
  export HTTP_PORT=8082
  export PUBLIC_HTTP_URL="$BACKEND_URL"
  go run . >"$BACKEND_LOG" 2>&1
) &
BACKEND_PID=$!

wait_for_backend() {
  local attempt
  for attempt in $(seq 1 180); do
    if curl -sf "$BACKEND_URL/health" >/dev/null 2>&1; then
      return 0
    fi
    if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
      echo "--- backend log ---"
      tail -n 40 "$BACKEND_LOG" || true
      return 1
    fi
    sleep 1
  done
  echo "--- backend log ---"
  tail -n 40 "$BACKEND_LOG" || true
  return 1
}

if ! wait_for_backend; then
  fail "Local backend did not become healthy at $BACKEND_URL"
fi
pass "Backend health check ($BACKEND_URL)"

COOKIE_JAR="$(mktemp)"
RESPONSE_FILE="$(mktemp)"
trap 'rm -f "$COOKIE_JAR" "$RESPONSE_FILE"; cleanup' EXIT

CSRF_TOKEN=""

UNIQUE="$(date +%s)-$RANDOM"
EMAIL="test-${UNIQUE}@example.com"
PASSWORD="Testpass1"
NAME="Test User"

api() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local auth="${4:-}"

  local args=(-sS -o "$RESPONSE_FILE" -w "%{http_code}" -b "$COOKIE_JAR" -c "$COOKIE_JAR")
  if [[ -n "$auth" ]]; then
    args+=(-H "Authorization: Bearer $auth")
  fi
  if [[ -n "$CSRF_TOKEN" ]]; then
    args+=(-H "X-CSRF-Token: $CSRF_TOKEN")
  fi
  if [[ -n "$body" ]]; then
    args+=(-H "Content-Type: application/json" -X "$method" -d "$body")
  else
    args+=(-X "$method")
  fi
  curl "${args[@]}" "$BACKEND_URL$path"
}

assert_status() {
  local expected="$1"
  local label="$2"
  local status="$3"
  if [[ "$status" != "$expected" ]]; then
    echo "Response body:"
    cat "$RESPONSE_FILE" || true
    fail "$label (expected HTTP $expected, got $status)"
  fi
}

json_field() {
  python3 - "$1" "$2" <<'PY'
import json, sys
field = sys.argv[1]
with open(sys.argv[2], encoding="utf-8") as handle:
    data = json.load(handle)
value = data
for part in field.split("."):
    if isinstance(value, dict) and "data" in value and part not in value:
        value = value["data"]
    value = value[part]
print(value)
PY
}

info "Auth: fetch CSRF token"
status="$(api GET /api/auth/csrf)"
assert_status 200 "csrf bootstrap" "$status"
CSRF_TOKEN="$(json_field csrfToken "$RESPONSE_FILE")"

info "Auth: register + login"
status="$(api POST /api/auth/register "{\"name\":\"$NAME\",\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")"
assert_status 201 "register" "$status"

status="$(api POST /api/auth/login "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")"
assert_status 200 "login" "$status"
ACCESS_TOKEN="$(json_field accessToken "$RESPONSE_FILE")"
pass "Auth flow"

info "Workspace + document + version"
status="$(api GET /api/me "" "$ACCESS_TOKEN")"
assert_status 200 "me" "$status"

status="$(api POST /api/workspaces '{"name":"Test Workspace"}' "$ACCESS_TOKEN")"
assert_status 201 "create workspace" "$status"
WORKSPACE_ID="$(json_field id "$RESPONSE_FILE")"

status="$(api POST "/api/workspaces/$WORKSPACE_ID/documents" '{"title":"Test Doc","content":"hello"}' "$ACCESS_TOKEN")"
assert_status 201 "create document" "$status"
DOCUMENT_ID="$(json_field id "$RESPONSE_FILE")"

status="$(api GET "/api/documents/$DOCUMENT_ID" "" "$ACCESS_TOKEN")"
assert_status 200 "get document" "$status"

status="$(api POST "/api/documents/$DOCUMENT_ID/versions" "" "$ACCESS_TOKEN")"
assert_status 201 "manual version save" "$status"

status="$(api GET "/api/documents/$DOCUMENT_ID/versions" "" "$ACCESS_TOKEN")"
assert_status 200 "list versions" "$status"

pass "Workspace/document/version flow"

info "Error response includes requestId"
status="$(api GET /api/documents/00000000-0000-0000-0000-000000000000 "" "$ACCESS_TOKEN")"
assert_status 404 "missing document" "$status"
REQUEST_ID="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["error"].get("requestId",""))' "$RESPONSE_FILE")"
if [[ -z "$REQUEST_ID" ]]; then
  cat "$RESPONSE_FILE"
  fail "error response missing requestId"
fi
pass "requestId in error response ($REQUEST_ID)"

echo
pass "All tests passed"
