#!/usr/bin/env bash
# End-to-end smoke test: build, start server, verify HTTP endpoints, shut down.
# Usage: ./scripts/e2e-smoke.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/bin/e2e-server"
CONF="$ROOT/configs"
HTTP_PORT="${E2E_HTTP_PORT:-18000}"
GRPC_PORT="${E2E_GRPC_PORT:-19000}"

cleanup() {
  if [[ -n "${SERVER_PID:-}" ]]; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
  rm -f "$BIN"
}
trap cleanup EXIT

echo "==> building server..."
(cd "$ROOT" && go build -o "$BIN" ./cmd/server)

echo "==> starting server (HTTP=$HTTP_PORT, gRPC=$GRPC_PORT)..."
APP_SERVER_HTTP_ADDR="0.0.0.0:$HTTP_PORT" \
APP_SERVER_GRPC_ADDR="0.0.0.0:$GRPC_PORT" \
"$BIN" -conf "$CONF" &
SERVER_PID=$!

echo "==> waiting for server to become ready..."
for i in $(seq 1 30); do
  if curl -sf "http://127.0.0.1:$HTTP_PORT/healthz" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    echo "FAIL: server exited before becoming ready"
    exit 1
  fi
  sleep 0.5
done

echo "==> testing /healthz..."
HEALTHZ=$(curl -sf "http://127.0.0.1:$HTTP_PORT/healthz")
echo "    $HEALTHZ"
echo "$HEALTHZ" | jq -e '.status == "ok"' >/dev/null || { echo "FAIL: healthz"; exit 1; }

echo "==> testing /readyz..."
READYZ=$(curl -sf "http://127.0.0.1:$HTTP_PORT/readyz")
echo "    $READYZ"
echo "$READYZ" | jq -e '.status == "ready"' >/dev/null || { echo "FAIL: readyz"; exit 1; }

echo "==> testing /meta..."
META=$(curl -sf "http://127.0.0.1:$HTTP_PORT/meta")
echo "    $META"
echo "$META" | jq -e '.status == "running"' >/dev/null || { echo "FAIL: meta"; exit 1; }

echo "==> testing greeter HTTP endpoint (GET /helloworld/e2e)..."
REPLY=$(curl -sf "http://127.0.0.1:$HTTP_PORT/helloworld/e2e")
echo "    $REPLY"
echo "$REPLY" | jq -e '.message == "Hello e2e"' >/dev/null || { echo "FAIL: greeter"; exit 1; }

echo ""
echo "==> all e2e smoke checks passed"
