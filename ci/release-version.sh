#!/usr/bin/env bash
# release-version.sh -- Resolves release version from git tag or VERSION file.
# Responsibility: provide a single canonical version string for CI/release scripts.

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_DIR"

raw_ref="${GITHUB_REF_NAME:-}"
if [[ -n "$raw_ref" && "$raw_ref" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "${raw_ref#v}"
  exit 0
fi

if [[ -f VERSION ]]; then
  tr -d '[:space:]' < VERSION
  exit 0
fi

echo "0.0.0-dev"
