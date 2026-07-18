---
name: oamstart
description: Use when starting, resuming, checking, or diagnosing this project's BS/OAM base-station simulation stack, including gva-acs-bs container state, Web port 8400, Connection Request port 7547, Inform delivery to ACS port 7458, and OAM logs.
---

# OAM Start

## Overview

Use the bundled scripts as the source of truth for the project's Docker lifecycle and health checks. Keep the vendor image, configuration, logs, and generated simulation output outside Git.

## Workflow

1. For start or resume requests, run `scripts/start-bs-oam.sh` from this Skill directory.
2. For status, Inform, port, or log diagnosis, run `scripts/status-bs-oam.sh`.
3. Report the contract and the script result. Do not reconstruct the long `docker run` command manually.

## Runtime contract

| Item | Value |
|---|---|
| Container | `gva-acs-bs` |
| Image | `bs:5GNR_t.5.1.0.r62694M_20241219_190249` |
| Restart policy | `no` (manual start only) |
| Configuration | `/root/code/gva-acs/bs-runtime/root/hb_ping/BS_config` |
| Logs | `/root/code/gva-acs/bs-runtime/logs` |
| Required processes | `oamProcess`, `odsNameServer`, `upapp`, `m2m.x86.bs` |
| Web management | `http://127.0.0.1:8400` |
| Connection Request | `http://127.0.0.1:7547` |
| ACS target | `host.docker.internal:7458` → `172.17.0.1:7458` |

Only report the BS/OAM stack healthy when the container, restart policy, four required processes, Web port 8400, and Connection Request port 7547 pass. If `oamProcess.log` shows `Connection refused` for ACS port 7458 while those checks pass, report that BS/OAM is running and ACS is not listening yet; do not call it a BS startup failure.

## Common mistakes

- Do not use systemd inside this legacy image under WSL2/cgroup v2. The start script uses the verified foreground vendor wrapper.
- Do not remove or recreate a healthy container.
- Do not treat unrelated physical NIC, UE center, dedicated hardware, NTP, or F1 alarms as failure of the four-process OAM startup check.
- Do not commit `/root/code/gva-acs/bs-runtime` or `server/plugin/tr069/lib/tr069-core-only`.

For live logs, use:

```bash
docker logs -f gva-acs-bs
tail -f /root/code/gva-acs/bs-runtime/logs/oamProcess.log
```
