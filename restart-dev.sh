#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

BACKEND_DIR="$ROOT_DIR/server"
WEB_DIR="$ROOT_DIR/web"

BACKEND_LOG="/tmp/gva-backend.log"
WEB_LOG="/tmp/gva-web.log"

BACKEND_PID_FILE="/tmp/gva-backend.pid"
WEB_PID_FILE="/tmp/gva-web.pid"

kill_pid() {
  local pid="$1"
  if [[ -z "${pid}" ]]; then
    return 0
  fi
  if ! kill -0 "${pid}" >/dev/null 2>&1; then
    return 0
  fi
  kill "${pid}" >/dev/null 2>&1 || true
  for _ in {1..30}; do
    if ! kill -0 "${pid}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.1
  done
  kill -9 "${pid}" >/dev/null 2>&1 || true
}

kill_port() {
  local port="$1"
  local pids
  pids="$(ss -ltnp "sport = :${port}" 2>/dev/null | awk -F'pid=' 'NR>1 {print $2}' | awk -F',' '{print $1}' | sort -u || true)"
  if [[ -z "${pids}" ]]; then
    return 0
  fi
  while read -r pid; do
    if [[ -n "${pid}" ]]; then
      kill_pid "${pid}"
    fi
  done <<< "${pids}"
}

kill_from_pidfile() {
  local file="$1"
  if [[ -f "${file}" ]]; then
    local pid
    pid="$(cat "${file}" 2>/dev/null || true)"
    rm -f "${file}" || true
    if [[ -n "${pid}" ]]; then
      kill_pid "${pid}"
    fi
  fi
}

kill_from_pidfile "${BACKEND_PID_FILE}"
kill_from_pidfile "${WEB_PID_FILE}"

kill_port 8888
kill_port 7458
kill_port 8080

mkdir -p /tmp >/dev/null 2>&1 || true
rm -f "${BACKEND_LOG}" "${WEB_LOG}" >/dev/null 2>&1 || true

(
  cd "${BACKEND_DIR}"
  if [[ "${1:-}" == "--no-tee" ]]; then
    nohup env GOWORK=off go run . -c "${BACKEND_DIR}/config.yaml" >"${BACKEND_LOG}" 2>&1 &
  else
    stdbuf -oL -eL env GOWORK=off go run . -c "${BACKEND_DIR}/config.yaml" 2>&1 | tee -a "${BACKEND_LOG}" &
  fi
  echo $! > "${BACKEND_PID_FILE}"
)

(
  cd "${WEB_DIR}"
  if [[ "${1:-}" == "--no-tee" ]]; then
    nohup npm run dev >"${WEB_LOG}" 2>&1 &
  else
    stdbuf -oL -eL npm run dev 2>&1 | tee -a "${WEB_LOG}" &
  fi
  echo $! > "${WEB_PID_FILE}"
)

echo "backend: http://localhost:8888 (log: ${BACKEND_LOG})"
echo "tr069:   http://localhost:7458"
echo "web:     http://localhost:8080 (log: ${WEB_LOG})"

if [[ "${1:-}" == "--follow" ]]; then
  echo "follow:  tail -f ${BACKEND_LOG}"
  tail -f "${BACKEND_LOG}"
fi
