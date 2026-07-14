# oamstart Project Skill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a project-level `oamstart` Skill that starts, resumes, and diagnoses the repository's BS/OAM Docker simulation stack through a repeatable, safe workflow.

**Architecture:** Keep operational knowledge in `.codex/skills/oamstart/SKILL.md` and put deterministic Docker operations in two Bash scripts. `start-bs-oam.sh` owns prerequisite checks and idempotent container lifecycle handling, then delegates health verification to `status-bs-oam.sh`; runtime images, configuration, and logs remain outside the Git repository.

**Tech Stack:** Codex project skills, Bash, Docker CLI, curl, Git

## Global Constraints

- Do not rebuild, remove, or replace the currently healthy `gva-acs-bs` container during validation.
- Do not add `/root/code/gva-acs/bs-runtime`, image layers, logs, generated simulation output, or `tr069-core-only` to Git.
- Treat `172.17.0.1:7458 Connection refused` as an ACS readiness warning, not a BS/OAM startup failure.
- A successful health result requires the container, restart policy, four named processes, and port 8400 to be healthy.
- Use `apply_patch` for hand-authored repository changes and the skill-creator initializer for the initial directory scaffold.

---

### Task 1: Establish the no-Skill baseline (RED)

**Files:**
- Create: `docs/superpowers/validation/2026-07-14-oamstart-baseline.md`

**Step 1: Run a clean-room baseline scenario**

Start a fresh subagent without conversation history and without access to the not-yet-created `oamstart` Skill. Ask it to explain how it would recover this project's BS/OAM stack if `gva-acs-bs` did not exist, including exact runtime paths, the direct wrapper needed for WSL2/cgroup v2, health criteria, and the meaning of ACS port 7458 refusing connections. The scenario must be read-only and must not modify Docker state.

**Step 2: Record the observed gaps**

Create `docs/superpowers/validation/2026-07-14-oamstart-baseline.md` containing:

- the exact prompt;
- the response or a faithful concise transcript;
- which required facts were missing or unsafe;
- the expected behavior after the Skill exists.

The baseline is RED only if the no-Skill response cannot reliably reconstruct at least one of the exact container contract, external runtime paths, direct-wrapper startup chain, or project-specific health interpretation. If it unexpectedly satisfies every requirement, stop and strengthen the test scenario before authoring the Skill.

**Step 3: Confirm the baseline artifact is isolated**

Run:

```bash
git diff --check
git status --short
```

Expected: only the baseline validation document is new at this point, with no Docker or runtime artifacts under the repository.

**Step 4: Commit the RED baseline**

```bash
git add docs/superpowers/validation/2026-07-14-oamstart-baseline.md
git commit -m "test: capture oamstart skill baseline"
```

---

### Task 2: Initialize the project Skill

**Files:**
- Create: `.codex/skills/oamstart/SKILL.md`
- Create: `.codex/skills/oamstart/agents/openai.yaml`
- Create: `.codex/skills/oamstart/scripts/`

**Step 1: Generate the scaffold with skill-creator**

Run:

```bash
python /root/.comet/skills/skills/.system/skill-creator/scripts/init_skill.py \
  oamstart \
  --path .codex/skills \
  --resources scripts \
  --interface display_name="OAM Start" \
  --interface short_description="启动并检查项目 BS/OAM 仿真协议栈" \
  --interface default_prompt='Use $oamstart to start and verify the project BS/OAM simulation stack.'
```

Expected: the initializer creates `.codex/skills/oamstart/`, including `agents/openai.yaml` and the requested `scripts/` directory.

**Step 2: Replace generated guidance with the project contract**

Write `.codex/skills/oamstart/SKILL.md` with YAML frontmatter containing only:

```yaml
---
name: oamstart
description: Use when starting, resuming, checking, or diagnosing this project's BS/OAM base-station simulation stack, including gva-acs-bs container state, Connection Request port 8400, Inform delivery to ACS port 7458, and OAM logs.
---
```

The body must direct the agent to:

1. run `scripts/start-bs-oam.sh` for start/resume requests;
2. run `scripts/status-bs-oam.sh` for status, Inform, port, and log diagnosis;
3. preserve the fixed container/image/path contract from the approved design;
4. interpret ACS connection refusal as a warning when BS health checks pass;
5. report the container, image, restart policy, processes, Connection Request URL, ACS target, and log commands;
6. avoid deleting unrelated containers or committing external runtime assets.

Keep the Skill concise and link to its scripts instead of duplicating the long Docker command.

**Step 3: Inspect generated interface metadata**

Run:

```bash
sed -n '1,120p' .codex/skills/oamstart/agents/openai.yaml
```

Expected: `display_name`, `short_description`, and a `default_prompt` explicitly naming `$oamstart` are present and correctly quoted.

---

### Task 3: Implement deterministic startup and status scripts (GREEN)

**Files:**
- Create: `.codex/skills/oamstart/scripts/start-bs-oam.sh`
- Create: `.codex/skills/oamstart/scripts/status-bs-oam.sh`

**Step 1: Implement `status-bs-oam.sh`**

Use `set -euo pipefail` and overridable environment defaults for:

- `BS_OAM_CONTAINER=gva-acs-bs`;
- `BS_OAM_LOG_DIR=/root/code/gva-acs/bs-runtime/logs`;
- `BS_OAM_WAIT_SECONDS=60`.

The script must poll only up to the configured timeout, then check:

- Docker CLI and daemon availability;
- container existence and running state;
- restart policy equals `unless-stopped`;
- `oamProcess`, `odsNameServer`, `upapp`, and `m2m.x86.bs` exist inside the container;
- `http://127.0.0.1:8400` responds at the TCP/HTTP layer;
- logs identify the ACS target as `172.17.0.1:7458` or `host.docker.internal:7458`.

Accumulate required-check failures and exit nonzero if any required health item fails. Print an explicit warning—but retain success—when logs show port 7458 connection refusal. End with the two operator commands:

```text
docker logs -f gva-acs-bs
tail -f /root/code/gva-acs/bs-runtime/logs/oamProcess.log
```

**Step 2: Implement `start-bs-oam.sh` prerequisites and lifecycle branches**

Use `set -euo pipefail` and overridable environment defaults for the fixed container, image, runtime root, configuration directory, and log directory.

Implement these branches:

- running container: print that it is already running and do not recreate it;
- stopped container: call `docker start gva-acs-bs`;
- missing container: require the exact image and external config assets, create the log directory, then run the container.

Before the create branch, require these configuration entries:

```text
m2mcfg.xml
packtCtrlCfg.xml
simulation_config_bs.txt
protStackCfg.sh
dataplane_env
```

Create the container with:

- `--privileged` and `--restart unless-stopped`;
- bridge networking, `host.docker.internal:host-gateway`, and `-p 8400:8400`;
- `/run` and `/run/lock` tmpfs mounts plus writable cgroup mount;
- read-only configuration mount and writable log mount;
- `SIMULATION_IP=192.168.10.212`, `SIMULATION_PORT=65432`, and `BSID=12345abc`;
- a direct `/bin/bash -lc` wrapper that copies configs, runs `ssh-keygen -A`, starts `/usr/sbin/sshd`, changes into the vendor application directory, and `exec`s `./start_macadapter_bs.sh` in the foreground.

Do not invoke systemd and do not remove any container automatically. After every successful lifecycle branch, execute the sibling `status-bs-oam.sh`.

**Step 3: Make scripts executable and run static checks**

Run:

```bash
chmod +x .codex/skills/oamstart/scripts/start-bs-oam.sh \
  .codex/skills/oamstart/scripts/status-bs-oam.sh
bash -n .codex/skills/oamstart/scripts/start-bs-oam.sh
bash -n .codex/skills/oamstart/scripts/status-bs-oam.sh
```

Expected: both syntax checks exit 0.

**Step 4: Validate the Skill package**

Run:

```bash
python /root/.comet/skills/skills/.system/skill-creator/scripts/quick_validate.py \
  .codex/skills/oamstart
```

Expected: validation succeeds without frontmatter, naming, description, or interface errors.

---

### Task 4: Verify live behavior without rebuilding the healthy container

**Files:**
- Modify only if verification exposes an issue: `.codex/skills/oamstart/SKILL.md`
- Modify only if verification exposes an issue: `.codex/skills/oamstart/scripts/start-bs-oam.sh`
- Modify only if verification exposes an issue: `.codex/skills/oamstart/scripts/status-bs-oam.sh`

**Step 1: Capture the current container identity**

Run:

```bash
before_id="$(docker inspect -f '{{.Id}}' gva-acs-bs)"
before_started="$(docker inspect -f '{{.State.StartedAt}}' gva-acs-bs)"
printf '%s\n%s\n' "$before_id" "$before_started"
```

Expected: both values are present.

**Step 2: Run the status script against the live stack**

Run:

```bash
.codex/skills/oamstart/scripts/status-bs-oam.sh
```

Expected: exit 0; all four processes, restart policy, and port 8400 pass. If ACS is still down, the output reports connection refusal only as a warning.

**Step 3: Prove startup idempotency**

Run:

```bash
.codex/skills/oamstart/scripts/start-bs-oam.sh
after_id="$(docker inspect -f '{{.Id}}' gva-acs-bs)"
after_started="$(docker inspect -f '{{.State.StartedAt}}' gva-acs-bs)"
test "$before_id" = "$after_id"
test "$before_started" = "$after_started"
```

Expected: the script reports that the container is already running, health checks pass, and both equality checks exit 0, proving no recreation or restart occurred.

**Step 4: Run the post-Skill scenario (GREEN)**

Start a fresh subagent and explicitly provide `$oamstart`. Re-run the same read-only recovery/diagnosis prompt from the baseline. Confirm its response uses the scripts, exact project contract, direct-wrapper rationale, health criteria, and correct ACS-refusal interpretation. Append the result to the baseline document under a `GREEN` section.

**Step 5: Refactor only from observed evidence**

If static, live, or scenario checks fail, make the smallest correction and repeat Steps 2–4. Do not add speculative options or general Docker documentation.

---

### Task 5: Final verification, commit, and push `dev`

**Files:**
- Verify: `.codex/skills/oamstart/SKILL.md`
- Verify: `.codex/skills/oamstart/agents/openai.yaml`
- Verify: `.codex/skills/oamstart/scripts/start-bs-oam.sh`
- Verify: `.codex/skills/oamstart/scripts/status-bs-oam.sh`
- Verify: `docs/superpowers/validation/2026-07-14-oamstart-baseline.md`

**Step 1: Run the complete verification set**

```bash
bash -n .codex/skills/oamstart/scripts/start-bs-oam.sh
bash -n .codex/skills/oamstart/scripts/status-bs-oam.sh
python /root/.comet/skills/skills/.system/skill-creator/scripts/quick_validate.py \
  .codex/skills/oamstart
.codex/skills/oamstart/scripts/status-bs-oam.sh
git diff --check
```

Expected: every command exits 0. A clearly labeled ACS connection-refused warning is acceptable.

**Step 2: Audit repository boundaries**

Run:

```bash
git status --short
git check-ignore -v server/plugin/tr069/lib/tr069-core-only \
  /root/code/gva-acs/bs-runtime 2>/dev/null || true
git ls-files | rg '(^|/)(bs-runtime|tr069-core-only)(/|$)' && exit 1 || true
```

Expected: only intended Skill, plan, and validation files are candidates for commit; no external runtime or local TR-069 core files are tracked.

**Step 3: Review the implementation diff**

Run:

```bash
git diff -- .codex/skills/oamstart docs/superpowers/validation
git log --oneline --decorate -5
```

Expected: the Skill matches the approved design and the baseline commit precedes the implementation commit.

**Step 4: Commit implementation artifacts**

```bash
git add .codex/skills/oamstart \
  docs/superpowers/validation/2026-07-14-oamstart-baseline.md
git commit -m "feat: add oamstart project skill"
```

**Step 5: Push the completed work**

```bash
git push origin dev
```

Expected: `origin/dev` advances to include the design, implementation plan, RED baseline, and final Skill commits.
