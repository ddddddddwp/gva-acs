# Local Ports and GVA Upstream Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Avoid OnCall port collisions and maintain a weekly pure mirror of official GVA without automatically merging it into `dev`.

**Architecture:** Local development uses host ports `18080` for Web and `18888` for the GVA API. Docker keeps its internal `8080` and `8888` ports but exposes them as `18080` and `18888`. A scheduled GitHub Actions workflow force-with-lease updates only `gva-upstream`; integration into `dev` happens through a human-reviewed `sync/gva-*` branch.

**Tech Stack:** Bash, YAML, GitHub Actions, Docker Compose, Vite, Go configuration.

## Global Constraints

- OnCall-reserved ports must not be used as GVA host defaults.
- TR-069 ACS remains on `7458`; BS Connection Request remains on `8400`.
- `gva-upstream` must contain the exact official `upstream/main` commit.
- Automation must never merge or push to `dev` or `main`.

---

### Task 1: Add a policy regression check

**Files:**
- Create: `scripts/ci/check-project-policy.sh`

- [ ] Assert local Web/API defaults are `18080` and `18888`.
- [ ] Assert Docker host mappings are `18080:8080` and `18888:8888`.
- [ ] Assert the weekly workflow targets only `gva-upstream`.
- [ ] Run `bash scripts/ci/check-project-policy.sh` and observe failure against the old defaults.

### Task 2: Move conflicting host defaults

**Files:**
- Modify: `web/.env.development`
- Modify: `server/config.yaml`
- Modify: `restart-dev.sh`
- Modify: `tools/restart-local.sh`
- Modify: `deploy/docker-compose/docker-compose.yaml`
- Modify: GVA local documentation and MCP local defaults that reference the API endpoint.

- [ ] Change local Web from `8080` to `18080`.
- [ ] Change local API from `8888` to `18888`.
- [ ] Keep Docker container ports unchanged.
- [ ] Re-run the policy check; the port section should pass.

### Task 3: Add the upstream mirror workflow and operator guide

**Files:**
- Create: `.github/workflows/sync-gva-upstream.yml`
- Create: `docs/development/gva-upstream-sync.md`

- [ ] Schedule a weekly refresh and provide `workflow_dispatch`.
- [ ] Fetch official `flipped-aurora/gin-vue-admin` `main`.
- [ ] Push only to `refs/heads/gva-upstream` using force-with-lease.
- [ ] Document the manual `sync/gva-YYYYMMDD` to `dev` review path.
- [ ] Run the policy check and YAML syntax checks.

### Task 4: Verify and publish

- [ ] Run `bash scripts/ci/check-project-policy.sh`.
- [ ] Run focused Go tests and the frontend build.
- [ ] Commit and push the implementation to `dev`.
- [ ] Point local and remote `gva-upstream` at the fetched official `upstream/main` commit.
