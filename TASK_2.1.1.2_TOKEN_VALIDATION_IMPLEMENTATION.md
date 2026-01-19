# Task 2.1.1.2: Token Validation Logic Implementation

## 任务概述

实现 Kiro 多账号管理器的 Token 验证逻辑，包括格式验证、过期检查和 API 集成。

## 实现状态: ✅ 已完成

## 实现内容

### 1. 增强的 Token 验证方法

#### 1.1 `ValidateToken` 方法增强
- **多层验证流程**:
  1. 基础验证（空值检查）
  2. 格式验证（长度、字符合法性）
  3. 远程 API 验证
  4. 过期时间验证

- **详细的错误处理**:
  - `401 Unauthorized`: Token 无效或已过期
  - `403 Forbidden`: Token 缺少必要权限
  - `429 Too Many Requests`: 请求频率限制
  - 其他状态码的通用错误处理

- **请求头增强**:
  - 添加 `User-Agent` 标识
  - 标准化 `Authorization` 头格式

### 2. Token 格式验证

#### 2.1 `validateTokenFormat` 方法
```go
func (as *AuthService) validateTokenFormat(token string) error
```

**验证规则**:
- 自动移除 "Bearer " 前缀
- 最小长度: 20 字符
- 最大长度: 2048 字符
- 允许的字符: 字母、数字、`-`、`_`、`.`、`~`、`+`、`/`、`=`
- 拒绝包含特殊字符的 Token

**设计理由**:
- 防止明显无效的 Token 发送到 API
- 减少不必要的网络请求
- 提供即时的客户端反馈

### 3. Token 过期验证

#### 3.1 `validateTokenExpiry` 方法
```go
func (as *AuthService) validateTokenExpiry(tokenInfo *TokenInfo) error
```

**功能**:
- 检查 Token 是否已过期
- 对即将过期的 Token（5分钟内）发出警告
- 提供详细的过期时间信息

**警告机制**:
- Token 在 5 分钟内过期时打印警告
- 不阻止操作，但提醒用户刷新

### 4. 重试机制

#### 4.1 `ValidateTokenWithRetry` 方法
```go
func (as *AuthService) ValidateTokenWithRetry(token string, maxRetries int) (*TokenInfo, error)
```

**特性**:
- 支持自动重试瞬态失败
- 指数退避策略: 1s, 2s, 4s
- 智能重试判断:
  - 认证错误（401）不重试
  - 格式错误不重试
  - 网络错误和服务器错误会重试

**使用场景**:
- 网络不稳定环境
- 服务器临时故障
- 批量操作中的容错

## 测试覆盖

### 1. `TestValidateToken`
测试场景:
- ✅ 有效 Token
- ✅ 空 Token
- ✅ Token 太短
- ✅ Token 包含非法字符
- ✅ 401 未授权
- ✅ 403 禁止访问
- ✅ 429 请求限制
- ✅ Token 即将过期（带警告）
- ✅ Token 已过期

### 2. `TestValidateTokenFormat`
测试场景:
- ✅ 有效的 JWT Token
- ✅ 带 Bearer 前缀的 Token
- ✅ Token 太短
- ✅ Token 太长
- ✅ 包含非法字符
- ✅ 包含合法特殊字符

### 3. `TestValidateTokenExpiry`
测试场景:
- ✅ 未来过期的有效 Token
- ✅ 即将过期的 Token（带警告）
- ✅ 已过期的 Token
- ✅ nil TokenInfo

### 4. `TestValidateTokenWithRetry`
测试场景:
- ✅ 首次尝试成功
- ✅ 重试后成功
- ✅ 认证错误不重试

### 5. 其他相关测试
- ✅ `TestGetUserProfile` - 通过
- ✅ `TestValidateAndCreateAccount` - 通过（已更新）
- ✅ `TestLoginWithPassword` - 通过

## 测试结果

```bash
=== RUN   TestValidateToken
--- PASS: TestValidateToken (0.01s)
    --- PASS: TestValidateToken/valid_token (0.01s)
    --- PASS: TestValidateToken/empty_token (0.00s)
    --- PASS: TestValidateToken/token_too_short (0.00s)
    --- PASS: TestValidateToken/token_with_invalid_characters (0.00s)
    --- PASS: TestValidateToken/invalid_token_-_401 (0.00s)
    --- PASS: TestValidateToken/forbidden_-_403 (0.00s)
    --- PASS: TestValidateToken/rate_limited_-_429 (0.00s)
    --- PASS: TestValidateToken/token_expiring_soon (0.00s)
    --- PASS: TestValidateToken/expired_token (0.00s)

=== RUN   TestValidateTokenFormat
--- PASS: TestValidateTokenFormat (0.00s)

=== RUN   TestValidateTokenExpiry
--- PASS: TestValidateTokenExpiry (0.00s)

=== RUN   TestValidateTokenWithRetry
--- PASS: TestValidateTokenWithRetry (1.01s)

PASS
ok      myapp   2.009s
```

## 构建验证

```bash
$ go build -o myapp_test
Exit Code: 0
```

✅ 构建成功，无错误

## API 集成说明

### 当前实现
- 基础 URL: `https://api.kiro.ai`
- 验证端点: `/auth/validate`
- 用户信息端点: `/user/profile`

### 待配置项
1. **实际 Kiro API 端点**
   - 需要从 Kiro 官方获取正确的 API 基础 URL
   - 可能需要不同的端点路径

2. **API 认证方式**
   - 当前使用 Bearer Token
   - 可能需要额外的 API Key 或签名

3. **响应格式**
   - 当前假设标准 JSON 响应
   - 需要根据实际 API 调整数据结构

## 代码质量

### 优点
1. **多层验证**: 客户端验证 + 服务器验证
2. **详细错误**: 针对不同场景的具体错误消息
3. **容错机制**: 重试逻辑处理瞬态故障
4. **安全性**: Token 格式验证防止注入攻击
5. **用户体验**: 过期警告提前通知用户

### 遵循的最佳实践
- ✅ 输入验证
- ✅ 错误处理
- ✅ 单元测试覆盖
- ✅ 代码注释
- ✅ 类型安全

## 使用示例

### 基础验证
```go
authService := NewAuthService()
tokenInfo, err := authService.ValidateToken("your-bearer-token-here")
if err != nil {
    log.Printf("Token validation failed: %v", err)
    return
}
log.Printf("Token valid until: %s", tokenInfo.ExpiresAt)
```

### 带重试的验证
```go
authService := NewAuthService()
tokenInfo, err := authService.ValidateTokenWithRetry("your-bearer-token-here", 3)
if err != nil {
    log.Printf("Token validation failed after retries: %v", err)
    return
}
log.Printf("Token validated successfully")
```

### 创建账号时验证
```go
authService := NewAuthService()
quotaService := NewQuotaService()

account, err := authService.ValidateAndCreateAccount(
    token,
    LoginMethodToken,
    "",
    quotaService,
)
if err != nil {
    log.Printf("Failed to create account: %v", err)
    return
}
log.Printf("Account created: %s", account.Email)
```

## 与其他模块的集成

### 1. AccountManager
- 添加账号时自动验证 Token
- 切换账号时检查 Token 有效性

### 2. QuotaService
- Token 验证后获取配额信息
- 定期刷新时重新验证

### 3. 前端界面
- Token 输入时的即时验证反馈
- 过期警告的 UI 提示

## 后续改进建议

### 短期
1. 配置实际的 Kiro API 端点
2. 添加 Token 刷新的自动触发
3. 实现 Token 缓存机制

### 中期
1. 支持多种 Token 类型（JWT、OAuth2）
2. Token 元数据解析（如 JWT claims）
3. 更细粒度的权限验证

### 长期
1. Token 生命周期管理
2. 自动续期机制
3. Token 使用统计和监控

## 依赖关系

### 外部依赖
- `net/http`: HTTP 客户端
- `encoding/json`: JSON 解析
- `time`: 时间处理

### 内部依赖
- `KiroAccount`: 账号数据结构
- `TokenInfo`: Token 信息结构
- `UserProfile`: 用户信息结构
- `QuotaService`: 配额服务

## 安全考虑

1. **Token 存储**: Token 在内存中处理，不记录到日志
2. **传输安全**: 使用 HTTPS（baseURL 配置）
3. **格式验证**: 防止恶意 Token 注入
4. **超时设置**: 30 秒超时防止挂起
5. **错误信息**: 不泄露敏感的系统信息

## 性能特性

- **客户端验证**: 减少无效请求
- **重试机制**: 提高成功率
- **超时控制**: 防止长时间等待
- **并发安全**: 无状态设计，支持并发调用

## 文档和注释

- ✅ 所有公共方法都有详细注释
- ✅ 复杂逻辑有内联说明
- ✅ 错误消息清晰明确
- ✅ 测试用例有描述性名称

## 验收标准检查

根据 requirements.md 的验收标准:

- ✅ **AC-001**: 支持 Token 方式添加账号
- ✅ **AC-004**: Token 验证失败时显示错误信息
- ✅ **AC-008**: 切换失败时显示错误信息
- ✅ **AC-021**: Token 等敏感信息安全处理
- ✅ **AC-024**: 异常情况下不泄露敏感信息

## 总结

Task 2.1.1.2 已成功完成，实现了完整的 Token 验证逻辑，包括：

1. ✅ 多层验证机制
2. ✅ 格式和过期检查
3. ✅ 详细的错误处理
4. ✅ 重试机制
5. ✅ 全面的单元测试
6. ✅ 构建验证通过

**下一步**: 继续 Task 2.1.1.3 "获取用户信息和配额"，该功能的基础已在 `ValidateAndCreateAccount` 方法中实现。

## 变更记录

- **2026-01-18**: 初始实现完成
  - 实现 Token 格式验证
  - 实现 Token 过期检查
  - 实现重试机制
  - 添加全面的单元测试
  - 更新相关测试用例
  - 验证构建成功
