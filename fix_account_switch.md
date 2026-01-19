# 账号切换问题诊断和修复

## 问题原因

切换账号后，`kiro-accounts.json` 文件已经更新，但 **OpenCode 不会自动重新读取配置文件**。OpenCode 在启动时读取账号信息并缓存在内存中，所以即使文件更新了，OpenCode 仍然使用旧的账号。

## 解决方案

### 方案 1：重启 OpenCode（推荐）

1. 切换账号后，**完全退出 OpenCode**
2. 重新启动 OpenCode
3. OpenCode 会读取新的账号配置

### 方案 2：添加自动重启功能

在切换账号后，自动重启 OpenCode：

```go
// 在 account_manager.go 的 SwitchAccount 函数中添加
func (am *AccountManager) SwitchAccount(id string) error {
    // ... 现有代码 ...
    
    // 应用到 OpenCode
    if err := openCodeSystem.ApplyAccountToOpenCode(newAccount); err != nil {
        return fmt.Errorf("failed to apply account to OpenCode: %w", err)
    }
    
    // 通知用户需要重启 OpenCode
    if am.ctx != nil {
        runtime.EventsEmit(am.ctx, "kiro-account-switched-restart-required", map[string]interface{}{
            "message": "账号已切换，请重启 OpenCode 使其生效",
            "accountEmail": newAccount.Email,
        })
    }
    
    return nil
}
```

### 方案 3：使用 OpenCode API 重新加载配置

如果 OpenCode 提供了重新加载配置的 API，可以调用它：

```bash
# 检查 OpenCode 是否在运行
ps aux | grep opencode

# 发送信号让 OpenCode 重新加载配置（如果支持）
pkill -HUP opencode
```

## 验证步骤

1. 在应用中切换账号
2. 检查文件是否更新：
   ```bash
   cat ~/.config/opencode/kiro-accounts.json | jq '.accounts[] | {email, id}'
   ```
3. 重启 OpenCode
4. 在 OpenCode 中使用 Kiro 功能，查看是否使用了新账号

## 当前状态

- ✅ 账号切换逻辑正确
- ✅ 文件写入成功
- ❌ OpenCode 未自动重新加载配置

## 建议

**立即实施**：在前端添加提示，告知用户切换账号后需要重启 OpenCode。

**长期方案**：研究 OpenCode 插件 API，实现自动重新加载或自动重启功能。
