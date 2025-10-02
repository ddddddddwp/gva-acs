# GitHub 仓库配置文档

## 仓库信息

**目标仓库**: `https://github.com/ddddddddwp/gva-tr069`  
**目标分支**: `V1`  
**仓库所有者**: `ddddddddwp`  
**仓库名称**: `gva-tr069`

## 项目结构

本项目包含两个主要组件：

### TR069-Core (核心库)
- **路径**: `server/plugin/tr069-core/`
- **性质**: 独立的TR069协议实现库
- **功能**: 提供完整的TR069/CWMP协议栈

### TR069-Adapter (GVA适配器)
- **路径**: `server/plugin/tr069-adapter/`
- **性质**: GVA框架插件
- **功能**: 将TR069-Core集成到GVA管理系统

## 推送配置

### 文件包含范围
推送到GitHub的文件应包括：

```
server/plugin/tr069-core/          # TR069核心库
server/plugin/tr069-adapter/       # GVA适配器插件
├── TR069_ARCHITECTURE.md          # 架构设计文档
├── GITHUB_CONFIG.md               # 本配置文档
└── README.md                      # 项目说明文档
```

### 排除文件
以下文件不应推送到GitHub：
- 编译生成的二进制文件
- 临时文件和缓存
- 敏感配置信息
- IDE特定文件

## Git 操作命令

### 初始化和添加远程仓库
```bash
# 在项目根目录执行
git init
git remote add origin https://github.com/ddddddddwp/gva-tr069.git
```

### 推送到V1分支
```bash
# 添加文件
git add server/plugin/tr069-core/
git add server/plugin/tr069-adapter/

# 提交更改
git commit -m "feat: 添加TR069核心库和GVA适配器插件"

# 推送到V1分支
git push -u origin V1
```

### 后续更新
```bash
# 添加更改的文件
git add .

# 提交更改
git commit -m "update: 更新TR069功能"

# 推送到V1分支
git push origin V1
```

## 分支策略

### V1分支
- **用途**: 主要开发分支
- **内容**: 包含完整的TR069实现和GVA集成
- **稳定性**: 开发版本，功能持续迭代

### 建议的分支管理
- `main`: 稳定发布版本
- `V1`: 当前开发版本
- `feature/*`: 功能开发分支
- `hotfix/*`: 紧急修复分支

## 版本标记

### 语义化版本控制
采用 `MAJOR.MINOR.PATCH` 格式：
- **MAJOR**: 不兼容的API更改
- **MINOR**: 向后兼容的功能添加
- **PATCH**: 向后兼容的错误修复

### 标签示例
```bash
# 创建版本标签
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

## 协作流程

### 提交信息规范
使用约定式提交格式：
```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

**类型 (type)**:
- `feat`: 新功能
- `fix`: 错误修复
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建过程或辅助工具的变动

**范围 (scope)**:
- `core`: TR069-Core相关
- `adapter`: TR069-Adapter相关
- `docs`: 文档相关
- `config`: 配置相关

### 示例提交信息
```bash
feat(core): 添加设备参数批量操作功能
fix(adapter): 修复设备连接状态同步问题
docs: 更新架构设计文档
refactor(core): 优化消息处理性能
```

## 发布流程

### 准备发布
1. 确保所有测试通过
2. 更新版本号和CHANGELOG
3. 创建发布标签
4. 推送到GitHub

### 发布检查清单
- [ ] 代码编译无错误
- [ ] 单元测试全部通过
- [ ] 文档已更新
- [ ] 版本号已更新
- [ ] CHANGELOG已更新

## 问题追踪

### Issue 标签
- `bug`: 错误报告
- `enhancement`: 功能增强
- `documentation`: 文档相关
- `question`: 问题咨询
- `help wanted`: 需要帮助

### Pull Request 模板
```markdown
## 更改描述
简要描述本次PR的更改内容

## 更改类型
- [ ] Bug修复
- [ ] 新功能
- [ ] 文档更新
- [ ] 代码重构
- [ ] 性能优化

## 测试
- [ ] 已添加单元测试
- [ ] 已进行集成测试
- [ ] 已进行手动测试

## 检查清单
- [ ] 代码遵循项目规范
- [ ] 已更新相关文档
- [ ] 已添加必要的测试
```

## 自动化配置

### GitHub Actions (建议)
可以配置以下自动化流程：
- 代码质量检查
- 自动化测试
- 文档生成
- 发布部署

### 配置文件位置
```
.github/
├── workflows/
│   ├── ci.yml          # 持续集成
│   ├── release.yml     # 发布流程
│   └── docs.yml        # 文档生成
└── ISSUE_TEMPLATE/     # Issue模板
```

---

**注意**: 
1. 推送前请确保已经测试过代码的编译和基本功能
2. 敏感信息（如数据库密码、API密钥）不应提交到公共仓库
3. 大文件应使用Git LFS进行管理
4. 定期同步上游更改，保持代码最新

**最后更新**: 2024年10月2日  
**维护者**: AI Assistant  
**联系方式**: 通过GitHub Issues进行沟通