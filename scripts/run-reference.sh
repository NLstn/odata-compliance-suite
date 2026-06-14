#!/usr/bin/env bash
#
# run-reference.sh — run the compliance suite against the go-odata reference
# implementation (NLstn/go-odata, cmd/complianceserver).
#
# It builds and starts the reference compliance server, runs this suite against
# it, then tears the server down. The suite's exit code is propagated, so this
# script is suitable for CI gating.
#
# Any extra arguments are passed straight through to the suite, e.g.:
#   scripts/run-reference.sh -verbose
#   scripts/run-reference.sh -version 4.0 -pattern filter
#
# Environment overrides:
#   GO_ODATA_REPO  git URL of the reference implementation
#                  (default: https://github.com/NLstn/go-odata)
#   GO_ODATA_REF   branch/tag/commit to check out (default: main)
#   GO_ODATA_DIR   use an existing local checkout instead of cloning
#                  (not cleaned up on exit)
#   SERVER_PORT    port for the reference server (default: 9090)
set -euo pipefail

GO_ODATA_REPO="${GO_ODATA_REPO:-https://github.com/NLstn/go-odata}"
GO_ODATA_REF="${GO_ODATA_REF:-main}"
SERVER_PORT="${SERVER_PORT:-9090}"
SERVER_URL="${SERVER_URL:-http://localhost:${SERVER_PORT}}"

suite_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cloned=0
if [ -n "${GO_ODATA_DIR:-}" ]; then
	ref_dir="$(cd "$GO_ODATA_DIR" && pwd)"
	echo "Using existing go-odata checkout: $ref_dir"
else
	ref_dir="$(mktemp -d)"
	cloned=1
	echo "Cloning ${GO_ODATA_REPO}@${GO_ODATA_REF} -> ${ref_dir}"
	git clone --depth 1 --branch "$GO_ODATA_REF" "$GO_ODATA_REPO" "$ref_dir"
fi

server_pid=""
cleanup() {
	if [ -n "$server_pid" ] && kill -0 "$server_pid" 2>/dev/null; then
		echo "Stopping reference server (pid ${server_pid})"
		kill "$server_pid" 2>/dev/null || true
		wait "$server_pid" 2>/dev/null || true
	fi
	if [ "$cloned" = "1" ]; then
		rm -rf "$ref_dir"
	fi
}
trap cleanup EXIT

# Build a native binary so we control the server's PID directly (`go run`
# spawns a child process that a kill on the parent would not reach).
server_bin="${ref_dir}/complianceserver.bin"
echo "Building reference compliance server..."
(cd "${ref_dir}/cmd/complianceserver" && go build -o "$server_bin" .)

echo "Starting reference compliance server on port ${SERVER_PORT}..."
server_log="${ref_dir}/server.log"
"$server_bin" -port "$SERVER_PORT" >"$server_log" 2>&1 &
server_pid=$!

echo "Running compliance suite against ${SERVER_URL}"
cd "$suite_dir"
set +e
go run . -server "$SERVER_URL" "$@"
status=$?
set -e

if [ "$status" -ne 0 ]; then
	echo "----- reference server log (tail) -----"
	tail -n 50 "$server_log" 2>/dev/null || true
	echo "----------------------------------------"
fi

exit "$status"
