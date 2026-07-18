#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

grep -Fq -- '--publish 8400:8400' "$root/scripts/start-bs-oam.sh"
grep -Fq -- '--publish 7547:7547' "$root/scripts/start-bs-oam.sh"
grep -Fq 'BS_OAM_WEB_URL:-http://127.0.0.1:8400' "$root/scripts/status-bs-oam.sh"
grep -Fq 'BS_OAM_CONNECTION_URL:-http://127.0.0.1:7547' "$root/scripts/status-bs-oam.sh"
grep -Fq 'Web endpoint responds at' "$root/scripts/status-bs-oam.sh"
grep -Fq 'Connection Request endpoint responds at' "$root/scripts/status-bs-oam.sh"
