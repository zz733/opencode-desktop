# 代码结构说明

## 当前模块组织

### 已拆分的模块
- `main.go` (1.7K) - 应用入口
- `search.go` (4.3K) - 文件搜索和替换功能
- `git.go` (3.1K) - Git 版本控制操作
- `terminal.go` (3.6K) - 终端管理
- `opencode.go` (7.9K) - OpenCode 服务管理
- `files.go` (9.3K) - 文件系统操作

### 待拆分的模块（当前在 app.go 中）
`app.go` (60K) - **需要拆分为以下模块：**

1. **session.go** - 会话管理
   - Session, Message, ImageData 类型
   - GetSessions, CreateSession
   - SendMessage, SendMessageWithModel
   - GetSessionMessages, CodeCompletion
   - SubscribeEvents, CheckConnection

2. **provider.go** - Provider 和配置
   - Provider, Model, ProviderInfo 类型
   - GetProviders, GetConfig

3. **plugin.go** - 插件管理
   - OhMyOpenCodeStatus, AntigravityAuthStatus 类型
   - InstallOhMyOpenCode, UninstallOhMyOpenCode
   - InstallAntigravityAuth, UninstallAntigravityAuth
   - RestartOpenCode

4. **mcp.go** - MCP 服务器管理
   - MCPServer, MCPConfig, MCPMarketItem 类型
   - GetMCPConfig, SaveMCPConfig
   - AddMCPServer, RemoveMCPServer
   - GetMCPStatus, GetMCPTools

5. **model.go** - 模型配置管理
   - ConfigModel 类型
   - GetConfigModels
   - readModelsFromConfig

## 拆分原则

1. **单一职责** - 每个文件只负责一个功能领域
2. **保持编译** - 拆分过程中确保代码能编译通过
3. **逐步迁移** - 一次拆分一个模块，测试后再继续
4. **避免循环依赖** - 保持清晰的依赖关系

## 下一步计划

1. 创建 `session.go` - 拆分会话管理相关代码
2. 创建 `provider.go` - 拆分 Provider 相关代码
3. 创建 `plugin.go` - 拆分插件管理代码
4. 创建 `mcp.go` - 拆分 MCP 管理代码
5. 创建 `model.go` - 拆分模型配置代码
6. 精简 `app.go` - 只保留核心 App 结构和初始化

## 注意事项

- 拆分时需要注意类型定义的重复
- 确保所有导出的函数都有接收者 `(a *App)`
- 保持包级别的私有函数在合适的文件中
- 测试每次拆分后的编译和功能
