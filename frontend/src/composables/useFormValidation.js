import { ref, reactive, computed, watch } from 'vue'

/**
 * 表单验证规则
 */
const validationRules = {
  required: (value) => {
    if (Array.isArray(value)) return value.length > 0
    return value !== null && value !== undefined && String(value).trim() !== ''
  },
  
  email: (value) => {
    if (!value) return true // 可选字段
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    return emailRegex.test(value)
  },
  
  minLength: (min) => (value) => {
    if (!value) return true
    return String(value).length >= min
  },
  
  maxLength: (max) => (value) => {
    if (!value) return true
    return String(value).length <= max
  },
  
  pattern: (regex) => (value) => {
    if (!value) return true
    return regex.test(value)
  },
  
  custom: (validator) => validator,
  
  bearerToken: (value) => {
    if (!value) return false
    // Bearer Token 通常是 Base64 编码的字符串
    const tokenRegex = /^[A-Za-z0-9+/=_-]+$/
    return tokenRegex.test(value.trim()) && value.trim().length > 20
  },
  
  password: (value) => {
    if (!value) return false
    return value.length >= 6
  },
  
  tags: (value) => {
    if (!Array.isArray(value)) return false
    return value.every(tag => 
      typeof tag === 'string' && 
      tag.trim().length > 0 && 
      tag.trim().length <= 20
    )
  }
}

/**
 * 错误消息模板
 */
const errorMessages = {
  required: '此字段为必填项',
  email: '请输入有效的邮箱地址',
  minLength: (min) => `最少需要 ${min} 个字符`,
  maxLength: (max) => `最多允许 ${max} 个字符`,
  pattern: '格式不正确',
  bearerToken: '请输入有效的 Bearer Token',
  password: '密码至少需要 6 个字符',
  tags: '标签格式不正确',
  custom: '验证失败'
}

/**
 * 表单验证 Composable
 * @param {Object} schema - 验证模式
 * @param {Object} options - 选项
 */
export function useFormValidation(schema = {}, options = {}) {
  const {
    validateOnChange = true,
    validateOnBlur = true,
    showErrorsImmediately = false
  } = options

  // 验证状态
  const validationState = reactive({
    errors: {},
    touched: {},
    validating: false,
    isValid: false,
    hasErrors: false
  })

  // 字段状态
  const fieldStates = reactive({})

  /**
   * 验证单个字段
   * @param {string} field - 字段名
   * @param {any} value - 字段值
   * @param {Object} rules - 验证规则
   */
  async function validateField(field, value, rules = {}) {
    const fieldRules = rules || schema[field] || {}
    const errors = []

    for (const [ruleName, ruleConfig] of Object.entries(fieldRules)) {
      let rule, params, message

      if (typeof ruleConfig === 'boolean' && ruleConfig) {
        rule = validationRules[ruleName]
        message = errorMessages[ruleName]
      } else if (typeof ruleConfig === 'function') {
        rule = ruleConfig
        message = errorMessages.custom
      } else if (typeof ruleConfig === 'object') {
        rule = ruleConfig.validator || validationRules[ruleName]
        params = ruleConfig.params
        message = ruleConfig.message || errorMessages[ruleName]
      } else {
        continue
      }

      if (!rule) continue

      try {
        let isValid
        if (params !== undefined) {
          isValid = typeof rule === 'function' ? rule(params)(value) : rule(value, params)
        } else {
          isValid = rule(value)
        }

        // 支持异步验证
        if (isValid instanceof Promise) {
          isValid = await isValid
        }

        if (!isValid) {
          const errorMessage = typeof message === 'function' ? message(params) : message
          errors.push(errorMessage)
        }
      } catch (error) {
        console.error(`Validation error for field ${field}:`, error)
        errors.push('验证过程中发生错误')
      }
    }

    // 更新字段状态
    validationState.errors[field] = errors
    fieldStates[field] = {
      isValid: errors.length === 0,
      hasError: errors.length > 0,
      errors,
      touched: validationState.touched[field] || false
    }

    return errors.length === 0
  }

  /**
   * 验证所有字段
   * @param {Object} formData - 表单数据
   */
  async function validateAll(formData) {
    validationState.validating = true

    try {
      const validationPromises = Object.keys(schema).map(field =>
        validateField(field, formData[field])
      )

      const results = await Promise.all(validationPromises)
      const isValid = results.every(result => result)

      validationState.isValid = isValid
      validationState.hasErrors = !isValid

      return isValid
    } finally {
      validationState.validating = false
    }
  }

  /**
   * 标记字段为已触摸
   * @param {string} field - 字段名
   */
  function touchField(field) {
    validationState.touched[field] = true
    if (fieldStates[field]) {
      fieldStates[field].touched = true
    }
  }

  /**
   * 标记所有字段为已触摸
   */
  function touchAll() {
    Object.keys(schema).forEach(field => {
      touchField(field)
    })
  }

  /**
   * 重置验证状态
   */
  function resetValidation() {
    validationState.errors = {}
    validationState.touched = {}
    validationState.validating = false
    validationState.isValid = false
    validationState.hasErrors = false

    Object.keys(fieldStates).forEach(field => {
      delete fieldStates[field]
    })
  }

  /**
   * 清除字段错误
   * @param {string} field - 字段名
   */
  function clearFieldError(field) {
    if (validationState.errors[field]) {
      validationState.errors[field] = []
    }
    if (fieldStates[field]) {
      fieldStates[field].errors = []
      fieldStates[field].hasError = false
      fieldStates[field].isValid = true
    }
  }

  /**
   * 设置字段错误
   * @param {string} field - 字段名
   * @param {string|Array} errors - 错误信息
   */
  function setFieldError(field, errors) {
    const errorArray = Array.isArray(errors) ? errors : [errors]
    validationState.errors[field] = errorArray
    
    if (!fieldStates[field]) {
      fieldStates[field] = {}
    }
    
    fieldStates[field].errors = errorArray
    fieldStates[field].hasError = errorArray.length > 0
    fieldStates[field].isValid = errorArray.length === 0
  }

  /**
   * 获取字段错误信息
   * @param {string} field - 字段名
   */
  function getFieldError(field) {
    const errors = validationState.errors[field] || []
    return errors.length > 0 ? errors[0] : null
  }

  /**
   * 获取字段所有错误信息
   * @param {string} field - 字段名
   */
  function getFieldErrors(field) {
    return validationState.errors[field] || []
  }

  /**
   * 检查字段是否有错误
   * @param {string} field - 字段名
   */
  function hasFieldError(field) {
    const errors = validationState.errors[field] || []
    const touched = validationState.touched[field] || showErrorsImmediately
    return touched && errors.length > 0
  }

  /**
   * 检查字段是否有效
   * @param {string} field - 字段名
   */
  function isFieldValid(field) {
    const errors = validationState.errors[field] || []
    return errors.length === 0
  }

  /**
   * 创建字段验证器
   * @param {string} field - 字段名
   * @param {Object} formData - 表单数据（响应式对象）
   */
  function createFieldValidator(field, formData) {
    // 监听字段值变化
    if (validateOnChange) {
      watch(() => formData[field], (newValue) => {
        if (validationState.touched[field] || showErrorsImmediately) {
          validateField(field, newValue)
        }
      })
    }

    return {
      validate: (value) => validateField(field, value || formData[field]),
      touch: () => touchField(field),
      clear: () => clearFieldError(field),
      setError: (error) => setFieldError(field, error),
      
      // 计算属性
      error: computed(() => getFieldError(field)),
      errors: computed(() => getFieldErrors(field)),
      hasError: computed(() => hasFieldError(field)),
      isValid: computed(() => isFieldValid(field)),
      touched: computed(() => validationState.touched[field] || false)
    }
  }

  // 计算属性
  const hasAnyError = computed(() => {
    return Object.values(validationState.errors).some(errors => 
      Array.isArray(errors) && errors.length > 0
    )
  })

  const touchedFields = computed(() => {
    return Object.keys(validationState.touched).filter(field => 
      validationState.touched[field]
    )
  })

  const errorCount = computed(() => {
    return Object.values(validationState.errors).reduce((count, errors) => {
      return count + (Array.isArray(errors) ? errors.length : 0)
    }, 0)
  })

  return {
    // 状态
    validationState: readonly(validationState),
    fieldStates: readonly(fieldStates),
    
    // 计算属性
    hasAnyError,
    touchedFields,
    errorCount,
    
    // 方法
    validateField,
    validateAll,
    touchField,
    touchAll,
    resetValidation,
    clearFieldError,
    setFieldError,
    getFieldError,
    getFieldErrors,
    hasFieldError,
    isFieldValid,
    createFieldValidator
  }
}

/**
 * 账号表单验证模式
 */
export const accountFormSchema = {
  displayName: {
    maxLength: { params: 50, message: '显示名称不能超过50个字符' }
  },
  
  bearerToken: {
    required: true,
    bearerToken: true
  },
  
  email: {
    required: true,
    email: true
  },
  
  password: {
    required: true,
    password: true
  },
  
  notes: {
    maxLength: { params: 500, message: '备注不能超过500个字符' }
  },
  
  tags: {
    tags: true,
    custom: {
      validator: (tags) => {
        if (!Array.isArray(tags)) return true
        return tags.length <= 10
      },
      message: '最多只能添加10个标签'
    }
  }
}

/**
 * 批量操作表单验证模式
 */
export const batchOperationSchema = {
  type: {
    required: true
  },
  
  selectedIds: {
    required: true,
    custom: {
      validator: (ids) => Array.isArray(ids) && ids.length > 0,
      message: '请至少选择一个账号'
    }
  },
  
  tags: {
    custom: {
      validator: (value) => {
        if (!value) return true
        const tags = value.split(',').map(tag => tag.trim()).filter(Boolean)
        return tags.every(tag => tag.length > 0 && tag.length <= 20)
      },
      message: '标签格式不正确'
    }
  }
}

/**
 * 导出表单验证模式
 */
export const exportFormSchema = {
  password: {
    minLength: { params: 6, message: '密码至少需要6个字符' }
  }
}

/**
 * 导入表单验证模式
 */
export const importFormSchema = {
  file: {
    required: true,
    custom: {
      validator: (file) => {
        if (!file) return false
        return file.type === 'application/json' || file.name.endsWith('.json')
      },
      message: '请选择有效的JSON文件'
    }
  },
  
  password: {
    // 密码可选，取决于文件是否加密
  }
}