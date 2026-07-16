# GVA 上游同步流程

本项目使用三类分支维护 GVA 上游更新：

- `gva-upstream`：`flipped-aurora/gin-vue-admin` 的 `main` 纯镜像，只允许自动刷新，不放项目代码。
- `sync/gva-YYYYMMDD`：一次性集成分支，从 `dev` 创建并人工解决冲突。
- `dev`：项目开发基线；任何上游更新都必须通过人工审查后才能合入。

## 自动刷新

`.github/workflows/sync-gva-upstream.yml` 每周一台北时间 03:15 运行，也可以在 GitHub Actions 中手动触发。工作流只强制更新 `gva-upstream`，不会写入 `dev` 或 `main`。

## 人工集成到 dev

```bash
git fetch origin dev gva-upstream
git switch dev
git pull --ff-only origin dev
git switch -c sync/gva-$(date +%Y%m%d)
git merge --no-ff origin/gva-upstream
```

解决冲突并完成后端、前端及 TR-069 联调测试，再推送临时分支：

```bash
git push -u origin HEAD
```

随后在 GitHub 创建目标为 `dev` 的 Pull Request。必须人工确认改动范围、测试结果以及 TR-069 插件未被覆盖后才能合并；合并后删除对应的 `sync/gva-*` 分支。

## 禁止事项

- 不在 `gva-upstream` 上提交项目代码。
- 不把 `gva-upstream` 直接合并到 `main`。
- 不配置自动合并 `sync/gva-*` 到 `dev`。
- 不在上游同步时提交 `server/plugin/tr069/lib/tr069-core-only`；该目录由独立仓库维护并已被忽略。
