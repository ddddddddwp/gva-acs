# BS Connection Request Port Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish and verify the BS `oamProcess` Connection Request listener on host port 7547 without conflating it with the port 8400 Web UI.

**Architecture:** Keep lifecycle behavior in the existing `oamstart` scripts. Add a small contract test, update the documented/runtime port contract, then recreate only `gva-acs-bs` because Docker cannot add a published port to an existing container.

**Tech Stack:** Bash, Docker CLI, curl, Codex project Skill

## Global Constraints

- Preserve `/root/code/gva-acs/bs-runtime` configuration and logs.
- Keep restart policy `no` for manual startup.
- `8400:8400` is Web; `7547:7547` is CWMP Connection Request; `7458` is GVA ACS.
- Do not stage or overwrite unrelated frontend changes.

---

### Task 1: Correct and test the oamstart port contract

**Files:**
- Create: `.codex/skills/oamstart/tests/port-contract.sh`
- Modify: `.codex/skills/oamstart/scripts/start-bs-oam.sh`
- Modify: `.codex/skills/oamstart/scripts/status-bs-oam.sh`
- Modify: `.codex/skills/oamstart/SKILL.md`

**Interfaces:**
- Consumes: `gva-acs-bs` Docker lifecycle and the two existing health-check URLs.
- Produces: `BS_OAM_WEB_URL` defaulting to `http://127.0.0.1:8400` and `BS_OAM_CONNECTION_URL` defaulting to `http://127.0.0.1:7547`.

- [ ] **Step 1: Write the failing contract test**

```bash
#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
grep -Fq -- '--publish 8400:8400' "$root/scripts/start-bs-oam.sh"
grep -Fq -- '--publish 7547:7547' "$root/scripts/start-bs-oam.sh"
grep -Fq 'BS_OAM_WEB_URL:-http://127.0.0.1:8400' "$root/scripts/status-bs-oam.sh"
grep -Fq 'BS_OAM_CONNECTION_URL:-http://127.0.0.1:7547' "$root/scripts/status-bs-oam.sh"
```

- [ ] **Step 2: Run the test and verify RED**

Run: `bash .codex/skills/oamstart/tests/port-contract.sh`

Expected: nonzero because the 7547 mapping and separate Web URL do not exist.

- [ ] **Step 3: Implement the minimal contract correction**

Add `--publish 7547:7547`, split the two status URLs, require both checks for health, and document 8400 as Web and 7547 as Connection Request.

- [ ] **Step 4: Verify GREEN and static checks**

Run:

```bash
bash .codex/skills/oamstart/tests/port-contract.sh
bash -n .codex/skills/oamstart/scripts/start-bs-oam.sh
bash -n .codex/skills/oamstart/scripts/status-bs-oam.sh
python /root/.comet/skills/skills/.system/skill-creator/scripts/quick_validate.py .codex/skills/oamstart
```

Expected: every command exits 0.

### Task 2: Recreate and verify the live BS container

**Files:**
- Runtime only: Docker container `gva-acs-bs`

**Interfaces:**
- Consumes: Task 1 startup and status scripts.
- Produces: host-accessible Web port 8400 and Connection Request port 7547.

- [ ] **Step 1: Remove only the replaceable container**

Run: `docker rm -f gva-acs-bs`

Expected: container is removed; external configuration and logs remain.

- [ ] **Step 2: Recreate through the project Skill script**

Run: `bash .codex/skills/oamstart/scripts/start-bs-oam.sh`

Expected: a new `gva-acs-bs` is created with both published ports and the four vendor processes become ready.

- [ ] **Step 3: Verify ports, processes, and endpoints**

Run:

```bash
docker port gva-acs-bs
curl --noproxy '*' --fail --max-time 3 http://127.0.0.1:8400/ -o /dev/null
curl --noproxy '*' --fail --max-time 3 http://127.0.0.1:7547/ -o /dev/null
bash .codex/skills/oamstart/scripts/status-bs-oam.sh
```

Expected: 8400 and 7547 are mapped and reachable; all status checks pass, with ACS refusal allowed only as a warning.
