# 代码结构说明

## 当前模块组织

### 已拆分的模块
- `main.go` (1.7K) - 应用入口
- `provider.go` (61 行) - Provider 和配置管理
- `model.go` (123 行) - 模型配置管理
- `session.go` (541 行) - 会话管理（Session, Message, 图片处理, SSE 订阅）
- `plugin.go` (497 行) - 插件管理（OhMyOpenCode, AntigravityAuth）
- `search.go` (4.3K) - 文件搜索和替换功能
- `git.go` (3.1K) - Git 版本控制操作
- `terminal.go` (3.6K) - 终端管理
- `opencode.go` (7.9K) - OpenCode 服务管理
- `files.go` (9.3K) - 文件系统操作

### 待拆分的模块（当前在 app.go 中）
`app.go` (970 行) - **还需要拆分：**

1. **mcp.go** - MCP 服务器管理（约 600 行）
   - MCPServer, MCPConfig, MCPMarketItem 类型
   - GetMCPConfig, SaveMCPConfig
   - AddMCPServer, RemoveMCPServer
   - GetMCPStatus, GetMCPTools
   - MCP 市场列表

2. **run.go** - 文件运行功能（约 150 行）
   - RunFile 及相关辅助函数
   - 支持多种语言的运行配置

3. **app_core.go** - 核心 App 结构（约 220 行）
   - App 结构定义
   - 初始化和启动函数
   - 文件夹选择和工作目录管理
   - 文件操作（删除、重命名、复制、移动）

## 拆分进度

- ✅ provider.go - Provider 和配置
- ✅ model.go - 模型配置管理
- ✅ session.go - 会话管理
- ✅ plugin.go - 插件管理
- ⏳ mcp.go - MCP 服务器管理（下一步）
- ⏳ run.go - 文件运行功能
- ⏳ app_core.go - 核心 App 结构

## 拆分原则

1. **单一职责** - 每个文件只负责一个功能领域
2. **保持编译** - 拆分过程中确保代码能编译通过
3. **逐步迁移** - 一次拆分一个模块，测试后再继续
4. **避免循环依赖** - 保持清晰的依赖关系

## 验证

- ✅ 编译成功
- ✅ 所有模块功能完整
- ✅ 代码行数从 2146 减少到 970（减少 55%）
