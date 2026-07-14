# oamstart 项目级 Skill 设计

## 目标

创建项目级 Skill `oamstart`，让 AI 在收到“启动 OAM”“启动 BS 基站协议栈”“检查 BS Inform”“查看 OAM 日志”等请求时，使用本项目已验证的 Docker 启动与诊断流程。

## 结构

```text
.codex/skills/oamstart/
├── SKILL.md
├── agents/openai.yaml
└── scripts/
    ├── start-bs-oam.sh
    └── status-bs-oam.sh
```

`SKILL.md` 记录触发条件、运行资产位置、容器契约、操作顺序、验证标准和常见故障。脚本承担容易出错的长 Docker 命令和状态检查，避免 AI 每次重新拼装命令。

## 固定运行契约

- 容器：`gva-acs-bs`
- 镜像：`bs:5GNR_t.5.1.0.r62694M_20241219_190249`
- 重启策略：`unless-stopped`
- 配置目录：`/root/code/gva-acs/bs-runtime/root/hb_ping/BS_config`
- 日志目录：`/root/code/gva-acs/bs-runtime/logs`
- Connection Request：`http://127.0.0.1:8400`
- ACS：`172.17.0.1:7458`，由 `host.docker.internal:7458` 解析得到
- 健康进程：`oamProcess`、`odsNameServer`、`upapp`、`m2m.x86.bs`

镜像、BS 配置、日志和仿真产物保持在 Git 仓库外，不加入 Skill 目录或主仓库历史。

## 启动行为

启动脚本必须幂等：容器已运行时不重建；容器已停止时先尝试启动；容器不存在时检查镜像和配置目录，再使用已验证的启动链创建容器。容器启动链复制只读配置、启动 `sshd`，随后以前台方式运行原厂 `start_macadapter_bs.sh`，绕过旧 CentOS systemd 与 WSL2/cgroup v2 的兼容问题。

状态脚本检查容器运行状态、重启策略、四个健康进程、8400 HTTP 端口和 OAM 日志中的 ACS 连接目标。ACS 尚未监听 7458 时，`Connection refused` 只表示管理端未启动，不判定 BS/OAM 启动失败。

## 安全和错误处理

- 不自动删除其他名称的容器，不修改 Git 仓库内业务代码。
- 缺少镜像、配置目录或 Docker 时明确失败并给出缺失项。
- 不把不存在的物理网卡、UE 仿真中心或专用硬件告警误判为 OAM 进程启动失败。
- 只有四个关键进程和 8400 端口均正常时，才报告 BS/OAM 协议栈已启动。

## 验证

对脚本执行 `bash -n`；运行 Skill 校验器；在现有容器上执行启动脚本验证幂等性，执行状态脚本验证关键进程与端口。验证过程中不重建当前健康容器。
