---
comet_change: sync-gva-upstream-main
role: technical-design
canonical_spec: openspec
---

# 同步 Gin-Vue-Admin 上游主分支技术设计

## 1. 设计目标

本设计用于把官方 Gin-Vue-Admin `main` 同步到当前 `V2`，保留双方 Git 历史，同时保护本地独立开发的 TR-069 插件。OpenSpec delta specs 是需求事实源；本文只定义实施结构、冲突决策、验证与回滚方法。

本次不安装 BS/UE 容器、不修改外部 `hp_ping.zip`、不新增 TR-069 功能，也不推送远程。

## 2. 当前结构与保护边界

当前仓库包含三类代码：

1. GVA 基础框架：通用后端、前端、部署、工具与文档。
2. TR-069 本地扩展：`server/plugin/tr069/`、前端插件页面/API 及其测试。
3. 共享接缝：插件注册、全局配置、菜单/路由初始化、依赖和部署端口。

同步不能只依赖文本冲突判断。上游可能在没有触碰 TR-069 专用目录的情况下改写插件注册流程，因此共享接缝必须作为独立保护面审查。

### 2.1 必须保留的行为锚点

- TR-069 插件在 GVA 插件初始化流程中被注册。
- 插件注册能够到达 `StartTR069Server`。
- TR-069 配置继续包含可配置监听地址，空值默认 `:7458`。
- CWMP 服务继续接受 `POST /` 与 `POST /acs`。
- 前端 TR-069 路由、菜单、页面和 API 导入能够通过生产构建。
- `tr069-core-only` 的本地/私有依赖关系不被上游依赖文件静默删除。

## 3. 选择的同步模型

采用“隔离 worktree + 保留历史的 merge”模型：

```text
origin/V2 ──────── local commits ────────┐
                                         ├── sync branch merge commit
upstream/main ─── upstream commits ──────┘
```

不采用 rebase，因为 `V2` 已有远程历史；不采用“复制上游文件再移植插件”，因为该方式不能可靠保留删除/改名信息，也容易遗漏 TR-069 之外的本地修复。

### 3.1 分支与提交结构

实施在隔离 worktree 的专用同步分支完成。建议形成以下可审查提交：

1. 设计与工作流产物提交：OpenSpec、handoff 和本 Design Doc。
2. 仓库卫生提交：仅包含根 `.gitignore` 的精确规则。
3. 上游 merge commit：第二父提交固定为实际 `upstream/main` 基线。
4. 必要时的兼容修复提交：只处理 merge 后验证发现且无法在 merge commit 中清晰表达的问题。

若团队偏好把冲突解决直接放进 merge commit，第 3、4 项可以合并，但不得把上游同步伪装成普通单父提交。

## 4. 实施流程

### 4.1 基线记录

在任何合并前记录：

- 当前分支、HEAD 和 `origin/V2`；
- 工作区与暂存区状态；
- remotes；
- `upstream/main` 提交哈希；
- `git merge-base HEAD upstream/main`；
- TR-069 保护路径和行为锚点。

如果不存在 merge-base，停止实施。`--allow-unrelated-histories` 会显著改变风险模型，必须作为新的设计决策交用户确认，不能自动启用。

### 4.2 Git 忽略规则

在干净的隔离分支中先提交 `.gitignore`，使后续工具运行不污染合并状态。规则按路径分组：

- `/.codegraph/`：整个本地索引目录；
- `/.codex/skills/openspec-*/`：OpenSpec 初始化生成的技能，保留已有项目规则；
- `/.comet/cache/`、`/.comet/tmp/`、`/.comet/log/`：仅机器本地状态；
- `/hp_ping/`、`/hb_ping/` 及同名压缩包/解压目录；
- 明确命名的 BS/UE Docker image tar、容器数据、仿真日志和本地覆盖配置。

不使用无边界的 `*.zip`、`*.tar` 或 `openspec/` 全局规则，以免屏蔽合法源码资产和审计文档。

使用 `git check-ignore -v` 验证正例和反例。反例至少包括 Go、Vue、YAML 文件、proposal、design、tasks、delta specs 和 change `.comet.yaml`。

### 4.3 上游 remote 与抓取

`upstream` 必须指向 `https://github.com/flipped-aurora/gin-vue-admin.git`。若 remote 已存在但 URL 不同，停止并报告，不静默改写。抓取仅针对所需分支和标签，记录最终 `upstream/main` 哈希及许可证文件内容。

### 4.4 执行 merge

在干净工作树执行 `git merge --no-commit --no-ff upstream/main`。这一步故意不自动提交，以便先检查：

- 未合并路径；
- rename/delete 冲突；
- 上游删除但本地仍依赖的文件；
- module path、包管理器和锁文件变化；
- 许可证及文档变化。

### 4.5 冲突所有权矩阵

| 文件类型 | 默认所有权 | 处理规则 |
|---|---|---|
| GVA 通用框架 | 上游 | 采用上游结构，再做最小本地兼容 |
| TR-069 专用后端/前端 | 本地 | 保留业务语义，适配新接口 |
| 插件注册与初始化 | 共享 | 人工合成并检查调用链 |
| 全局配置与配置模板 | 共享 | 保留上游字段和 TR-069 字段 |
| Go/Node 依赖与锁文件 | 共享 | 以上游基线为主，补回 TR-069 必需依赖 |
| Docker/部署端口 | 共享 | 保留上游服务结构并暴露所需 TR-069 入口 |
| 许可证与上游文件头 | 上游 | 保留 BSL 1.1 及归属声明 |
| OpenSpec/Comet artifacts | 本地 | 保留当前 change 的审计记录 |

禁止对全部冲突批量使用 `git checkout --ours`、`--theirs` 或等价策略。每个删除/改名冲突必须找到上游替代位置或记录保留原因。

### 4.6 依赖兼容策略

上游的 Go 与 Node 依赖文件作为新基础，但必须保留 TR-069 的直接依赖和 module path 兼容。若上游改变 Go module path，优先更新本地插件 import 到新模块，不通过重复模块或长期 replace 掩盖结构问题；私有 `tr069-core-only` 依赖例外，其可用性必须单独验证。

前端按照同步后锁文件选择包管理器，避免同时生成 npm、yarn、pnpm 多套锁文件。若上游已采用新的插件发现或路由机制，将 TR-069 迁移到该扩展点，不保留两套并行注册逻辑。

## 5. 验证设计

### 5.1 Git 完整性

- `git diff --check` 不得报告冲突标记或空白错误。
- `git diff --name-only --diff-filter=U` 必须为空。
- merge commit 必须包含本地与上游两个父提交。
- 审查上游删除、改名和意外的大文件加入。
- 最终摘要记录上游 commit、merge-base 和许可证变化。

### 5.2 Git 忽略规则

通过临时正反样例验证：

- 应忽略：CodeGraph DB、生成的 OpenSpec skills、Comet cache、`hp_ping`/`hb_ping`、BS image tar、仿真日志。
- 不应忽略：`server/*.go`、`web/*.vue`、YAML 模板、OpenSpec artifacts 和 `.comet.yaml`。

样例验证后删除临时文件，不把测试夹具残留在工作树。

### 5.3 Go 后端

在同步后的 `server/` 中：

1. 对实际修改的 Go 文件执行 `gofmt`。
2. 检查依赖一致性，不在未审查情况下接受 `go mod tidy` 的大范围删除。
3. 执行 `go test ./...`。
4. 执行 `go build ./...` 或上游等价构建命令。

若数据库、Redis、网络或私有 SDK 阻止完整测试，保存失败命令和错误输出，继续执行不依赖该外部条件的包级测试与编译；未运行或被阻塞的项目不能标记为通过。

### 5.4 Web 前端

读取同步后的 `package.json` 与锁文件后选择唯一包管理器，安装依赖并执行：

- 上游提供的 lint/test 命令；
- 生产构建命令；
- TR-069 页面、API 和路由导入检查。

构建产物不得进入最终提交。

### 5.5 TR-069 结构与运行检查

使用 CodeGraph 验证插件注册到 `StartTR069Server` 的调用关系，并核对配置和路由实现。若运行依赖齐全，再启动服务并检查 8888 与 7458 监听；否则结构检查、单元测试和编译构成最低验证门槛。

重点回归：

- 默认/配置监听地址；
- `POST /` 和 `POST /acs`；
- Redis dispatcher 初始化；
- 菜单、API 与数据库迁移注册；
- 前端设备、命令、告警和数据模型入口。

## 6. 错误处理与回滚

- 抓取失败：不修改代码，报告网络或认证错误后重试。
- remote URL 冲突：暂停，要求用户决定复用、改名或替换。
- 无共同祖先：中止普通 merge，回到设计决策。
- 冲突无法在当前范围解决：保持冲突清单，停止 build，不扩大为 TR-069 重构。
- 验证失败：保留失败证据并回到兼容修复任务，不创建“已完成”结论。
- 用户拒绝最终差异：删除隔离 worktree/同步分支即可恢复；不对 `V2` 执行硬重置。

已公开的历史不使用强制重写回滚。若合并已进入共享分支，使用新的 revert 提交；当前 change 不负责推送该操作。

## 7. 完成条件

当且仅当以下条件全部满足，才可把同步标记为完成：

- 官方 `main` 的具体提交已通过双父 merge 集成；
- Git 不存在未合并路径或未解释的大规模删除；
- TR-069 行为锚点全部保留；
- 后端与前端验证通过，或外部阻塞项被准确记录且最低验证门槛通过；
- 本地工具与 BS/UE 资产被忽略，OpenSpec/Comet 审计产物仍可跟踪；
- BSL 1.1 与相关归属声明被保留；
- 最终差异经过用户审查，且没有未经授权的远程推送。
