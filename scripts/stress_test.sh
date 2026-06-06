#!/usr/bin/env bash
#
# Load-test the metering-service HTTP API directly with vegeta (not unit tests).
#
# Usage:
#   ./scripts/stress_test.sh
#
# Env knobs (with defaults):
#   RATE=5000          requests/sec per scenario (0 = vegeta max throughput)
#   DURATION=10s       attack duration per scenario
#   MAX_WORKERS=500    max concurrent vegeta workers (raise to push concurrency)
#   PORT=8089          port for the locally-started server
#   TARGET=            base URL of an already-running server (skips starting one)
#   GO=go              go binary to use for building/serving
#   REPORT=<repo>/stress-report.html   interactive HTML report (vegeta plot) output path
#
set -euo pipefail

RATE="${RATE:-5000}"
DURATION="${DURATION:-10s}"
MAX_WORKERS="${MAX_WORKERS:-500}"
PORT="${PORT:-8089}"
TARGET="${TARGET:-}"
GO="${GO:-go}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPORT="${REPORT:-$REPO_ROOT/stress-report.html}"
WORKDIR="$(mktemp -d)"
SERVER_PID=""
RESULT_BINS=()

cleanup() {
	[ -n "$SERVER_PID" ] && kill "$SERVER_PID" 2>/dev/null || true
	[ -n "$SERVER_PID" ] && wait "$SERVER_PID" 2>/dev/null || true
	rm -rf "$WORKDIR"
}
trap cleanup EXIT

hr() { printf '%s\n' "------------------------------------------------------------"; }

if ! command -v vegeta >/dev/null 2>&1; then
	cat >&2 <<'EOF'
vegeta is not installed. Install it with one of:

  brew install vegeta
  go install github.com/tsenart/vegeta/v12@latest   # then add $(go env GOPATH)/bin to PATH

EOF
	exit 1
fi

# start_server <REQUEST_LIMIT> <STORAGE_LIMIT_BYTES> <MAX_UPLOAD_BYTES>
start_server() {
	[ -n "$TARGET" ] && return 0
	"$GO" build -o "$REPO_ROOT/bin/metering-service" "$REPO_ROOT/cmd/apiserver"
	SERVER_ADDR="127.0.0.1:$PORT" \
		REQUEST_LIMIT="$1" STORAGE_LIMIT_BYTES="$2" MAX_UPLOAD_BYTES="$3" LOG_LEVEL=warn \
		"$REPO_ROOT/bin/metering-service" server >"$WORKDIR/server.log" 2>&1 &
	SERVER_PID=$!
	for _ in $(seq 1 50); do
		curl -sf "http://127.0.0.1:$PORT/health" >/dev/null 2>&1 && return 0
		sleep 0.1
	done
	echo "server did not become healthy; log:" >&2
	cat "$WORKDIR/server.log" >&2
	exit 1
}

stop_server() {
	[ -n "$TARGET" ] && return 0
	[ -n "$SERVER_PID" ] && kill "$SERVER_PID" 2>/dev/null || true
	[ -n "$SERVER_PID" ] && wait "$SERVER_PID" 2>/dev/null || true
	SERVER_PID=""
}

base_url() {
	if [ -n "$TARGET" ]; then printf '%s' "${TARGET%/}"; else printf 'http://127.0.0.1:%s' "$PORT"; fi
}

# Build the multipart upload body once (PNG magic bytes -> sniffs as image/png).
BOUNDARY="vegetaStress"
build_upload_body() {
	{
		printf -- '--%s\r\n' "$BOUNDARY"
		printf 'Content-Disposition: form-data; name="file"; filename="x.png"\r\n'
		printf 'Content-Type: application/octet-stream\r\n'
		printf '\r\n'
		printf '\x89PNG\r\n\x1a\n'
		head -c 256 /dev/zero
		printf '\r\n'
		printf -- '--%s--\r\n' "$BOUNDARY"
	} >"$WORKDIR/upload_body.bin"
}

# target_simple <METHOD> <path>  -> writes a one-line targets file, echoes its path
target_simple() {
	local f="$WORKDIR/t_$(echo "$2" | tr '/' '_').txt"
	printf '%s %s%s\n' "$1" "$(base_url)" "$2" >"$f"
	printf '%s' "$f"
}

# target_upload -> targets file for the multipart POST /upload
target_upload() {
	local f="$WORKDIR/t_upload.txt"
	{
		printf 'POST %s/upload\n' "$(base_url)"
		printf 'Content-Type: multipart/form-data; boundary=%s\n' "$BOUNDARY"
		printf '@%s\n' "$WORKDIR/upload_body.bin"
	} >"$f"
	printf '%s' "$f"
}

# attack <label> <targets-file>
attack() {
	local label="$1" targets="$2"
	local slug bin
	slug="$(printf '%s' "$label" | tr -cs 'A-Za-z0-9' '_')"
	bin="$WORKDIR/$slug.bin"
	hr
	printf '### %s  (rate=%s, duration=%s, max-workers=%s)\n' "$label" "$RATE" "$DURATION" "$MAX_WORKERS"
	hr
	vegeta attack -name="$label" -targets="$targets" -rate="$RATE" -duration="$DURATION" \
		-max-workers="$MAX_WORKERS" -output="$bin"
	vegeta report "$bin"
	vegeta report -type='hist[0,1ms,5ms,25ms,100ms,500ms]' "$bin"
	RESULT_BINS+=("$bin")
	echo
}

build_upload_body

# ============================================================================
# Section A — throughput (uncapped limits, expect ~100% 2xx)
# ============================================================================
echo
echo "########## SECTION A: THROUGHPUT (uncapped) ##########"
start_server 0 0 0
echo "serving at $(base_url)"
attack "GET  /health"        "$(target_simple GET  /health)"
attack "POST /api/endpoint1 (atomic counter)" "$(target_simple POST /api/endpoint1)"
attack "GET  /api/metrics"   "$(target_simple GET  /api/metrics)"
attack "POST /upload (storage channel-actor)" "$(target_upload)"
attack "GET  /storage"       "$(target_simple GET  /storage)"
stop_server

# ============================================================================
# Section B — cap enforcement under load. The two caps are tested independently
# because every upload also counts toward the request cap: if the request budget
# were exhausted first, uploads would 429 before reaching the storage check.
# ============================================================================
echo
echo "########## SECTION B: CAP ENFORCEMENT UNDER LOAD ##########"

# B1 — request cap: REQUEST_LIMIT=1000, storage unlimited.
start_server 1000 0 0
echo "B1: REQUEST_LIMIT=1000 (storage unlimited) — serving at $(base_url)"
attack "POST /api/endpoint1 past the 1000-request cap" "$(target_simple POST /api/endpoint1)"
echo "### metrics (expect total_requests == 1000 — the cap is exact under load):"
echo "  $(curl -s "$(base_url)/api/metrics")"
echo
stop_server

# B2 — storage cap: requests unlimited, small storage cap so uploads fill it then 507.
start_server 0 262144 0
echo "B2: STORAGE_LIMIT=256 KiB (requests unlimited) — serving at $(base_url)"
attack "POST /upload past the 256 KiB storage cap" "$(target_upload)"
echo "### storage (expect used at/near the cap; the overflow rejected with 507):"
echo "  $(curl -s "$(base_url)/storage")"
echo
stop_server

# ============================================================================
# Interactive HTML report — one series per scenario (vegeta plot)
# ============================================================================
hr
if [ "${#RESULT_BINS[@]}" -gt 0 ]; then
	vegeta plot -title="metering-service stress test" "${RESULT_BINS[@]}" >"$REPORT"
	echo "Interactive HTML report: $REPORT  (open in a browser)"
fi

hr
cat <<'EOF'
What to look for:
  * Section A: high success ratio + throughput on every endpoint. Compare latency of
    POST /api/endpoint1 (lock-free atomic counter) vs POST /upload (single-goroutine
    storage actor — expect higher p95/p99 as concurrent uploads serialize).
  * Section B: status codes split into 200/429 (request cap) and 201/507 (storage cap);
    /api/metrics shows total_requests == 1000 (the cap is exact under load).
EOF
