#!/usr/bin/env bash
# Start the HTTPS web server for a region.
# Usage:
#   ./scripts/start_server.sh              # default: pacific
#   ./scripts/start_server.sh pacific
#   ./scripts/start_server.sh caribbean
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ ! -f go.mod ]] || [[ ! -d cmd/web ]]; then
  echo "Run from project root (go.mod and cmd/web required)."
  exit 1
fi

# shellcheck source=region_env.sh
source "$(dirname "$0")/region_env.sh"
region_env_setup "web" "$@"
set -- "${REGION_ENV_REMAINING_ARGS[@]+"${REGION_ENV_REMAINING_ARGS[@]}"}"

echo "Starting HTTPS web server REGION=${REGION} DATA_DIR=${DATA_DIR} LISTEN=${LISTEN} (needs certs/ — run ./scripts/gen_dev_certs.sh)..."
exec go run ./cmd/web/
