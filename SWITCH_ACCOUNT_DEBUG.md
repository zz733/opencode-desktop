# 账号切换问题诊断

## 问题现象
- 点击切换按钮后，终端没有任何输出
- `kiro-accounts.json` 文件没有更新
- 账号没有切换成功

## 问题原因
**前端的切换函数没有被调用**，或者调用时出错了但没有显示。

## 诊断步骤

### 1. 检查浏览器 Console
在应用中按 `Cmd+Option+I` 打开开发者工具，切换到 Console 标签，然后点击切换按钮，看看有没有：
- JavaScript 错误
- 我添加的日志（`=== 前端: switchAccount 开始 ===`）

### 2. 检查按钮是否被禁用
查看切换按钮是否是灰色的（被禁用状态）

### 3. 手动测试后端函数
在浏览器 Console 中直接运行：
```javascript
window.go.main.App.SwitchKiroAccount("你的账号ID").then(
  () => console.log("成功"),
  (err) => console.error("失败:", err)
)
```

### 4. 检查日志文件
```bash
tail -50 /tmp/kiro-account-manager.log
```

## 可能的解决方案

### 方案 1：前端代码没有生效
- 完全退出应用
- 删除缓存：`rm -rf myapp/frontend/dist`
- 重新编译：`wails dev`

### 方案 2：Wails 绑定问题
- 重新生成绑定：`wails generate module`
- 重新编译

### 方案 3：事件被拦截
- 检查是否有其他代码拦截了点击事件
- 检查对话框的 `@click` 事件是否阻止了冒泡

## 临时解决方案

直接在终端运行切换命令（需要先找到账号 ID）：
```bash
# 查看所有账号
cat ~/Library/Application\ Support/OpenCode/KiroAccountManager/data/accounts.json.enc

# 手动修改 kiro-accounts.json
# （需要知道账号的 RefreshToken 和 BearerToken）
```
