# Kiro API 配置说明

## 当前状态

✅ **前端已完成**: 支持 Refresh Token 输入,界面已优化
✅ **后端已完成**: 实现了 Refresh Token → Bearer Token 的自动转换流程
⚠️ **需要配置**: Kiro API 端点地址需要配置为实际的 API 地址

## 问题说明

当您尝试添加账号时出现 "Refresh Token 验证失败: undefined" 错误,这是因为:

1. 后端尝试调用 Kiro API 来刷新 Token
2. 默认的 API 地址是 `https://api.kiro.ai`
3. 这个地址可能不是实际的 Kiro API 地址,或者需要特定的认证方式

## 需要的信息

要让账号管理功能正常工作,我们需要知道:

### 1. Kiro API 的实际端点地址

请提供以下信息:
- **Token 刷新端点**: 用于将 Refresh Token 转换为 Bearer Token
  - 当前默认: `https://api.kiro.ai/auth/refresh`
  - 实际地址: `?`

- **Token 验证端点**: 用于验证 Bearer Token 是否有效
  - 当前默认: `https://api.kiro.ai/auth/validate`
  - 实际地址: `?`

- **用户信息端点**: 用于获取用户的邮箱、名称等信息
  - 当前默认: `https://api.kiro.ai/user/profile`
  - 实际地址: `?`

- **配额信息端点**: 用于获取用户的配额使用情况
  - 当前默认: `https://api.kiro.ai/user/quota`
  - 实际地址: `?`

### 2. API 请求格式

- **请求方法**: POST / GET?
- **请求头**: 需要哪些 HTTP 头?
- **请求体格式**: JSON 格式是什么?
- **响应格式**: 返回的数据结构是什么?

### 3. Token 格式

- **Refresh Token 格式**: 您输入的 Token 看起来是正确的格式
- **Bearer Token 格式**: API 返回的 Bearer Token 是什么格式?

## 配置方法

### 方法 1: 使用环境变量

1. 复制 `.env.example` 为 `.env`:
   ```bash
   cp .env.example .env
   ```

2. 编辑 `.env` 文件,填入实际的 API 地址:
   ```bash
   KIRO_API_BASE_URL=https://actual-kiro-api.com
   KIRO_AUTH_REFRESH_URL=https://actual-kiro-api.com/auth/refresh
   # ... 其他配置
   ```

3. 重新启动应用

### 方法 2: 直接修改代码

编辑 `myapp/kiro_api_config.go` 文件,修改 `DefaultKiroAPIConfig()` 函数中的默认值。

## 测试步骤

配置完成后,按以下步骤测试:

1. **启动应用**
   ```bash
   cd myapp
   wails dev
   ```

2. **打开 Kiro 账号管理**
   - 点击设置图标
   - 选择 "Kiro 账号"

3. **添加账号**
   - 点击 "添加账号"
   - 选择 "Refresh Token" 方式
   - 粘贴您的 Refresh Token
   - 点击 "添加账号"

4. **查看日志**
   - 在终端中查看后端日志
   - 会显示正在调用的 API 端点
   - 会显示详细的错误信息

## 调试建议

### 1. 使用 curl 测试 API

先用 curl 命令测试 Kiro API 是否可以访问:

```bash
# 测试 Token 刷新
curl -X POST https://api.kiro.ai/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "your-refresh-token-here",
    "grant_type": "refresh_token"
  }'
```

### 2. 检查网络连接

确保应用可以访问 Kiro API:
- 检查防火墙设置
- 检查代理设置
- 检查 DNS 解析

### 3. 查看详细日志

后端会输出详细的日志信息:
```
Attempting to refresh token using endpoint: https://api.kiro.ai/auth/refresh
failed to send request to https://api.kiro.ai/auth/refresh: ...
```

## 临时解决方案

如果暂时无法配置正确的 API 地址,您可以:

1. **手动输入 Bearer Token**: 
   - 修改前端,临时支持直接输入 Bearer Token
   - 跳过 Refresh Token 转换步骤

2. **使用模拟数据**:
   - 创建一个本地的模拟 API 服务器
   - 返回测试数据

## 下一步

请提供:
1. ✅ Kiro API 的实际端点地址
2. ✅ API 的请求/响应格式示例
3. ✅ 任何必需的认证信息(API Key、Client ID 等)

有了这些信息,我们就可以完成配置,让账号管理功能正常工作!

---

**文档创建时间**: 2026-01-18
**状态**: 等待 API 配置信息
