---
change: sync-gva-upstream-main
design-doc: docs/superpowers/specs/2026-07-14-sync-gva-upstream-main-design.md
base-ref: a2ef03daffe178525b8183f4e5a09cc6d0038c77
---

# 同步 Gin-Vue-Admin 上游主分支实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把官方 Gin-Vue-Admin `main` 以可追溯 merge 集成到本地 `V2`，保留 TR-069 插件，并清理本地工具及 BS/UE 资产的 Git 噪音。

**Architecture:** 在 Comet 选择的隔离分支或 worktree 中先提交精确 `.gitignore`，再添加 `upstream`、验证共同祖先并执行 `--no-commit --no-ff` merge。冲突按“上游拥有、本地拥有、共享接缝”分类处理，最后用 Git、Go、前端构建和 TR-069 路由测试验证。

**Tech Stack:** Git、Go 1.24、Gin、Vue 3、Vite、项目同步后的 Node 包管理器、CodeGraph、OpenSpec/Comet。

## Global Constraints

- 上游 MUST 是 `https://github.com/flipped-aurora/gin-vue-admin.git` 的 `main`，并记录实际 commit。
- MUST 保留 `V2` 与上游历史；禁止 rebase、强制重置、整仓覆盖和自动启用 `--allow-unrelated-histories`。
- MUST 保留 TR-069 后端、前端、注册接缝、配置、`POST /`、`POST /acs` 和默认 `:7458`。
- GVA 通用框架优先采用上游；TR-069 专用目录优先本地；共享接缝逐项人工合并。
- MUST 保留上游 BSL 1.1、版权和归属声明。
- BS/UE 只进入忽略规则；禁止加载镜像、创建容器或修改外部 `hp_ping.zip`。
- OpenSpec artifacts 与 change `.comet.yaml` MUST 可跟踪；Comet 本地 checkpoint/cache、CodeGraph DB 和生成的 Codex skills MUST 忽略。
- 未经用户授权禁止 push、发布或修改远程 `origin/V2`。

---

### Task 1: 建立 Git 忽略边界

**Files:**
- Modify: `.gitignore`
- Modify: `openspec/changes/sync-gva-upstream-main/tasks.md`

**Interfaces:**
- Consumes: Design Doc 第 4.2 节的精确忽略范围。
- Produces: 干净工作树所需的项目级忽略规则；后续 merge 和验证依赖该规则。

- [ ] **Step 1: 创建忽略规则失败样例并确认当前未命中**

在仓库根目录创建空样例路径，样例只用于 `git check-ignore`：

```bash
mkdir -p .comet/cache hb_ping/runtime
touch .comet/cache/state.json hb_ping/runtime/bs.log bs_5GNR_test.tar
git check-ignore -v .codegraph/codegraph.db .codex/skills/openspec-explore/SKILL.md .comet/cache/state.json hb_ping/runtime/bs.log bs_5GNR_test.tar
```

Expected: 命令非零退出，至少 `.codex/skills`、Comet cache、`hb_ping` 和 BS tar 尚未全部命中根 `.gitignore`。

- [ ] **Step 2: 写入受限的根忽略规则**

在 `.gitignore` 末尾增加带注释的规则，必须覆盖以下路径语义，且不得加入全局 `*.zip`、`*.tar` 或 `openspec/`：

```gitignore
# Local code intelligence and generated OpenSpec integrations
/.codegraph/
/.codex/skills/openspec-*/

# Comet machine-local runtime state
/.comet/cache/
/.comet/tmp/
/.comet/log/
/openspec/changes/*/.comet/artifacts.json
/openspec/changes/*/.comet/checkpoint.json
/openspec/changes/*/.comet/context.md
/openspec/changes/*/.comet/run-state.json
/openspec/changes/*/.comet/skill-snapshots/
/openspec/changes/*/.comet/state-events.jsonl
/openspec/changes/*/.comet/trajectory.jsonl

# External BS/UE simulation assets and runtime data
/hp_ping/
/hb_ping/
/hp_ping.zip
/hb_ping.zip
/bs_5GNR*.tar
/ue_5GNR*.tar
/**/simulation-logs/
```

- [ ] **Step 3: 验证正反样例**

Run:

```bash
git check-ignore -v .codegraph/codegraph.db .codex/skills/openspec-explore/SKILL.md .comet/cache/state.json hb_ping/runtime/bs.log bs_5GNR_test.tar
test -z "$(git check-ignore openspec/changes/sync-gva-upstream-main/proposal.md openspec/changes/sync-gva-upstream-main/.comet.yaml server/main.go web/src/main.js 2>/dev/null)"
```

Expected: 第一条为每个正例输出命中规则；第二条退出码为 0 且无输出。

- [ ] **Step 4: 删除样例并确认状态**

```bash
rm -rf .comet/cache hb_ping bs_5GNR_test.tar
git status --short --ignored
```

Expected: 本地 CodeGraph、生成 skills 和 Comet runtime 显示为 ignored；OpenSpec change 仍保持 tracked。

- [ ] **Step 5: 更新 OpenSpec task 2.1-2.3 并提交**

把 `tasks.md` 的 2.1、2.2、2.3 勾选，随后：

```bash
git add .gitignore openspec/changes/sync-gva-upstream-main/tasks.md
git commit -m "chore: ignore local tooling and simulator assets"
```

Expected: 提交仅包含 `.gitignore` 与 task 状态。

### Task 2: 记录同步基线并验证上游关系

**Files:**
- Create: `docs/upstream-sync/2026-07-14-gva-main-sync.md`
- Modify: `openspec/changes/sync-gva-upstream-main/tasks.md`

**Interfaces:**
- Consumes: 当前隔离分支 HEAD、`origin/V2` 与官方上游 URL。
- Produces: 实际上游 commit、merge-base、许可证基线和 TR-069 保护清单，供 merge 与最终审查使用。

- [ ] **Step 1: 记录当前 Git 基线**

Run:

```bash
git status --short --branch
git rev-parse HEAD
git rev-parse origin/V2
git remote -v
```

Expected: 工作树除已忽略文件外干净；当前分支是 Comet 选择的隔离分支；`origin/V2` 可解析。

- [ ] **Step 2: 添加或验证 upstream remote**

```bash
if git remote get-url upstream >/dev/null 2>&1; then
  test "$(git remote get-url upstream)" = "https://github.com/flipped-aurora/gin-vue-admin.git"
else
  git remote add upstream https://github.com/flipped-aurora/gin-vue-admin.git
fi
git fetch upstream main --tags --prune
```

Expected: `upstream/main` 可解析。若已有 URL 不同，`test` 必须失败并暂停，不得静默改写。

- [ ] **Step 3: 验证共同祖先**

```bash
UPSTREAM_HEAD=$(git rev-parse upstream/main)
MERGE_BASE=$(git merge-base HEAD upstream/main)
test -n "$UPSTREAM_HEAD"
test -n "$MERGE_BASE"
printf 'upstream=%s\nmerge-base=%s\n' "$UPSTREAM_HEAD" "$MERGE_BASE"
```

Expected: 输出两个 40 位 commit。`git merge-base` 失败时立即停止，并向用户请求是否重新设计；禁止添加 `--allow-unrelated-histories`。

- [ ] **Step 4: 创建真实基线报告**

创建 `docs/upstream-sync/2026-07-14-gva-main-sync.md`，写入命令实际输出，而不是示例值。文件必须包含：本地 base-ref、`origin/V2`、`upstream/main`、merge-base、上游 `LICENSE` 首行/许可证类型、保护路径 `server/plugin/tr069/` 与 `web/src/plugin/tr069/`，以及共享接缝 `server/initialize/`、全局配置、依赖文件和部署端口。

- [ ] **Step 5: 更新 OpenSpec task 1.1、1.3、3.1、3.2 并提交**

```bash
git add docs/upstream-sync/2026-07-14-gva-main-sync.md openspec/changes/sync-gva-upstream-main/tasks.md
git commit -m "docs: record GVA upstream sync baseline"
```

Expected: 报告中的 hash 与 `git rev-parse` 当前输出一致。

### Task 3: 合并 upstream/main 并按所有权解决冲突

**Files:**
- Modify: 上游 merge 涉及的 GVA 基础框架文件
- Preserve/Modify: `server/plugin/tr069/**`
- Preserve/Modify: `web/src/plugin/tr069/**`
- Modify: 共享注册、配置、依赖与部署文件（以实际冲突清单为准）
- Modify: `docs/upstream-sync/2026-07-14-gva-main-sync.md`
- Modify: `openspec/changes/sync-gva-upstream-main/tasks.md`

**Interfaces:**
- Consumes: Task 2 固定的 `upstream/main` commit 与保护清单。
- Produces: 双父 merge 候选，GVA 使用上游结构且 TR-069 行为锚点仍存在。

- [ ] **Step 1: 执行非自动提交 merge**

```bash
git status --porcelain
git merge --no-commit --no-ff upstream/main
```

Expected: 合并进入待提交状态；若有冲突，Git 输出冲突文件。第一条在 merge 前必须无输出。

- [ ] **Step 2: 保存冲突分类清单**

```bash
git diff --name-only --diff-filter=U
git status --short
git diff --summary HEAD
```

把未合并文件和 rename/delete 变化写入同步报告，逐项标记“上游拥有”“本地拥有”或“共享接缝”。任何未分类文件都阻止继续。

- [ ] **Step 3: 处理上游拥有文件**

对通用 GVA 文件采用 `upstream/main` 结构，再检查本地非 TR-069 修复是否仍必要。每次解决一组后，对照同步报告逐个执行 `git add --` 加入该组的真实路径，再运行：

```bash
git diff --name-only --diff-filter=U
```

Expected: 未合并列表单调减少；不得对仓库根执行批量 `--theirs`。

- [ ] **Step 4: 处理本地拥有与共享接缝**

保留 TR-069 专用目录语义，并把注册、配置、Go/Node 依赖、菜单/路由和部署端口合成到上游新结构。以下循环为每个未合并文件导出本地与上游版本到 `/tmp/gva-sync-review/`；不存在于某一侧的文件会生成对应 `.missing` 标记：

```bash
rm -rf /tmp/gva-sync-review
mkdir -p /tmp/gva-sync-review
git diff --name-only --diff-filter=U -z | while IFS= read -r -d '' file; do
  key=$(printf '%s' "$file" | sha256sum | cut -d' ' -f1)
  git show "HEAD:$file" > "/tmp/gva-sync-review/$key.local" 2>/dev/null || touch "/tmp/gva-sync-review/$key.local.missing"
  git show "upstream/main:$file" > "/tmp/gva-sync-review/$key.upstream" 2>/dev/null || touch "/tmp/gva-sync-review/$key.upstream.missing"
  printf '%s %s\n' "$key" "$file" >> /tmp/gva-sync-review/index.txt
done
```

路径在一侧不存在时使用 `git ls-tree -r --name-only` 查找上游替代位置；不得凭文件名猜测。

- [ ] **Step 5: 检查许可证和未合并项**

```bash
git diff --name-only --diff-filter=U
git diff --check
git grep -nE '^(<<<<<<<|=======|>>>>>>>)' -- . ':!docs/superpowers/plans/*'
sed -n '1,30p' LICENSE
```

Expected: 未合并列表、冲突标记搜索和 `git diff --check` 均无错误；`LICENSE`/声明与实际上游基线一致。

- [ ] **Step 6: 创建 merge commit**

先把报告中的冲突决策补全，再：

```bash
git add -A
git status --short
git commit
git show -s --format='%H%n%P%n%s' HEAD
```

提交标题使用 Git 自动生成或 `Merge upstream/main into V2 sync branch`。Expected: 父提交行包含两个 commit，第二父与 Task 2 记录的上游 commit 相同。

- [ ] **Step 7: 更新 OpenSpec task 3.3、4.1-4.5**

仅在冲突全部解决且 merge commit 双父验证通过后勾选；把 task 状态与同步报告作为后续验证提交的一部分，不修改 merge commit 的父结构。

### Task 4: 添加 TR-069 路由保护测试并验证后端

**Files:**
- Create or Modify: `server/plugin/tr069/initialize/server_test.go`
- Modify as required by upstream APIs: `server/plugin/tr069/initialize/server.go`
- Modify as required by upstream APIs: `server/plugin/tr069/plugin.go`
- Modify: `server/go.mod`
- Modify: `server/go.sum`
- Modify: `openspec/changes/sync-gva-upstream-main/tasks.md`

**Interfaces:**
- Consumes: `SetupEngine() *gin.Engine` 与 `StartTR069Server()`。
- Produces: 自动保护 `POST /`、`POST /acs` 的 Go 测试，以及可编译的同步后端。

- [ ] **Step 1: 写路由保护测试**

在 `server/plugin/tr069/initialize/server_test.go` 添加表驱动测试，调用 `SetupEngine().Routes()`，构造 `map[string]bool{method + " " + path: true}`，断言 `POST /` 和 `POST /acs` 都存在。测试不得启动监听端口或依赖数据库。

- [ ] **Step 2: 运行定向测试并记录初始结果**

```bash
cd server
go test ./plugin/tr069/initialize -run TestSetupEngineRegistersCWMPRoutes -v
```

Expected: 若 merge 保留接口则 PASS；若编译或断言失败，输出必须直接指向上游 API 兼容或路由缺失问题。

- [ ] **Step 3: 做最小兼容修复**

只修改编译、注册和路由所需代码。`SetupEngine` 仍返回 Gin engine；路由继续由同一个 CWMP handler 处理。不得在此任务重构会话、命令队列或告警业务。

- [ ] **Step 4: 验证 Go 格式、测试和构建**

```bash
cd server
gofmt -w plugin/tr069/initialize/server.go plugin/tr069/initialize/server_test.go plugin/tr069/plugin.go
go test ./plugin/tr069/initialize ./plugin/tr069/...
go test ./...
go build ./...
```

Expected: 全部通过。若私有 `tr069-core-only`、数据库或网络阻塞，保存完整错误到同步报告，运行所有不受阻塞的包级测试；不得把被阻塞命令标成通过。

- [ ] **Step 5: 使用 CodeGraph 验证注册链**

用 `codegraph_context` 查询同步后的 TR-069 注册入口，再用一次 `codegraph_explore` 检查 `Register`、`StartTR069Server` 和 `SetupEngine`。Expected: 注册链仍到达独立服务，路由和默认 `:7458` 可定位；若出现 staleness banner，只读取其列出的待同步文件。

- [ ] **Step 6: 更新 OpenSpec task 5.1、5.3 的后端部分并提交**

```bash
git add server/plugin/tr069 server/go.mod server/go.sum docs/upstream-sync/2026-07-14-gva-main-sync.md openspec/changes/sync-gva-upstream-main/tasks.md
git commit -m "test: preserve TR-069 integration after upstream sync"
```

Expected: 提交不包含无关格式化或生成二进制。

### Task 5: 验证并适配前端

**Files:**
- Preserve/Modify: `web/src/plugin/tr069/**`
- Modify as required by upstream structure: `web/src/router/**`
- Modify as required by upstream structure: `web/package.json`
- Modify exactly one upstream-selected lock file under `web/`
- Modify: `docs/upstream-sync/2026-07-14-gva-main-sync.md`
- Modify: `openspec/changes/sync-gva-upstream-main/tasks.md`

**Interfaces:**
- Consumes: 同步后的上游 Vue/Vite 插件、路由和包管理结构。
- Produces: 能被生产构建解析的 TR-069 页面、API 与路由导入。

- [ ] **Step 1: 确定唯一包管理器**

```bash
cd web
find . -maxdepth 1 -type f \( -name 'pnpm-lock.yaml' -o -name 'package-lock.json' -o -name 'yarn.lock' \) -print
node -e 'const p=require("./package.json"); console.log(p.packageManager || "unspecified", p.scripts)'
```

Expected: 按上游锁文件和 `packageManager` 选定一个工具；删除 merge 意外引入的第二套本地锁文件，不新生成额外锁文件。

- [ ] **Step 2: 安装依赖并运行上游提供的检查**

使用选定工具的 frozen/immutable 安装命令；随后只运行 `package.json` 中实际存在的 lint/test 脚本。Expected: 安装使用锁文件且不改写第二套锁文件。

- [ ] **Step 3: 运行生产构建**

```bash
cd web
if [ -f pnpm-lock.yaml ]; then
  corepack pnpm run build
elif [ -f package-lock.json ]; then
  npm run build
elif [ -f yarn.lock ]; then
  corepack yarn build
else
  echo "No supported lock file found" >&2
  exit 1
fi
```

Expected: Vite 生产构建成功，输出中无 TR-069 import、route 或 component resolution 错误。实际命令必须根据 Step 1 的工具替换，不得同时运行多套包管理器。

- [ ] **Step 4: 做最小前端兼容修复并重跑构建**

若上游改变路由或插件发现机制，把 TR-069 导入迁移到唯一的新扩展点；禁止保留旧、新两套并行注册。重新执行 Step 2 的现有检查与 Step 3 构建，Expected: 全部通过。

- [ ] **Step 5: 清理构建产物并提交**

```bash
rm -rf web/dist
git status --short
git add web docs/upstream-sync/2026-07-14-gva-main-sync.md openspec/changes/sync-gva-upstream-main/tasks.md
git commit -m "fix: adapt TR-069 web plugin to upstream GVA"
```

仅在生产构建通过后勾选 OpenSpec task 5.2；提交不得包含 `node_modules` 或 `dist`。

### Task 6: 最终审计与交付候选

**Files:**
- Modify: `docs/upstream-sync/2026-07-14-gva-main-sync.md`
- Modify: `openspec/changes/sync-gva-upstream-main/tasks.md`

**Interfaces:**
- Consumes: 所有同步、兼容和验证提交。
- Produces: 可供 Comet verify 使用的完整证据和无未解释改动的候选分支。

- [ ] **Step 1: 重跑 Git 与忽略规则检查**

```bash
git diff --check
test -z "$(git diff --name-only --diff-filter=U)"
git log --graph --oneline --decorate --max-count=20
git check-ignore -v .codegraph/codegraph.db .codex/skills/openspec-explore/SKILL.md openspec/changes/sync-gva-upstream-main/.comet/checkpoint.json
test -z "$(git check-ignore openspec/changes/sync-gva-upstream-main/proposal.md openspec/changes/sync-gva-upstream-main/.comet.yaml 2>/dev/null)"
```

Expected: 无 diff/冲突错误，正例被忽略，OpenSpec 反例未被忽略。

- [ ] **Step 2: 重跑后端与前端最终命令**

执行 Task 4 的 Go 定向测试、可用全量测试和构建，以及 Task 5 选定包管理器的生产构建。把命令、退出码、通过/阻塞状态写入同步报告。

- [ ] **Step 3: 审查变更范围和大文件**

```bash
git diff --stat a2ef03daffe178525b8183f4e5a09cc6d0038c77..HEAD
git diff --name-status a2ef03daffe178525b8183f4e5a09cc6d0038c77..HEAD
git ls-files | grep -E '(^|/)(hp_ping|hb_ping)(/|$)|bs_5GNR.*\.tar$|ue_5GNR.*\.tar$' && exit 1 || true
```

Expected: 无 BS/UE 资产被跟踪；所有大范围删除和目录迁移在同步报告中有解释。

- [ ] **Step 4: 完成报告和剩余 OpenSpec tasks**

同步报告必须包含：上游 commit、merge-base、merge commit 两个父、冲突分类摘要、TR-069 保留点、许可证变化、验证命令/结果、外部阻塞和未授权 push 声明。完成证据后勾选 tasks.md 对应的 1.2、5.4、6.1、6.2；6.3 保留到用户选择分支收尾时完成。

- [ ] **Step 5: 提交最终审计**

```bash
git add docs/upstream-sync/2026-07-14-gva-main-sync.md openspec/changes/sync-gva-upstream-main/tasks.md
git commit -m "docs: record upstream sync verification"
git status --short --branch
```

Expected: 除已忽略的机器本地状态外工作树干净；不执行 `git push`。
