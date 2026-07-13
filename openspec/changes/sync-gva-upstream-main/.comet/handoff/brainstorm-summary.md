# Brainstorm Summary

- Change: `sync-gva-upstream-main`
- Date: 2026-07-14
- Language: `zh-CN`

## 已确认事实与约束

- 同步目标是官方 Gin-Vue-Admin 的 `main`，用途为学习。
- 当前开发分支为 `V2`，必须保留其历史；不推送远程、不发布版本。
- TR-069 是本地独立设计的插件，后端、前端、注册接缝、配置和 `:7458` CWMP 服务必须保留。
- BS/UE 容器不属于本 change；仿真包、镜像和运行数据只加入忽略规则。
- OpenSpec change 文档与 change `.comet.yaml` 可跟踪；CodeGraph、生成的 Codex skills 和 Comet 本地缓存应忽略。

## 确认的技术方案

### 方案 A：保留历史的 merge（已确认）

在隔离 worktree/同步分支中添加 `upstream` remote，抓取 `upstream/main`，以 `--no-commit --no-ff` 方式合并。按“上游拥有 / 本地拥有 / 共享接缝”三类逐项解决冲突；先形成 `.gitignore` 独立提交，再创建可审计的上游合并提交。

优点是历史完整、以后可增量同步、不会重写 `V2`；代价是首次冲突可能较多。

## 未采用的方案

### 方案 B：rebase 本地提交到上游 main

提交图较线性，但会重写已发布的 `V2` 历史，冲突可能在大量本地提交上重复出现，也增加远程协作风险。

### 方案 C：以上游 main 新建基线并移植插件（备选）

可以获得最干净的上游目录结构，但容易遗漏 TR-069 之外的本地修复、配置和历史，移植面实际上比 merge 更难审计。

## 推荐方案的关键技术细节

- 合并前记录当前 HEAD、上游 HEAD、merge-base、TR-069 保护路径和共享接缝。
- 若本地与上游不存在共同祖先，停止普通 merge 并把“允许 unrelated histories”作为新的设计决策交用户确认，不自动继续。
- 专用插件目录优先保留本地语义；通用框架文件优先上游；注册、配置、依赖和部署入口人工合成。
- 不对所有冲突批量执行 `ours` 或 `theirs`。
- 许可证文件及上游文件头采用上游 BSL 1.1 声明并在交付摘要标注变化。
- `.gitignore` 使用精确路径和受限文件模式，避免忽略正常 tar/zip 源码资产或 OpenSpec 文档。

## 关键取舍与风险

- 大跨度 merge 会产生较多冲突，但比重写历史或整仓覆盖更可追溯。
- 没有文本冲突的共享初始化文件也可能静默丢失插件注册，因此需要 CodeGraph 和定向入口检查。
- 上游可能改变 Go module path、Node 包管理器或插件机制；只做维持 TR-069 编译和注册所需的最小适配，不顺带重构业务。
- 当前 `server/go.mod` 使用本地 `tr069-core-only` replace/依赖，验证环境必须保留该本地 SDK 或明确报告缺失。
- 上游许可证已变化，技术上可同步不等于可忽略使用条款；必须保留声明。

## 测试策略

- Git：检查 merge-base、未合并路径、意外删除、最终 diff 与同步 commit hash。
- 忽略规则：用 `git check-ignore -v` 对 CodeGraph、Codex skills、Comet 缓存、BS 资产和 OpenSpec 文档做正反样例。
- 后端：在 `server/` 执行 Go 格式、`go test ./...` 和 `go build ./...`；外部服务或本地 SDK 阻塞时记录证据并运行可用子集。
- 前端：按同步后的锁文件选择包管理器，执行生产构建；若上游提供 lint/test 脚本则一并运行。
- TR-069：确认插件注册调用链仍到达 `StartTR069Server`，配置保留 address，路由保留 `POST /` 与 `POST /acs`，默认监听 `:7458`。
- 运行层：若依赖环境允许，启动后端并检查 8888 与 7458 监听；否则以编译、单元测试和结构检查为最低门槛。

## Spec Patch

无。现有 delta specs 已覆盖同步来源、历史保留、插件保护、验证、许可证和本地产物卫生要求。
