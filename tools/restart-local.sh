#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
BACKEND_DIR="$ROOT_DIR/server"
WEB_DIR="$ROOT_DIR/web"

kill_pid() {
  local pid="${1:-}"
  if [[ -z "$pid" ]]; then return 0; fi
  if ! kill -0 "$pid" >/dev/null 2>&1; then return 0; fi
  kill "$pid" >/dev/null 2>&1 || true
  for _ in {1..30}; do
    if ! kill -0 "$pid" >/dev/null 2>&1; then return 0; fi
    sleep 0.1
  done
  kill -9 "$pid" >/dev/null 2>&1 || true
}

kill_port() {
  local port="$1"
  local pids
  pids="$(ss -ltnp "sport = :${port}" 2>/dev/null | awk -F'pid=' 'NR>1{print $2}' | awk -F',' '{print $1}' | sort -u || true)"
  if [[ -z "$pids" ]]; then return 0; fi
  while read -r pid; do
    [[ -n "$pid" ]] && kill_pid "$pid"
  done <<< "$pids"
}

# Close ports before restart
for port in 8888 7458 8080; do
  kill_port "$port"
done

# Start backend
(
  cd "$BACKEND_DIR"
  env GOWORK=off go run . -c "$BACKEND_DIR/config.yaml"
) &

# Start frontend
(
  cd "$WEB_DIR"
  npm run serve
) &

wait
