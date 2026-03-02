#!/usr/bin/env bash
# coverage.sh - compute project-wide cross-package coverage excluding example/ packages.
# Only a single number (cross-package aggregated coverage) is printed.
#
# Usage:
#   scripts/coverage.sh
#   PROFILE=custom.out scripts/coverage.sh
#   COVER_MODE=count scripts/coverage.sh
#
# Env vars:
#   PROFILE     coverprofile path (default: coverage.out)
#   COVER_MODE  covermode (default: atomic)
#   VERBOSE=1   print underlying go test output (otherwise suppressed)
set -euo pipefail

PROFILE=${PROFILE:-coverage.out}
COVER_MODE=${COVER_MODE:-atomic}

# Collect packages excluding example/ (any path segment named 'example'). Works on macOS bash 3.2.
PKGS=""
while IFS= read -r pkg; do
  [ -n "${pkg}" ] || continue
  PKGS+=" ${pkg}"
done < <(go list ./... | grep -v '/example/')

if [ -z "${PKGS// /}" ]; then
  echo "No packages found" >&2
  exit 1
fi

# Trim leading space and convert to array safely for later loops.
PKGS_TRIMMED=${PKGS# }

# Build comma-separated list for -coverpkg (instrument all packages so tests in one package
# can contribute coverage for code in another).
COVERPKG=$(echo "${PKGS_TRIMMED}" | tr ' ' ',')

COUNT_PKGS=$(echo "${PKGS_TRIMMED}" | tr ' ' '\n' | wc -l | tr -d ' ')

echo "Running cross-package coverage across ${COUNT_PKGS} packages..." >&2

LOGFILE=$(mktemp 2>/dev/null || mktemp -t graphql_cover)
if GOFLAGS="${GOFLAGS:-}" go test -covermode="${COVER_MODE}" -coverpkg="${COVERPKG}" -coverprofile="${PROFILE}" ${PKGS_TRIMMED} >"${LOGFILE}" 2>&1; then
  :
else
  echo "go test failed; showing output:" >&2
  cat "${LOGFILE}" >&2
  rm -f "${LOGFILE}"
  exit 1
fi

[ "${VERBOSE:-}" != "" ] && cat "${LOGFILE}" >&2
rm -f "${LOGFILE}"

TOTAL_PERCENT=$(go tool cover -func="${PROFILE}" | awk '/total:/ {print $3}')
if [ -z "${TOTAL_PERCENT}" ]; then
  echo "ERROR: could not parse total coverage from ${PROFILE}" >&2
  exit 2
fi

echo "${TOTAL_PERCENT}"  # single number output
