# Kiro 账号切换自动化修复

## 问题描述

用户反馈在使用"Kiro账户管理"中的"切换账号"功能时，Kiro 相关模型无法使用。

经检查发现以下问题：
1. **配置文件缺失**：`~/.config/opencode/kiro-accounts.json` 不存在，只有备份文件 `.bak`
2. **Token 过期**：即使有配置文件，token 也可能已经过期
3. **手动修复无意义**：手动改名或复制文件只能临时解决，下次切换或其他用户仍会遇到同样问题

## 解决方案

修改 `ApplyAccountToOpenCode` 函数，实现完全自动化的账号切换：

### 1. 自动检查和刷新 Token

```go
// 检查 token 是否过期，如果过期则刷新
if time.Now().After(expiresAt.Add(-5 * time.Minute)) {
    fmt.Printf("  ⚠ Token 已过期或即将过期\n")
    fmt.Printf("  → 正在刷新 Token...\n")
    
    // 使用 Kiro API 客户端刷新 token
    kiroClient := NewKiroAPIClient()
    tokenResp, err := kiroClient.RefreshKiroToken(refreshToken)
    if err != nil {
        return fmt.Errorf("token 刷新失败: %w", err)
    }
    
    bearerToken = tokenResp.AccessToken
    refreshToken = tokenResp.RefreshToken
    expiresAt = time.Now().Add(1 * time.Hour)
    
    // 更新账号对象中的 token
    account.BearerToken = bearerToken
    account.RefreshToken = refreshToken
    account.TokenExpiry = expiresAt
}
```

### 2. 自动创建配置文件

改进 `WriteKiroAccounts` 函数：

```go
// Ensure directory exists
dir := filepath.Dir(path)
if err := os.MkdirAll(dir, 0755); err != nil {
    return fmt.Errorf("failed to create directory %s: %w", dir, err)
}

// Atomic write with proper error handling
tmpPath := path + ".tmp"
if err := os.WriteFile(tmpPath, data, 0644); err != nil {
    return fmt.Errorf("failed to write temp file: %w", err)
}

// Remove old file if exists (to avoid permission issues)
if _, err := os.Stat(path); err == nil {
    if err := os.Remove(path); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("failed to remove old file: %w", err)
    }
}

if err := os.Rename(tmpPath, path); err != nil {
    os.Remove(tmpPath)
    return fmt.Errorf("failed to rename temp file: %w", err)
}
```

## 修复效果

现在切换账号时会自动：

1. ✅ **检查 Token 有效性**
   - 如果 token 在 5 分钟内过期，自动刷新
   - 刷新后更新账号对象中的 token

2. ✅ **创建配置目录**
   - 如果 `~/.config/opencode/` 不存在，自动创建

3. ✅ **写入配置文件**
   - 使用原子写入（先写临时文件，再重命名）
   - 处理文件权限问题
   - 确保配置文件正确创建

4. ✅ **完全自动化**
   - 用户只需点击"切换"按钮
   - 所有问题自动处理
   - 无需手动干预

## 测试方法

### 方法 1：使用测试脚本

```bash
cd myapp
./test_switch_fix.sh
```

这会显示：
- 当前配置文件状态
- Token 是否过期
- 测试说明

### 方法 2：实际测试

1. 启动应用：
   ```bash
   wails dev
   ```

2. 打开"Kiro 账户管理"

3. 点击任意账号的"切换"按钮

4. 观察终端日志，应该看到：
   ```
   ========================================
     → ApplyAccountToOpenCode 开始
     → 账号 ID: kiro-xxx
     → 账号邮箱: xxx@example.com
     ⚠ Token 已过期或即将过期 (过期时间: 2026-01-22 08:34:29)
     → 正在刷新 Token...
     ✓ Token 刷新成功
     ✓ 创建 OpenCode 账号结构完成
     ...
     ✓ 文件写入成功！
   ========================================
   ```

5. 验证配置文件：
   ```bash
   cat ~/.config/opencode/kiro-accounts.json
   ```

   应该看到：
   - 正确的账号信息
   - 新的 accessToken
   - 未来的过期时间

## 日志输出示例

```
========================================
  → ApplyAccountToOpenCode 开始
  → 账号 ID: kiro-b670fa310d6e9f4322c3e69850a4aad6
  → 账号邮箱: nguyenngocchaupha7t6ua@k25.huas.edu.vn
  → RefreshToken 长度: 234
  → BearerToken 长度: 198
  ⚠ Token 已过期或即将过期 (过期时间: 2026-01-22 07:34:29)
  → 正在刷新 Token...
  ✓ Token 刷新成功
  ✓ 创建 OpenCode 账号结构完成 (authMethod=idc, profileArn=arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK)
  ✓ 创建账号文件结构完成（账号数: 1）
  → 目标文件路径: /Users/xxx/.config/opencode/kiro-accounts.json
  → 开始写入文件...
  ✓ 文件写入成功！新修改时间: 2026-01-22 08:45:00
  ✓ 验证: 文件中账号数 = 1
  ✓ 验证: 第一个账号邮箱 = nguyenngocchaupha7t6ua@k25.huas.edu.vn
  → 更新 kiro-usage.json...
  ✓ usage 文件更新成功
========================================
```

## 相关文件

- `myapp/opencode_kiro.go` - OpenCode Kiro 系统集成
- `myapp/account_manager.go` - 账号管理器
- `myapp/kiro_api_real.go` - Kiro API 客户端
- `myapp/test_switch_fix.sh` - 测试脚本

## 注意事项

1. **Token 刷新失败**：如果 refreshToken 也失效，需要重新登录
2. **网络问题**：刷新 token 需要网络连接
3. **权限问题**：确保有权限写入 `~/.config/opencode/` 目录

## 总结

这次修复实现了完全自动化的账号切换：
- ✅ 不需要手动创建配置文件
- ✅ 不需要手动刷新 token
- ✅ 不需要手动处理任何问题
- ✅ 适用于所有用户和所有账号

用户只需点击"切换"按钮，系统会自动处理所有细节。
