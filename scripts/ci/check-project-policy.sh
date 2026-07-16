#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

assert_line() {
  local file="$1"
  local expected="$2"

  if ! grep -Fqx "$expected" "$file"; then
    echo "policy check failed: $file must contain exact line: $expected" >&2
    exit 1
  fi
}

assert_contains() {
  local file="$1"
  local expected="$2"

  if ! grep -Fq -- "$expected" "$file"; then
    echo "policy check failed: $file must contain: $expected" >&2
    exit 1
  fi
}

assert_not_contains() {
  local file="$1"
  local forbidden="$2"

  if grep -Fq -- "$forbidden" "$file"; then
    echo "policy check failed: $file must not contain: $forbidden" >&2
    exit 1
  fi
}

assert_line web/.env.development "VITE_CLI_PORT = 18080"
assert_line web/.env.development "VITE_SERVER_PORT = 18888"
assert_contains server/config.yaml "    addr: 18888"
assert_contains deploy/docker-compose/docker-compose.yaml "'18080:8080'"
assert_contains deploy/docker-compose/docker-compose.yaml "'18888:8888'"

workflow=.github/workflows/sync-gva-upstream.yml
assert_contains "$workflow" "cron: '15 19 * * 0'"
assert_contains "$workflow" "refs/heads/gva-upstream"
assert_not_contains "$workflow" "refs/heads/dev"
assert_not_contains "$workflow" "refs/heads/main:refs/heads"

echo "project policy check passed"
