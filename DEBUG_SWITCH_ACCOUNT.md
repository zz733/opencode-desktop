# 调试账号切换 "undefined" 错误

## 问题现象

界面显示：**"切换账号失败: undefined"**

## 可能原因

1. **后端代码未重新编译**
   - 修改了 `opencode_kiro.go` 但应用还在运行旧代码
   
2. **后端返回错误格式不正确**
   - Go 错误对象在 JavaScript 中可能没有 `message` 属性

3. **Wails 绑定问题**
   - 后端函数签名改变但前端绑定未更新

## 调试步骤

### 1. 停止当前应用

确保完全停止 `wails dev` 进程。

### 2. 重新生成 Wails 绑定

```bash
cd myapp
wails generate module
```

### 3. 重新构建前端

```bash
cd frontend
npm run build
cd ..
```

### 4. 重新启动应用

```bash
wails dev
```

### 5. 查看终端日志

启动后，尝试切换账号，观察终端输出：

**期望看到：**
```
========================================
  → ApplyAccountToOpenCode 开始
  → 账号 ID: kiro-xxx
  → 账号邮箱: xxx@example.com
  → RefreshToken 长度: 234
  → BearerToken 长度: 198
  ⚠ Token 已过期或即将过期 (过期时间: 2026-01-22 08:34:29)
  → 正在刷新 Token...
  ✓ Token 刷新成功
  ...
```

**如果看到错误：**
- 记录完整的错误信息
- 检查是否是网络问题
- 检查 refreshToken 是否有效

### 6. 查看浏览器控制台

打开浏览器开发者工具（F12），查看 Console 标签：

**期望看到：**
```
=== 前端: switchAccount 开始 ===
→ 账号 ID: kiro-xxx
→ 账号邮箱: xxx@example.com
→ 调用后端 SwitchKiroAccount...
✓ 后端调用成功
→ 重新加载账号列表...
✓ 账号列表已重新加载
=== 前端: switchAccount 完成 ===
```

**如果看到错误：**
```
✗ 切换账号失败: [错误对象]
错误类型: [类型]
错误内容: [JSON]
```

记录这些信息以便进一步调试。

## 常见问题

### Q1: 显示 "undefined"

**原因：** 后端返回的错误对象在 JavaScript 中没有 `message` 属性。

**解决：** 已修改前端代码，现在会显示完整的错误信息。

### Q2: Token 刷新失败

**原因：** refreshToken 可能已失效。

**解决：** 
1. 删除该账号
2. 重新添加账号
3. 或者手动更新 refreshToken

### Q3: 配置文件权限问题

**原因：** 无法写入 `~/.config/opencode/kiro-accounts.json`

**解决：**
```bash
chmod 755 ~/.config/opencode
chmod 644 ~/.config/opencode/kiro-accounts.json
```

### Q4: 数据库锁定

**原因：** SQLite 数据库被其他进程锁定。

**解决：**
```bash
# 停止所有相关进程
pkill -f myapp
pkill -f wails

# 删除锁文件
rm ~/.config/opencode/kiro.db-wal
rm ~/.config/opencode/kiro.db-shm

# 重新启动
wails dev
```

## 手动测试切换功能

如果界面切换失败，可以在浏览器控制台手动测试：

```javascript
// 1. 获取所有账号
window.go.main.App.GetKiroAccounts().then(accounts => {
  console.log('账号列表:', accounts)
})

// 2. 切换到指定账号
window.go.main.App.SwitchKiroAccount('kiro-xxx').then(
  () => console.log('✓ 切换成功'),
  (err) => console.error('✗ 切换失败:', err)
)

// 3. 查看当前激活账号
window.go.main.App.GetActiveKiroAccount().then(account => {
  console.log('当前激活账号:', account)
})
```

## 验证修复

切换成功后，验证以下内容：

1. **配置文件存在且正确**
   ```bash
   cat ~/.config/opencode/kiro-accounts.json | jq
   ```

2. **Token 未过期**
   ```bash
   # 获取过期时间（毫秒）
   EXPIRES=$(cat ~/.config/opencode/kiro-accounts.json | jq -r '.accounts[0].expiresAt')
   # 转换为可读格式
   date -r $(($EXPIRES / 1000)) '+%Y-%m-%d %H:%M:%S'
   ```

3. **Kiro 模型可用**
   - 在聊天界面选择 Kiro 模型
   - 发送测试消息
   - 应该能正常响应

## 需要帮助？

如果以上步骤都无法解决问题，请提供：

1. 完整的终端日志（从启动到切换失败）
2. 浏览器控制台的完整错误信息
3. `~/.config/opencode/kiro-accounts.json` 的内容（隐藏 token）
4. 数据库中的账号信息：
   ```bash
   sqlite3 ~/.config/opencode/kiro.db "SELECT id, email, is_healthy FROM accounts"
   ```
