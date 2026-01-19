# Kiro 账号管理器使用指南

## 功能说明

Kiro 账号管理器允许您管理多个 Kiro 账号，并在它们之间快速切换。

### 核心功能

1. **添加账号**：通过 Refresh Token 添加新的 Kiro 账号
2. **查看账号**：查看所有已添加的账号及其配额信息
3. **切换账号**：点击"切换"按钮，将选中的账号应用到 OpenCode 插件
4. **删除账号**：删除不需要的账号
5. **编辑账号**：修改账号的显示名称和备注

## 存储位置

### 账号管理器数据
- **位置**：`~/.config/opencode/data/accounts.json.enc`
- **格式**：加密的 JSON 文件
- **内容**：所有已添加的 Kiro 账号信息

### OpenCode 插件当前账号
- **位置**：`~/.aws/sso/cache/kiro-auth-token.json`
- **格式**：明文 JSON 文件
- **内容**：当前激活账号的 Token 信息

## 工作流程

### 1. 添加账号

```
用户输入 Refresh Token
    ↓
调用 Kiro API 刷新 Token
    ↓
获取 Access Token 和用户信息
    ↓
获取配额信息
    ↓
创建账号对象
    ↓
保存到账号管理器
```

**后端 API 调用链**：
```
AddKiroAccount()
  → addAccountByToken()
    → NewKiroAPIClient()
    → RefreshKiroToken()  // 调用 Kiro Auth API
    → GetKiroUsageLimits() // 调用 Kiro Usage API
    → ConvertKiroResponseToAccount()
    → AccountManager.AddAccount()
      → StorageService.SaveAccountData()
```

### 2. 切换账号

```
用户点击"切换"按钮
    ↓
调用 SwitchAccount(accountId)
    ↓
更新账号激活状态
    ↓
调用 ApplyAccountToSystem()
    ↓
写入 Token 到 OpenCode 配置
    ↓
（可选）更新 Machine ID
    ↓
保存账号管理器数据
```

**后端 API 调用链**：
```
SwitchKiroAccount(id)
  → AccountManager.SwitchAccount(id)
    → KiroSystem.ApplyAccountToSystem()
      → writeTokenFile()  // 写入 ~/.aws/sso/cache/kiro-auth-token.json
      → updateMachineID() // 可选：更新 Machine ID
    → StorageService.SaveAccountData()
```

### 3. Token 文件格式

`~/.aws/sso/cache/kiro-auth-token.json`:
```json
{
  "accessToken": "aoaAAAAAGlrx...",
  "refreshToken": "aorAAAAAGnLG...",
  "expiresAt": "2025-01-18T12:00:00Z",
  "authMethod": "social",
  "provider": "builderid",
  "profileArn": "arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK"
}
```

## API 端点

### Kiro Auth API
- **Base URL**: `https://prod.us-east-1.auth.desktop.kiro.dev`
- **刷新 Token**: `POST /refreshToken`
  ```json
  {
    "refreshToken": "aorAAAAAGnLG..."
  }
  ```
  响应：
  ```json
  {
    "accessToken": "aoaAAAAAGlrx...",
    "refreshToken": "aorAAAAAGnLG...",
    "expiresIn": 3600,
    "profileArn": "arn:aws:codewhisperer:..."
  }
  ```

### Kiro Usage API
- **Base URL**: `https://codewhisperer.us-east-1.amazonaws.com`
- **获取配额**: `GET /getUsageLimits?isEmailRequired=true&origin=AI_EDITOR&profileArn=...`
  - Header: `Authorization: Bearer <accessToken>`
  
  响应：
  ```json
  {
    "userInfo": {
      "email": "user@example.com",
      "userId": "..."
    },
    "subscriptionInfo": {
      "subscriptionTitle": "KIRO FREE",
      "type": "Q_DEVELOPER_STANDALONE_FREE"
    },
    "usageBreakdownList": [{
      "resourceType": "...",
      "usageLimit": 50,
      "currentUsage": 0,
      "freeTrialInfo": {
        "usageLimit": 500,
        "currentUsage": 0
      }
    }]
  }
  ```

## 配置选项

### 自动切换 Machine ID
- **位置**：设置 → Kiro 账号设置 → 自动切换机器码
- **说明**：开启后，切换账号时会自动更新系统的 machineId、sqmId 和 deviceId
- **用途**：实现账号间的完全隔离，避免账号关联

## 故障排查

### 问题 1：添加账号后 UI 不显示

**可能原因**：
1. 账号数据未正确保存
2. 前端未刷新账号列表
3. 后端 API 返回错误

**解决方法**：
```bash
# 检查账号数据文件是否存在
ls -la ~/.config/opencode/data/accounts.json.enc

# 检查应用日志
# 在应用中查看控制台输出
```

### 问题 2：切换账号后 OpenCode 插件未生效

**可能原因**：
1. Token 文件未正确写入
2. OpenCode 插件未重新加载配置

**解决方法**：
```bash
# 检查 Token 文件
cat ~/.aws/sso/cache/kiro-auth-token.json

# 重启 OpenCode 插件或 Kiro IDE
```

### 问题 3：Refresh Token 无效

**可能原因**：
1. Token 已过期
2. Token 格式错误
3. 网络连接问题

**解决方法**：
- 重新获取 Refresh Token
- 检查网络连接
- 查看错误信息

## 测试步骤

### 手动测试添加账号

1. 启动应用：`wails dev`
2. 打开设置面板
3. 点击"Kiro 账号"
4. 点击"添加账号"
5. 选择"Refresh Token"方式
6. 粘贴 Refresh Token
7. 点击"添加账号"
8. 等待 API 调用完成
9. 查看账号列表

### 验证账号数据

```bash
# 检查账号管理器数据
ls -la ~/.config/opencode/data/

# 检查 OpenCode Token（切换账号后）
cat ~/.aws/sso/cache/kiro-auth-token.json
```

## 开发说明

### 相关文件

**后端**：
- `myapp/kiro_api_real.go` - Kiro API 客户端
- `myapp/app.go` - Wails 应用 API
- `myapp/account_manager.go` - 账号管理器
- `myapp/kiro_system.go` - 系统集成（写入 Token 文件）
- `myapp/storage_service.go` - 数据存储服务
- `myapp/config_manager.go` - 配置管理

**前端**：
- `myapp/frontend/src/components/KiroAccountManager.vue` - UI 组件
- `myapp/frontend/wailsjs/go/main/App.js` - Wails 绑定

### 添加新功能

1. 在 `app.go` 中添加新的导出方法
2. 运行 `wails dev` 自动生成前端绑定
3. 在 Vue 组件中导入并使用新方法

## 安全注意事项

1. **Token 加密**：账号管理器数据使用 AES-256-GCM 加密
2. **文件权限**：Token 文件权限设置为 0600（仅所有者可读写）
3. **敏感信息**：前端 API 返回的账号对象不包含 Token（已清空）
4. **备份**：账号数据会自动备份到 `~/.config/opencode/backups/`
