package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// SkillInfo 技能信息
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Source      string `json:"source"` // "project" | "global"
	Content     string `json:"content,omitempty"`
	Enabled     bool   `json:"enabled"`
}

// SkillFrontmatter SKILL.md 的 frontmatter
type SkillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// SkillTemplate 技能模板
type SkillTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Category    string `json:"category"`
}

// SkillsManager 技能管理器
type SkillsManager struct {
	workDir string
}

// NewSkillsManager 创建技能管理器
func NewSkillsManager(workDir string) *SkillsManager {
	return &SkillsManager{
		workDir: workDir,
	}
}

// SetWorkDir 设置工作目录
func (sm *SkillsManager) SetWorkDir(dir string) {
	sm.workDir = dir
}

// GetSkillsDirectories 获取技能目录列表
func (sm *SkillsManager) GetSkillsDirectories() []string {
	dirs := []string{}

	// 全局目录
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		// OpenCode 全局技能目录
		dirs = append(dirs, filepath.Join(homeDir, ".config", "opencode", "skills"))
		// Claude 兼容目录
		dirs = append(dirs, filepath.Join(homeDir, ".claude", "skills"))
	}

	// 项目目录
	if sm.workDir != "" {
		// OpenCode 项目技能目录
		dirs = append(dirs, filepath.Join(sm.workDir, ".opencode", "skills"))
		// Claude 兼容目录
		dirs = append(dirs, filepath.Join(sm.workDir, ".claude", "skills"))
	}

	return dirs
}

// ListSkills 列出所有技能
func (sm *SkillsManager) ListSkills() ([]SkillInfo, error) {
	skills := []SkillInfo{}
	seen := make(map[string]bool)

	dirs := sm.GetSkillsDirectories()

	for _, dir := range dirs {
		source := "global"
		if strings.Contains(dir, sm.workDir) && sm.workDir != "" {
			source = "project"
		}

		// 检查目录是否存在
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		// 遍历技能目录
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			skillName := entry.Name()
			skillPath := filepath.Join(dir, skillName)
			skillFile := filepath.Join(skillPath, "SKILL.md")

			// 检查 SKILL.md 是否存在
			if _, err := os.Stat(skillFile); os.IsNotExist(err) {
				continue
			}

			// 避免重复（项目级覆盖全局级）
			if seen[skillName] {
				continue
			}
			seen[skillName] = true

			// 读取技能内容
			content, err := os.ReadFile(skillFile)
			if err != nil {
				continue
			}

			// 解析 frontmatter
			frontmatter, body := parseFrontmatter(string(content))

			skill := SkillInfo{
				Name:        frontmatter.Name,
				Description: frontmatter.Description,
				Path:        skillPath,
				Source:      source,
				Content:     body,
				Enabled:     true, // 默认启用
			}

			// 如果 frontmatter 中没有 name，使用目录名
			if skill.Name == "" {
				skill.Name = skillName
			}

			skills = append(skills, skill)
		}
	}

	return skills, nil
}

// GetSkill 获取单个技能详情
func (sm *SkillsManager) GetSkill(name string) (*SkillInfo, error) {
	skills, err := sm.ListSkills()
	if err != nil {
		return nil, err
	}

	for _, skill := range skills {
		if skill.Name == name {
			return &skill, nil
		}
	}

	return nil, fmt.Errorf("skill not found: %s", name)
}

// CreateSkill 创建新技能
func (sm *SkillsManager) CreateSkill(name, description, content string, global bool) error {
	// 验证技能名称
	if !isValidSkillName(name) {
		return fmt.Errorf("invalid skill name: must be lowercase alphanumeric with hyphens, 1-64 chars")
	}

	// 验证描述长度
	if len(description) < 1 || len(description) > 1024 {
		return fmt.Errorf("description must be 1-1024 characters")
	}

	// 确定目标目录
	var targetDir string
	if global {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot get home directory: %w", err)
		}
		targetDir = filepath.Join(homeDir, ".config", "opencode", "skills", name)
	} else {
		if sm.workDir == "" {
			return fmt.Errorf("work directory not set")
		}
		targetDir = filepath.Join(sm.workDir, ".opencode", "skills", name)
	}

	// 检查是否已存在
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		return fmt.Errorf("skill already exists: %s", name)
	}

	// 创建目录
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// 生成 SKILL.md 内容
	skillContent := fmt.Sprintf(`---
name: %s
description: %s
---

%s
`, name, description, content)

	// 写入文件
	skillFile := filepath.Join(targetDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte(skillContent), 0644); err != nil {
		// 清理目录
		os.RemoveAll(targetDir)
		return fmt.Errorf("failed to write SKILL.md: %w", err)
	}

	return nil
}

// UpdateSkill 更新技能
func (sm *SkillsManager) UpdateSkill(name, description, content string) error {
	skill, err := sm.GetSkill(name)
	if err != nil {
		return err
	}

	// 验证描述长度
	if len(description) < 1 || len(description) > 1024 {
		return fmt.Errorf("description must be 1-1024 characters")
	}

	// 生成新的 SKILL.md 内容
	skillContent := fmt.Sprintf(`---
name: %s
description: %s
---

%s
`, name, description, content)

	// 写入文件
	skillFile := filepath.Join(skill.Path, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte(skillContent), 0644); err != nil {
		return fmt.Errorf("failed to update SKILL.md: %w", err)
	}

	return nil
}

// DeleteSkill 删除技能
func (sm *SkillsManager) DeleteSkill(name string) error {
	skill, err := sm.GetSkill(name)
	if err != nil {
		return err
	}

	// 删除整个技能目录
	if err := os.RemoveAll(skill.Path); err != nil {
		return fmt.Errorf("failed to delete skill: %w", err)
	}

	return nil
}

// GetSkillTemplates 获取技能模板列表
func (sm *SkillsManager) GetSkillTemplates() []SkillTemplate {
	return []SkillTemplate{
		{
			ID:          "code-review",
			Name:        "代码审查",
			Description: "审查代码质量，检查潜在问题和改进建议",
			Category:    "development",
			Content: `# 代码审查技能

## 任务
审查提供的代码，关注以下方面：

1. **代码质量**
   - 可读性和命名规范
   - 代码结构和组织
   - 注释和文档

2. **潜在问题**
   - Bug 和逻辑错误
   - 安全漏洞
   - 性能问题

3. **最佳实践**
   - 设计模式使用
   - 错误处理
   - 测试覆盖

## 输出格式
- 问题列表（按严重程度排序：Critical > High > Medium > Low）
- 每个问题包含：位置、描述、建议修复方案
- 总体评价和改进建议`,
		},
		{
			ID:          "doc-generator",
			Name:        "文档生成",
			Description: "为代码生成清晰的文档和注释",
			Category:    "documentation",
			Content: `# 文档生成技能

## 任务
为提供的代码生成文档，包括：

1. **函数/方法文档**
   - 功能描述
   - 参数说明
   - 返回值说明
   - 使用示例

2. **类/模块文档**
   - 概述
   - 属性说明
   - 方法列表
   - 使用场景

3. **README 文档**
   - 项目介绍
   - 安装说明
   - 使用方法
   - API 参考

## 输出格式
根据代码语言使用对应的文档格式（JSDoc、GoDoc、Docstring 等）`,
		},
		{
			ID:          "bug-fix",
			Name:        "Bug 修复",
			Description: "分析和修复代码中的 Bug",
			Category:    "development",
			Content: `# Bug 修复技能

## 任务
分析并修复代码中的 Bug：

1. **问题分析**
   - 复现步骤
   - 根本原因
   - 影响范围

2. **修复方案**
   - 最小改动原则
   - 不引入新问题
   - 保持向后兼容

3. **验证**
   - 测试用例
   - 边界条件
   - 回归测试

## 输出格式
- 问题描述
- 根因分析
- 修复代码（diff 格式）
- 测试建议`,
		},
		{
			ID:          "test-writer",
			Name:        "测试编写",
			Description: "为代码编写单元测试和集成测试",
			Category:    "testing",
			Content: `# 测试编写技能

## 任务
为提供的代码编写测试：

1. **单元测试**
   - 覆盖所有公共方法
   - 正常路径测试
   - 边界条件测试
   - 错误处理测试

2. **测试结构**
   - Arrange-Act-Assert 模式
   - 清晰的测试命名
   - 独立的测试用例

3. **Mock 和 Stub**
   - 外部依赖模拟
   - 数据库模拟
   - API 模拟

## 输出格式
使用项目对应的测试框架（Jest、Go testing、pytest 等）`,
		},
		{
			ID:          "refactor",
			Name:        "代码重构",
			Description: "重构代码以提高可读性和可维护性",
			Category:    "development",
			Content: `# 代码重构技能

## 任务
重构代码以提高质量：

1. **重构目标**
   - 提高可读性
   - 减少重复
   - 简化复杂度
   - 改善性能

2. **重构技术**
   - 提取方法/函数
   - 重命名变量
   - 简化条件表达式
   - 移除死代码

3. **安全重构**
   - 保持功能不变
   - 小步迭代
   - 测试验证

## 输出格式
- 重构前后对比
- 改进说明
- 测试验证结果`,
		},
		{
			ID:          "api-design",
			Name:        "API 设计",
			Description: "设计 RESTful API 或 GraphQL Schema",
			Category:    "architecture",
			Content: `# API 设计技能

## 任务
设计清晰、一致的 API：

1. **RESTful 设计**
   - 资源命名
   - HTTP 方法使用
   - 状态码规范
   - 版本控制

2. **请求/响应**
   - 数据格式
   - 分页设计
   - 错误处理
   - 认证授权

3. **文档**
   - OpenAPI/Swagger
   - 示例请求
   - 错误码说明

## 输出格式
- API 端点列表
- 请求/响应示例
- OpenAPI 规范（可选）`,
		},
	}
}

// CreateSkillFromTemplate 从模板创建技能
func (sm *SkillsManager) CreateSkillFromTemplate(templateID, customName string, global bool) error {
	templates := sm.GetSkillTemplates()

	var template *SkillTemplate
	for _, t := range templates {
		if t.ID == templateID {
			template = &t
			break
		}
	}

	if template == nil {
		return fmt.Errorf("template not found: %s", templateID)
	}

	name := customName
	if name == "" {
		name = template.ID
	}

	return sm.CreateSkill(name, template.Description, template.Content, global)
}

// parseFrontmatter 解析 SKILL.md 的 frontmatter
func parseFrontmatter(content string) (SkillFrontmatter, string) {
	var fm SkillFrontmatter

	// 检查是否以 --- 开头
	if !strings.HasPrefix(content, "---") {
		return fm, content
	}

	// 查找结束的 ---
	parts := strings.SplitN(content[3:], "---", 2)
	if len(parts) != 2 {
		return fm, content
	}

	// 解析 YAML
	yaml.Unmarshal([]byte(parts[0]), &fm)

	// 返回 body 部分
	body := strings.TrimSpace(parts[1])
	return fm, body
}

// isValidSkillName 验证技能名称
func isValidSkillName(name string) bool {
	if len(name) < 1 || len(name) > 64 {
		return false
	}

	// 必须是小写字母、数字、连字符
	matched, _ := regexp.MatchString(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`, name)
	return matched
}
