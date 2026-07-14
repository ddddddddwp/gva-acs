# oamstart Skill RED/GREEN Validation

## RED: no-Skill baseline

### Prompt

The fresh agent received no conversation history and was forbidden from reading repository files, inspecting Docker, or modifying state:

> 在 `/root/code/gva-acs/gva-acs` 项目中，如果 `gva-acs-bs` 容器不存在，请恢复并启动 BS/OAM 基站仿真协议栈。请给出精确的容器镜像、仓库外运行目录和配置/日志路径、适配 WSL2/cgroup v2 的直接启动 wrapper、健康判据、Connection Request 地址，并解释 ACS 7458 端口 Connection refused 是否意味着 BS 启动失败。

### Observed response

The agent refused to invent project-specific values, which was safe, but it could not provide an executable recovery procedure. It explicitly reported that it could not determine:

- the complete image name and tag;
- external runtime, configuration, and log paths;
- container-side mount destinations and original startup parameters;
- the Connection Request address;
- whether the image should use systemd, a foreground process, or an entrypoint.

It offered only a placeholder Docker wrapper containing unresolved values such as `<完整镜像名:tag或@digest>` and `<仓库外绝对运行目录>`. The suggested generic flags included `--cgroupns=host`, while the verified project container instead needs a direct foreground vendor startup command, writable cgroup mount, two tmpfs mounts, configuration copies, SSH initialization, and `start_macadapter_bs.sh`.

The agent correctly separated BS health from ACS readiness and stated that ACS port 7458 refusing a TCP connection does not by itself mean that BS/OAM failed to start.

### RED result

The control failed the executable-recovery requirement because it lacked all of these required project facts:

- image `bs:5GNR_t.5.1.0.r62694M_20241219_190249`;
- runtime root `/root/code/gva-acs/bs-runtime` and its configuration/log paths;
- exact direct-wrapper command and mounts required under WSL2/cgroup v2;
- required processes `oamProcess`, `odsNameServer`, `upapp`, and `m2m.x86.bs`;
- Connection Request URL `http://127.0.0.1:8400`;
- ACS target `172.17.0.1:7458` through `host.docker.internal`.

The post-Skill GREEN run must use the bundled scripts and recover these details without guessing or rebuilding an already healthy container.

## GREEN: Skill-enabled forward test

### Prompt

A fresh agent was given only the path to `$oamstart` and the same recovery/diagnosis request. It could read `SKILL.md` and its bundled scripts but could not run Docker, execute the scripts, inspect other repository files, or modify state.

### Observed response

The agent directed the user to run:

```bash
./.codex/skills/oamstart/scripts/start-bs-oam.sh
./.codex/skills/oamstart/scripts/status-bs-oam.sh
```

It accurately recovered the container, exact image tag, restart policy, external runtime/config/log paths, five required configuration files, WSL2/cgroup v2 direct-wrapper approach, four health processes, `http://127.0.0.1:8400`, and the `host.docker.internal:7458` to `172.17.0.1:7458` ACS mapping.

It also preserved the required interpretation: when the container, restart policy, four processes, and port 8400 pass, a port 7458 `Connection refused`/`errno[111]` message means ACS is not listening and does not mean BS/OAM startup failed. It told the user not to reconstruct `docker run` manually and did not suggest replacing a healthy container.

### GREEN result

PASS. The Skill closed every project-specific gap observed in the RED control without requiring leaked conversation context or live-state inspection. No new loophole or unsafe rationalization appeared, so no refactor was required.
