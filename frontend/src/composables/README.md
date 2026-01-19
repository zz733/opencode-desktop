# Kiro 多账号管理器 - 响应式数据管理

本文档描述了 Kiro 多账号管理器的响应式数据管理系统的设计和实现。

## 概述

响应式数据管理系统基于 Vue 3 Composition API 构建，提供了一套完整的状态管理、UI交互、表单验证和数据持久化解决方案。

## 核心架构

### 1. 响应式存储基础 (`useReactiveStore.js`)

提供了两个核心函数：

#### `createReactiveStore(initialState)`
创建基础的响应式存储，包含：
- **状态管理**: 自动管理 loading、error、lastUpdated 等状态
- **操作执行**: 统一的异步操作执行和错误处理
- **事件监听**: 集中的事件监听器管理
- **批量操作**: 支持批量异步操作执行

```javascript
const store = createReactiveStore({
  customData: 'initial value'
})

// 执行操作
await store.executeOperation('my-operation', async () => {
  // 异步操作逻辑
  return result
})
```

#### `createReactiveCollection(options)`
创建响应式集合管理器，扩展基础存储功能：
- **CRUD操作**: 添加、删除、更新、查询项目
- **选择管理**: 多选、全选、范围选择
- **筛选排序**: 搜索、字段筛选、多字段排序
- **统计信息**: 自动计算总数、筛选数、选中数

```javascript
const collection = createReactiveCollection({
  keyField: 'id',
  initialItems: [],
  sortBy: 'name'
})

// 添加项目
collection.addItems([{ id: '1', name: 'Item 1' }])

// 筛选和排序
collection.setSearchQuery('search term')
collection.setSorting('name', 'desc')
```

### 2. 账号数据管理 (`useAccountStore.js`)

基于响应式集合构建的专门用于 Kiro 账号管理的存储：

#### 核心功能
- **账号CRUD**: 完整的账号增删改查操作
- **账号切换**: 安全的账号切换机制
- **配额管理**: 自动配额刷新和警告
- **批量操作**: 批量Token刷新、删除、标签管理
- **事件同步**: 与后端事件系统同步

#### 计算属性
- `activeAccount`: 当前激活的账号
- `validAccountCount`: 有效账号数量
- `allTags`: 所有标签列表
- `subscriptionStats`: 订阅类型统计
- `quotaAlerts`: 配额警告列表

#### 使用示例
```javascript
const accountStore = useGlobalAccountStore()

// 加载账号
await accountStore.loadAccounts()

// 切换账号
await accountStore.switchAccount('account-id')

// 批量操作
await accountStore.batchRefreshTokens(['id1', 'id2'])
```

### 3. UI状态管理 (`useUIState.js`)

管理所有UI相关的响应式状态：

#### 功能模块
- **对话框管理**: 统一的对话框开关和数据管理
- **选择管理**: 多选状态和操作
- **筛选管理**: 搜索、排序、筛选状态
- **通知管理**: 消息通知系统
- **视图管理**: 布局、主题、密度设置
- **加载状态**: 细粒度的加载状态管理

#### 使用示例
```javascript
const uiState = useUIState()

// 打开对话框
uiState.dialogs.open('addAccount', { initialData })

// 显示通知
uiState.notifications.success('操作成功')

// 管理选择
uiState.selection.select(['id1', 'id2'])
```

### 4. 表单验证 (`useFormValidation.js`)

提供强大的表单验证功能：

#### 验证规则
- **内置规则**: required, email, minLength, maxLength, pattern
- **自定义规则**: bearerToken, password, tags
- **异步验证**: 支持异步验证函数
- **条件验证**: 基于其他字段的条件验证

#### 验证模式
预定义了多个表单验证模式：
- `accountFormSchema`: 账号表单验证
- `batchOperationSchema`: 批量操作验证
- `exportFormSchema`: 导出表单验证
- `importFormSchema`: 导入表单验证

#### 使用示例
```javascript
const validation = useFormValidation(accountFormSchema)

// 验证单个字段
await validation.validateField('email', 'test@example.com')

// 验证整个表单
const isValid = await validation.validateAll(formData)
```

## 集成使用

### 完整的账号管理器

使用 `createKiroAccountManager()` 创建完整的账号管理器实例：

```javascript
import { createKiroAccountManager } from '@/composables'

const accountManager = createKiroAccountManager()

// 初始化
await accountManager.initialize()

// 添加账号（带验证）
await accountManager.addAccountWithValidation(formData)

// 批量操作（带验证）
await accountManager.executeBatchOperationWithValidation('refreshTokens', {
  selectedIds: ['id1', 'id2']
})
```

### 在组件中使用

```vue
<script setup>
import { useGlobalAccountStore, useUIState } from '@/composables'

const accountStore = useGlobalAccountStore()
const uiState = useUIState()

// 响应式数据
const accounts = accountStore.filteredItems
const isLoading = accountStore.isLoading
const notifications = uiState.state.notifications

// 操作方法
async function addAccount(data) {
  try {
    await accountStore.addAccount('token', data)
    uiState.notifications.success('账号添加成功')
  } catch (error) {
    uiState.notifications.error('添加失败: ' + error.message)
  }
}
</script>
```

## 设计原则

### 1. 单一职责
每个 composable 都有明确的职责：
- `useReactiveStore`: 基础响应式存储
- `useAccountStore`: 账号数据管理
- `useUIState`: UI状态管理
- `useFormValidation`: 表单验证

### 2. 组合优于继承
通过组合多个 composable 来构建复杂功能，而不是创建大型的单体类。

### 3. 响应式优先
所有状态都是响应式的，UI 会自动响应数据变化。

### 4. 类型安全
使用 TypeScript 类型定义确保类型安全（在 .ts 版本中）。

### 5. 错误处理
统一的错误处理机制，所有异步操作都有适当的错误处理。

### 6. 性能优化
- 防抖和节流机制
- 操作去重
- 智能缓存
- 懒加载

## 测试

### 单元测试
每个 composable 都有对应的单元测试：
- `reactiveStore.test.js`: 基础存储测试
- 更多测试文件可以根据需要添加

### 手动测试
提供了 `manual-test.js` 用于在浏览器中手动测试功能。

### 集成测试
`KiroAccountManagerEnhanced.vue` 组件展示了完整的集成使用。

## 扩展指南

### 添加新的存储
```javascript
export function useCustomStore() {
  const store = createReactiveStore({
    customData: []
  })
  
  // 添加自定义方法
  async function customOperation() {
    return store.executeOperation('custom-op', async () => {
      // 自定义逻辑
    })
  }
  
  return {
    ...store,
    customOperation
  }
}
```

### 添加新的验证规则
```javascript
const customValidationRules = {
  customRule: (value) => {
    // 验证逻辑
    return isValid
  }
}

const validation = useFormValidation({
  field: {
    customRule: true
  }
})
```

### 添加新的UI状态
```javascript
const uiState = useUIState()

// 扩展状态
uiState.updateState(state => {
  state.customUIState = {
    // 自定义UI状态
  }
})
```

## 最佳实践

1. **使用全局存储**: 对于需要跨组件共享的状态，使用全局存储实例
2. **合理的粒度**: 将相关的状态和操作组织在一起，但避免过度耦合
3. **错误处理**: 始终处理异步操作的错误情况
4. **性能考虑**: 对于频繁更新的数据，考虑使用防抖或节流
5. **类型安全**: 在 TypeScript 项目中，为所有状态和方法提供类型定义
6. **测试覆盖**: 为关键的业务逻辑编写测试
7. **文档维护**: 保持文档与代码同步更新

## 故障排除

### 常见问题

1. **状态不更新**: 确保使用响应式引用，避免直接修改状态
2. **内存泄漏**: 记得在组件卸载时清理事件监听器
3. **性能问题**: 检查是否有不必要的计算属性重新计算
4. **验证不工作**: 确保验证规则正确配置，检查异步验证的处理

### 调试技巧

1. 使用 Vue DevTools 查看响应式状态
2. 在浏览器控制台运行手动测试脚本
3. 检查网络请求和错误日志
4. 使用 `console.log` 跟踪状态变化

## 版本历史

- **v1.0.0**: 初始实现，包含基础响应式存储和账号管理
- 后续版本将根据需求添加新功能和优化

## 贡献指南

1. 遵循现有的代码风格和架构模式
2. 为新功能添加相应的测试
3. 更新相关文档
4. 确保向后兼容性
5. 提交前运行所有测试