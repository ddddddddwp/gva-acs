#!/usr/bin/env bash
set -euo pipefail

container="${BS_OAM_CONTAINER:-gva-acs-bs}"
runtime_root="${BS_OAM_RUNTIME_ROOT:-/root/code/gva-acs/bs-runtime}"
config_dir="${BS_OAM_CONFIG_DIR:-${runtime_root}/root/hb_ping/BS_config}"
log_dir="${BS_OAM_LOG_DIR:-${runtime_root}/logs}"
wait_seconds="${BS_OAM_WAIT_SECONDS:-60}"
connection_url="${BS_OAM_CONNECTION_URL:-http://127.0.0.1:8400}"
required_processes=(oamProcess odsNameServer upapp m2m.x86.bs)

pass() { printf '[PASS] %s\n' "$*"; }
warn() { printf '[WARN] %s\n' "$*"; }
fail() { printf '[FAIL] %s\n' "$*" >&2; }

if ! command -v docker >/dev/null 2>&1; then
  fail "Docker CLI is not installed or not in PATH."
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  fail "Docker daemon is unavailable."
  exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
  fail "curl is required to check ${connection_url}."
  exit 1
fi

if ! [[ "$wait_seconds" =~ ^[0-9]+$ ]]; then
  fail "BS_OAM_WAIT_SECONDS must be a non-negative integer: ${wait_seconds}"
  exit 1
fi

if ! docker container inspect "$container" >/dev/null 2>&1; then
  fail "Container ${container} does not exist. Run start-bs-oam.sh first."
  exit 1
fi

stack_ready() {
  local process

  [ "$(docker inspect -f '{{.State.Running}}' "$container" 2>/dev/null || true)" = "true" ] || return 1
  for process in "${required_processes[@]}"; do
    docker exec "$container" pgrep -f -- "$process" >/dev/null 2>&1 || return 1
  done
  curl -sS --max-time 3 -o /dev/null "$connection_url" >/dev/null 2>&1 || return 1
}

deadline=$((SECONDS + wait_seconds))
until stack_ready; do
  if (( SECONDS >= deadline )); then
    break
  fi
  sleep 2
done

failures=0
running="$(docker inspect -f '{{.State.Running}}' "$container" 2>/dev/null || true)"
if [ "$running" = "true" ]; then
  pass "Container ${container} is running."
else
  fail "Container ${container} is not running."
  failures=$((failures + 1))
fi

restart_policy="$(docker inspect -f '{{.HostConfig.RestartPolicy.Name}}' "$container" 2>/dev/null || true)"
if [ "$restart_policy" = "unless-stopped" ]; then
  pass "Restart policy is unless-stopped."
else
  fail "Restart policy is '${restart_policy:-unknown}', expected unless-stopped."
  failures=$((failures + 1))
fi

for process in "${required_processes[@]}"; do
  if [ "$running" = "true" ] && docker exec "$container" pgrep -f -- "$process" >/dev/null 2>&1; then
    pass "Process ${process} is running."
  else
    fail "Process ${process} is not running."
    failures=$((failures + 1))
  fi
done

if curl -sS --max-time 3 -o /dev/null "$connection_url" >/dev/null 2>&1; then
  pass "Connection Request endpoint responds at ${connection_url}."
else
  fail "Connection Request endpoint is unavailable at ${connection_url}."
  failures=$((failures + 1))
fi

oam_log="${log_dir}/oamProcess.log"
acs_config="${config_dir}/simulation_config_bs.txt"
if { [ -r "$acs_config" ] && grep -Eq 'host\.docker\.internal:7458|172\.17\.0\.1:7458' "$acs_config"; } || \
   { [ -r "$oam_log" ] && grep -Eq 'host\.docker\.internal|172\.17\.0\.1:7458' "$oam_log"; }; then
  pass "ACS target is host.docker.internal:7458 (172.17.0.1:7458)."
else
  warn "Could not confirm the expected ACS target in configuration or OAM logs."
fi

if [ -r "$oam_log" ] && grep -Eq "Connection refused|errno\[111\]" "$oam_log"; then
  warn "ACS 172.17.0.1:7458 is refusing connections; BS/OAM health is evaluated separately."
fi

printf '\nLive logs:\n'
printf '  docker logs -f %s\n' "$container"
printf '  tail -f %s/oamProcess.log\n' "$log_dir"

if (( failures > 0 )); then
  fail "BS/OAM stack has ${failures} required health-check failure(s)."
  exit 1
fi

printf '\nBS/OAM simulation stack is healthy.\n'
