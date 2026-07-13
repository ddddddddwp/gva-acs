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
