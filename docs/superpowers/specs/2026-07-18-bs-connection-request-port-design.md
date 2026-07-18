# BS Connection Request 端口修复设计

## 目标

修正 BS/OAM 本地容器的端口职责，使 GVA 能通过 CPE 上报的 `Device.ManagementServer.ConnectionRequestURL` 主动唤醒基站。

## 端口契约

- `8400:8400`：BS Web 管理页面，由容器内 `thttpd` 监听。
- `7547:7547`：TR-069 Connection Request，由容器内 `oamProcess` 监听。
- `7458`：GVA ACS，由 BS 通过 `http://host.docker.internal:7458/acs` 主动发送 Inform。
- 本地宿主机运行 GVA 时，BS 上报 `http://127.0.0.1:7547`。

## 实施

启动脚本同时发布 8400 和 7547。状态脚本分别验证 Web 地址与 Connection Request 地址，并继续检查容器、手动启动策略、四个 OAM 进程及 ACS 目标。项目 Skill 文档同步区分两个入口。

现有容器无法在线增加 Docker 端口映射，因此停止并重建 `gva-acs-bs`。配置和日志位于宿主机 `/root/code/gva-acs/bs-runtime`，重建不得删除这些目录。

## 验证

- Shell 语法和 Skill 包校验通过。
- Docker 显示 `8400:8400` 与 `7547:7547`。
- `http://127.0.0.1:8400` 和 `http://127.0.0.1:7547` 均可访问。
- `oamProcess`、`odsNameServer`、`upapp`、`m2m.x86.bs` 均运行。
- ACS 未监听 7458 时仅报告警告，不误判为 BS 启动失败。
