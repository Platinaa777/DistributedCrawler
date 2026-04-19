#!/usr/bin/env bash
# Build Go binaries for one or all components.
#
# Usage:
#   ./build.sh                      # build all components
#   ./build.sh grpc_server          # build specific component(s)
#   ./build.sh fetch_worker parser_worker
#
# Environment variables:
#   BIN_DIR   – output directory (default: <project_root>/bin)
#   LDFLAGS   – extra linker flags (default: empty)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
BIN_DIR="${BIN_DIR:-${PROJECT_ROOT}/bin}"

COMPONENTS=(
  grpc_server
  fetch_worker
  parser_worker
  export_worker
  scheduler_worker
  memory_broker
)

build() {
  local name="$1"
  local src="${PROJECT_ROOT}/cmd/${name}/main.go"

  if [[ ! -f "$src" ]]; then
    echo "ERROR: entry point not found: ${src}" >&2
    exit 1
  fi

  echo "==> Building ${name}..."
  go build ${LDFLAGS:+-ldflags "$LDFLAGS"} -o "${BIN_DIR}/${name}" "${src}"
  echo "    → ${BIN_DIR}/${name}"
}

mkdir -p "${BIN_DIR}"

if [[ $# -gt 0 ]]; then
  for arg in "$@"; do
    found=false
    for name in "${COMPONENTS[@]}"; do
      if [[ "$name" == "$arg" ]]; then
        build "$name"
        found=true
        break
      fi
    done
    if [[ "$found" == false ]]; then
      echo "ERROR: Unknown component '$arg'." >&2
      echo "Available: ${COMPONENTS[*]}" >&2
      exit 1
    fi
  done
else
  for name in "${COMPONENTS[@]}"; do
    build "$name"
  done
fi

echo ""
echo "==> Build complete. Binaries in: ${BIN_DIR}/"
