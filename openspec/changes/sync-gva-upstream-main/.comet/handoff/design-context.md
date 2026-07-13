# Comet Design Handoff

- Change: sync-gva-upstream-main
- Phase: design
- Mode: compact
- Context hash: 8de6bf8ed88633e9b8f5af745394b0fd6f6b9f240b21b0c9540c1a4af8b006d3

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/sync-gva-upstream-main/proposal.md

- Source: openspec/changes/sync-gva-upstream-main/proposal.md
- Lines: 1-31
- SHA256: 4398ecedb4a97b72fff4a1613bff70951295cc54af553bdd7af1ac15015b687e

```md
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

```

## openspec/changes/sync-gva-upstream-main/design.md

- Source: openspec/changes/sync-gva-upstream-main/design.md
- Lines: 1-89
- SHA256: eb0e921047f5b559739918298771ffcb5c611cf6bb38eb8630b3c420281350b3

[TRUNCATED]

```md
## Context

当前 `V2` 分支的 `origin` 指向 `ddddddddwp/gva-acs`，项目由 Gin-Vue-Admin 基础框架和独立开发的 TR-069 插件组成。TR-069 后端位于 `server/plugin/tr069/`，插件注册会启动独立 CWMP Gin 服务并监听 `:7458`；前端也有对应插件页面和 API。官方 Gin-Vue-Admin 的 `main` 已继续演进，且许可证已变更为 BSL 1.1。

本次同步横跨后端、前端、依赖与部署配置。同步必须保留当前分支历史，避免把 TR-069 专用实现当作普通上游冲突覆盖，同时要清除 CodeGraph、Codex/OpenSpec 生成项、Comet 缓存和外部 BS 仿真资产造成的工作区噪音。

## Goals / Non-Goals

**Goals:**

- 将官方 Gin-Vue-Admin `main` 的完整基础框架变更同步到当前 `V2`，保留可追溯的上游来源和合并历史。
- 保留 TR-069 后端、前端、插件注册、配置和 `:7458` CWMP 接口，并使其适配同步后的框架。
- 建立明确、可复查的冲突处理规则，完成后端和前端验证。
- 让本地数据库、缓存、生成技能、仿真镜像与运行数据不再出现在 Git 待提交列表中，同时保留 OpenSpec change 文档。
- 保留上游代码携带的许可证、版权和归属声明。

**Non-Goals:**

- 不安装、加载或运行 BS/UE 容器，也不修改外部 `hp_ping.zip` 的内容。
- 不新增或重新设计 TR-069 业务功能、数据模型或用户界面。
- 不使用 rebase 重写 `V2` 历史，不通过强制覆盖替换当前仓库。
- 不在本次 change 中推送远程分支或发布版本。

## Decisions

### 1. 使用上游 remote + merge commit 同步

添加名为 `upstream` 的 remote，抓取 `upstream/main`，在隔离工作区内执行非自动提交合并。确认冲突处理和验证结果后再产生合并提交。

选择 merge 而不是 rebase，是因为 `V2` 已有公开远程历史，重写会增加协作风险；选择 merge 而不是复制上游工作树，是为了保留文件来源、删除记录和后续增量同步能力。

### 2. 按所有权处理冲突

冲突分为三类：

- **上游拥有**：通用 GVA 框架、基础 UI、依赖与构建工具优先采用上游版本，再应用必要兼容修正。
- **本地拥有**：`server/plugin/tr069/`、前端 TR-069 插件目录及其专用测试优先保留本地实现。
- **共享接缝**：插件注册、全局配置、菜单/路由初始化、依赖清单和 Docker 端口必须人工合并；结果既要符合上游结构，也要继续注册 TR-069 并监听 `:7458`。

每个删除/改名冲突必须确认上游替代路径，不能仅以 `ours` 或 `theirs` 批量解决全部冲突。

### 3. 先建立保护清单，再处理合并冲突

合并前记录 TR-069 关键目录、注册入口、配置字段和验证命令。合并后通过路径检查、CodeGraph 结构查询以及构建/测试确认扩展点仍存在。这样可以把“插件没有文本冲突但注册被上游改写”的静默回归纳入检查。

### 4. Git 忽略规则区分可审计产物与机器本地状态

根 `.gitignore` 将忽略：

- `.codegraph/` 本地索引数据库及状态；
- OpenSpec 初始化生成的 `.codex/skills/`；
- `.comet/cache/`、`.comet/tmp/`、`.comet/log/` 等机器本地缓存；
- `hp_ping`、`hb_ping`、相关 zip/tar 镜像归档与解压运行目录；
- 本地容器数据、日志、临时文件和私有覆盖配置。

`openspec/changes/`、change 内 `.comet.yaml` 和团队需要的项目规则不被忽略。忽略规则应尽量使用精确路径或受限模式，避免屏蔽正常源码归档和项目配置。

### 5. 许可证随同步来源保留

同步上游 `main` 时保留其 BSL 1.1 许可证、文件头和归属声明，不把当前 Apache 2.0 文件强行覆盖到新上游代码。交付摘要明确记录许可证基线发生变化，便于后续使用者判断适用范围。

### 6. 分层验证同步结果

验证顺序为：Git 冲突和未合并项检查、Go 格式化/依赖/测试或编译、前端依赖安装和构建、TR-069 注册与 `:7458` 入口结构检查、`.gitignore` 命中测试。若上游自身测试依赖外部服务，报告环境限制并至少完成可离线的编译与静态检查。

## Risks / Trade-offs

- **上游跨度较大导致冲突数量多** → 在隔离 worktree/分支操作，先生成冲突清单，按所有权逐项处理并保留可回滚基线。
- **共享初始化文件无文本冲突但插件失效** → 使用 CodeGraph 检查插件注册调用关系，并验证 CWMP 服务入口仍包含 `/`、`/acs` 和 `:7458`。
- **依赖大版本升级破坏本地代码** → 以同步后的上游依赖文件为基线，针对 TR-069 编译错误做最小兼容适配，不顺带重构业务。
- **前端上游目录或路由机制变化** → 将 TR-069 页面作为本地拥有模块迁移到新扩展点，并通过生产构建确认导入完整。
- **BS 归档忽略模式过宽** → 优先忽略明确目录与大文件命名，使用 `git check-ignore` 验证正常源码和 OpenSpec 文档未被误屏蔽。
- **许可证变化影响后续用途** → 保留 BSL 1.1 声明并在交付文档中显著说明；本次不提供商业使用判断。

## Migration Plan

1. 保存当前提交与工作区状态，创建隔离工作区或专用同步分支。
2. 完成 `.gitignore` 精确规则并验证 OpenSpec artifacts 仍可跟踪。
3. 添加/校验 `upstream` remote，抓取官方 `main` 并记录同步提交哈希。
4. 执行非自动提交合并，生成冲突与改动清单。

```

Full source: openspec/changes/sync-gva-upstream-main/design.md

## openspec/changes/sync-gva-upstream-main/tasks.md

- Source: openspec/changes/sync-gva-upstream-main/tasks.md
- Lines: 1-38
- SHA256: 610b758b4b5f5d362556432409a5a23686e2dd4394616ac79e7e45f15871d6dc

```md
## 1. 同步前基线与隔离

- [ ] 1.1 记录当前 `V2` 提交、工作区状态、现有 remotes 和可回滚基线
- [ ] 1.2 建立隔离 worktree/同步分支，并确认用户现有改动未被带入或覆盖
- [ ] 1.3 记录 TR-069 后端、前端、插件注册、配置和 `:7458` CWMP 入口保护清单

## 2. 仓库本地产物卫生

- [ ] 2.1 更新根 `.gitignore`，精确忽略 `.codegraph/`、生成的 `.codex/skills/` 和 Comet 本地缓存/临时/日志目录
- [ ] 2.2 添加 `hp_ping`/`hb_ping`、BS/UE 镜像归档、解压运行目录、容器数据与仿真日志忽略规则
- [ ] 2.3 使用 `git check-ignore` 验证本地资产被忽略，同时 OpenSpec artifacts、`.comet.yaml` 和代表性源码未被误伤

## 3. 获取并合并上游主分支

- [ ] 3.1 添加或校验 `upstream` remote 指向官方 Gin-Vue-Admin 仓库
- [ ] 3.2 抓取 `upstream/main` 并记录本次同步的准确提交哈希与许可证基线
- [ ] 3.3 以非自动提交方式合并 `upstream/main`，收集全部冲突、删除、改名和依赖变化

## 4. 解决冲突并适配 TR-069

- [ ] 4.1 对通用 GVA 框架文件采用上游结构，逐项处理后端、前端、部署和文档冲突
- [ ] 4.2 保留 `server/plugin/tr069/` 与前端 TR-069 专用目录，并修复上游依赖/API 变化导致的兼容问题
- [ ] 4.3 人工合并插件注册、全局配置、菜单/路由初始化、依赖清单和 Docker 端口等共享接缝
- [ ] 4.4 确认 TR-069 插件仍启动独立 CWMP 服务，保留 `:7458`、`POST /` 和 `POST /acs`
- [ ] 4.5 保留上游 BSL 1.1、版权和文件归属声明，并移除所有未合并路径

## 5. 后端与前端验证

- [ ] 5.1 对修改的 Go 文件执行格式化，并完成 Go 依赖一致性、测试和编译验证
- [ ] 5.2 使用上游同步后的包管理配置安装前端依赖并完成 lint/生产构建中可用的验证项
- [ ] 5.3 使用 CodeGraph 和定向测试检查 TR-069 注册调用链、配置结构及关键入口
- [ ] 5.4 记录因数据库、Redis、网络或其他外部依赖未能运行的验证项，不将其误报为通过

## 6. 差异审查与交付

- [ ] 6.1 审查最终 Git 差异、上游同步哈希、冲突决策和未跟踪文件，确认没有 BS 资产或本地缓存进入变更
- [ ] 6.2 汇总许可证变化、保留的 TR-069 扩展点、验证结果和已知限制供用户审查
- [ ] 6.3 在用户确认后完成本地分支收尾；未经授权不推送远程或发布版本

```

## openspec/changes/sync-gva-upstream-main/specs/local-artifact-hygiene/spec.md

- Source: openspec/changes/sync-gva-upstream-main/specs/local-artifact-hygiene/spec.md
- Lines: 1-29
- SHA256: 2b25c0a0c8bbdb5687c466ebeb13ff9e0f148841d7b45fe303fe9e0c6a5ca0d1

```md
## ADDED Requirements

### Requirement: 忽略机器本地工具状态
版本控制规则 MUST 忽略 CodeGraph 数据库、OpenSpec 初始化生成的 Codex skills 和 Comet 的缓存、临时及日志目录。

#### Scenario: 本地工具生成状态文件
- **WHEN** CodeGraph、Codex/OpenSpec 或 Comet 在项目中生成机器相关状态
- **THEN** 这些文件不会出现在 Git 待提交列表中

### Requirement: 忽略外部基站仿真资产
版本控制规则 MUST 忽略 `hp_ping`/`hb_ping` 仿真包、Docker 镜像 tar、解压运行目录、容器数据和仿真日志。

#### Scenario: 将 BS 包放入项目工作区
- **WHEN** 开发者为本地测试复制或解压 BS 仿真资产
- **THEN** 大型归档、镜像、运行配置副本和日志不会被 Git 跟踪

### Requirement: 保留可审计工作流产物
版本控制规则 MUST 允许 OpenSpec change 文档及 change 内 `.comet.yaml` 被 Git 跟踪。

#### Scenario: Comet 创建 change artifacts
- **WHEN** Comet 生成 proposal、design、specs、tasks 或 change 状态文件
- **THEN** 这些可审计产物不会被 `.gitignore` 屏蔽

### Requirement: 忽略规则不得误伤源码
新增忽略规则 MUST 使用精确路径或受限模式，并通过 `git check-ignore` 验证正常源码、配置模板和 OpenSpec 文档仍可跟踪。

#### Scenario: 验证忽略模式
- **WHEN** `.gitignore` 更新完成
- **THEN** 本地工具及仿真样例路径被命中，而代表性的 Go、Vue、YAML 和 OpenSpec 文件不被命中

```

## openspec/changes/sync-gva-upstream-main/specs/upstream-framework-sync/spec.md

- Source: openspec/changes/sync-gva-upstream-main/specs/upstream-framework-sync/spec.md
- Lines: 1-51
- SHA256: d9542c91231029867cd996fe4e48d306d3f0323b4b8155a17132b0c25fb02af7

```md
## ADDED Requirements

### Requirement: 可追溯的上游来源
同步流程 MUST 使用 `https://github.com/flipped-aurora/gin-vue-admin.git` 的 `main` 分支作为上游来源，并记录实际同步的提交标识。

#### Scenario: 获取上游主分支
- **WHEN** 执行框架同步
- **THEN** 仓库存在可识别的上游 remote，且同步基线能够对应到官方 `main` 的具体提交

### Requirement: 保留当前分支历史
同步流程 MUST 通过保留历史的合并方式集成上游，不得通过整仓覆盖、强制重置或改写已发布 `V2` 历史完成同步。

#### Scenario: 集成上游变更
- **WHEN** 上游变更被引入当前开发分支
- **THEN** 本地历史和上游历史均可从 Git 提交图追溯

### Requirement: 保护 TR-069 本地扩展
同步结果 MUST 保留 TR-069 后端插件、前端功能、插件注册、配置字段和独立 CWMP 服务入口。

#### Scenario: 上游同步后启动 TR-069 插件
- **WHEN** 同步后的 GVA 应用注册业务插件
- **THEN** TR-069 插件仍被注册，CWMP 服务仍可配置为监听 `:7458`，并保留 `/` 与 `/acs` 请求入口

#### Scenario: TR-069 专用文件发生冲突
- **WHEN** 上游合并与 TR-069 专用目录或代码产生冲突
- **THEN** 冲突处理保留本地业务语义，并仅做适配新框架所必需的修改

### Requirement: 共享接缝人工审查
插件注册、全局配置、路由/菜单初始化、依赖清单和部署端口等共享接缝 MUST 逐项审查，不得使用单一 `ours` 或 `theirs` 策略批量解决全部冲突。

#### Scenario: 初始化文件发生冲突
- **WHEN** 上游修改与本地插件注册位于同一共享文件
- **THEN** 合并结果同时包含上游框架结构和 TR-069 所需注册行为，并留下可审查差异

### Requirement: 同步结果可验证
同步结果 MUST 通过适用于当前环境的后端编译/测试、前端生产构建、TR-069 结构检查和 Git 未合并项检查。

#### Scenario: 同步完成候选交付
- **WHEN** 所有合并冲突已处理
- **THEN** Git 不存在未合并路径，后端与前端验证成功，且 TR-069 关键扩展点仍可定位

#### Scenario: 外部服务阻止完整测试
- **WHEN** 某项测试因数据库、Redis 或网络等外部依赖无法运行
- **THEN** 必须记录具体限制，并完成不依赖该外部服务的编译、静态或单元验证，不得将未运行描述为通过

### Requirement: 保留上游许可证声明
同步结果 MUST 保留官方 `main` 所携带的 BSL 1.1 许可证、版权和文件归属声明。

#### Scenario: 同步许可证文件
- **WHEN** 上游许可证与本地原许可证不同
- **THEN** 交付结果采用并保留上游适用声明，且在同步摘要中明确记录许可证变化

```
