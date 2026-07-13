## Why

当前项目长期基于 Gin-Vue-Admin 开发，但尚未配置可重复的上游同步方式，基础框架更新与本地 TR-069 插件改动混杂，直接覆盖代码容易丢失插件入口或引入不可控冲突。现在需要以官方 `main` 为同步源，在保留本地插件能力和历史的前提下完成一次可审计的框架同步，并阻止本地工具、仿真包和运行数据被误提交。

## What Changes

- 将 `https://github.com/flipped-aurora/gin-vue-admin.git` 配置为上游来源，并把其 `main` 分支以保留历史的合并方式同步到当前 `V2` 分支。
- 对合并冲突逐项审查：GVA 基础框架优先采用上游实现；TR-069 后端插件、前端页面、插件注册点及独立的 `:7458` CWMP 服务必须保留并适配上游结构。
- 对上游依赖、配置、构建脚本和目录结构变化进行兼容调整，避免通过整仓覆盖或历史重写完成同步。
- 更新项目根 `.gitignore`，忽略 CodeGraph 本地数据库、OpenSpec/Codex 生成技能、Comet 本地缓存，以及 `hp_ping`/`hb_ping` 仿真压缩包、Docker 镜像归档、解压运行目录、日志和临时数据。
- 保持 OpenSpec change 文档和 change 内 `.comet.yaml` 可跟踪，以便恢复工作流和审计同步决策。
- 保留并遵循同步进来的上游版权及 BSL 1.1 声明；本次用途为学习。
- 本次不安装或运行 BS/UE 容器，不推送远程，不重新设计 TR-069 业务功能。

## Capabilities

### New Capabilities

- `upstream-framework-sync`: 定义从 Gin-Vue-Admin `main` 同步基础框架、保留本地历史和保护 TR-069 扩展点的要求。
- `local-artifact-hygiene`: 定义本地工具状态、Comet 缓存和 BS/UE 仿真资产不得进入版本控制，同时保留可审计 OpenSpec 产物的要求。

### Modified Capabilities

无。

## Impact

- 可能影响除专用 TR-069 业务目录外的整个 GVA 基础框架，包括 `server/`、`web/`、`deploy/`、依赖锁文件、构建配置和项目文档。
- 必须重点保护 `server/plugin/tr069/`、前端 TR-069 插件目录、插件初始化/注册代码、配置结构和 CWMP `:7458` 监听入口。
- 上游当前许可证与本项目原有 Apache 2.0 基线不同；同步代码和文档时必须保留新的许可证及归属声明。
- 合并可能产生大量文本、依赖和结构冲突，需要在隔离分支或 worktree 中执行并完成 Go 与前端验证后再交付。
