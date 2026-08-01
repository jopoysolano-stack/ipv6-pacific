#!/usr/bin/env bash
# Start the collector for a region.
# Usage:
#   ./scripts/start_collector.sh -run-once
#   ./scripts/start_collector.sh pacific -run-once
#   ./scripts/start_collector.sh caribbean -run-once -country=JM
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ ! -f go.mod ]] || [[ ! -d cmd/collector ]]; then
  echo "Run from project root (go.mod and cmd/collector required)."
  exit 1
fi

# shellcheck source=region_env.sh
source "$(dirname "$0")/region_env.sh"
region_env_setup "collector" "$@"
set -- "${REGION_ENV_REMAINING_ARGS[@]+"${REGION_ENV_REMAINING_ARGS[@]}"}"

echo "Starting collector REGION=${REGION} DATA_DIR=${DATA_DIR} $*"
exec go run ./cmd/collector/ "$@"
