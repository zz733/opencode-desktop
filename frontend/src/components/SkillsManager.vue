<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { 
  GetSkills, GetSkillTemplates, CreateSkill, UpdateSkill, DeleteSkill, CreateSkillFromTemplate 
} from '../../wailsjs/go/main/App'

const { t } = useI18n()

// 状态
const skills = ref([])
const templates = ref([])
const loading = ref(false)
const showAddDialog = ref(false)
const showEditDialog = ref(false)
const showTemplateDialog = ref(false)
const showConfirmDialog = ref(false)
const confirmTarget = ref(null)
const editingSkill = ref(null)

// 表单
const skillForm = ref({
  name: '',
  description: '',
  content: '',
  global: false
})

const selectedTemplate = ref(null)
const templateCustomName = ref('')
const templateGlobal = ref(false)

// 分类
const categoryNames = {
  development: '开发',
  documentation: '文档',
  testing: '测试',
  architecture: '架构'
}

// 按来源分组的技能
const groupedSkills = computed(() => {
  const groups = { project: [], global: [] }
  skills.value.forEach(skill => {
    if (skill.source === 'project') {
      groups.project.push(skill)
    } else {
      groups.global.push(skill)
    }
  })
  return groups
})

// 按分类分组的模板
const groupedTemplates = computed(() => {
  const groups = {}
  templates.value.forEach(template => {
    const cat = template.category || 'other'
    if (!groups[cat]) groups[cat] = []
    groups[cat].push(template)
  })
  return groups
})

// 加载技能列表
async function loadSkills() {
  loading.value = true
  try {
    const [skillList, templateList] = await Promise.all([
      GetSkills(),
      GetSkillTemplates()
    ])
    skills.value = skillList || []
    templates.value = templateList || []
  } catch (e) {
    console.error('加载技能失败:', e)
  } finally {
    loading.value = false
  }
}

// 打开添加对话框
function openAddDialog() {
  skillForm.value = { name: '', description: '', content: '', global: false }
  showAddDialog.value = true
}

// 打开编辑对话框
function openEditDialog(skill) {
  editingSkill.value = skill
  skillForm.value = {
    name: skill.name,
    description: skill.description,
    content: skill.content,
    global: skill.source === 'global'
  }
  showEditDialog.value = true
}

// 打开模板对话框
function openTemplateDialog() {
  selectedTemplate.value = null
  templateCustomName.value = ''
  templateGlobal.value = false
  showTemplateDialog.value = true
}

// 选择模板
function selectTemplate(template) {
  selectedTemplate.value = template
  templateCustomName.value = template.id
}

// 保存新技能
async function saveSkill() {
  if (!skillForm.value.name || !skillForm.value.description) return
  
  try {
    await CreateSkill(
      skillForm.value.name,
      skillForm.value.description,
      skillForm.value.content,
      skillForm.value.global
    )
    showAddDialog.value = false
    await loadSkills()
  } catch (e) {
    console.error('创建技能失败:', e)
    alert('创建失败: ' + e)
  }
}

// 更新技能
async function updateSkill() {
  if (!skillForm.value.description) return
  
  try {
    await UpdateSkill(
      editingSkill.value.name,
      skillForm.value.description,
      skillForm.value.content
    )
    showEditDialog.value = false
    await loadSkills()
  } catch (e) {
    console.error('更新技能失败:', e)
    alert('更新失败: ' + e)
  }
}

// 从模板创建
async function createFromTemplate() {
  if (!selectedTemplate.value) return
  
  try {
    await CreateSkillFromTemplate(
      selectedTemplate.value.id,
      templateCustomName.value,
      templateGlobal.value
    )
    showTemplateDialog.value = false
    await loadSkills()
  } catch (e) {
    console.error('从模板创建失败:', e)
    alert('创建失败: ' + e)
  }
}

// 确认删除
function askDeleteSkill(skill) {
  confirmTarget.value = skill
  showConfirmDialog.value = true
}

// 执行删除
async function confirmDeleteSkill() {
  const skill = confirmTarget.value
  showConfirmDialog.value = false
  confirmTarget.value = null
  
  try {
    await DeleteSkill(skill.name)
    await loadSkills()
  } catch (e) {
    console.error('删除技能失败:', e)
    alert('删除失败: ' + e)
  }
}

onMounted(() => {
  loadSkills()
})
</script>

<template>
  <div class="skills-manager">
    <!-- 头部 -->
    <div class="skills-header">
      <div class="header-title">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/>
        </svg>
        <span>技能管理</span>
      </div>
      <div class="header-actions">
        <button class="btn-icon" @click="openTemplateDialog" title="从模板创建">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/>
          </svg>
        </button>
        <button class="btn-icon" @click="openAddDialog" title="手动创建">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 5v14M5 12h14"/>
          </svg>
        </button>
        <button class="btn-icon" @click="loadSkills" title="刷新">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M23 4v6h-6"/><path d="M1 20v-6h6"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="loading">加载中...</div>

    <!-- 技能列表 -->
    <div v-else class="skills-content">
      <!-- 项目技能 -->
      <div v-if="groupedSkills.project.length > 0" class="skill-group">
        <div class="group-title">
          <span class="group-icon">📁</span>
          项目技能
          <span class="group-count">{{ groupedSkills.project.length }}</span>
        </div>
        <div class="skill-list">
          <div v-for="skill in groupedSkills.project" :key="skill.name" class="skill-item">
            <div class="skill-info">
              <div class="skill-name">{{ skill.name }}</div>
              <div class="skill-desc">{{ skill.description }}</div>
            </div>
            <div class="skill-actions">
              <button class="btn-icon small" @click="openEditDialog(skill)" title="编辑">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                  <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                </svg>
              </button>
              <button class="btn-icon small danger" @click="askDeleteSkill(skill)" title="删除">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 全局技能 -->
      <div v-if="groupedSkills.global.length > 0" class="skill-group">
        <div class="group-title">
          <span class="group-icon">🌐</span>
          全局技能
          <span class="group-count">{{ groupedSkills.global.length }}</span>
        </div>
        <div class="skill-list">
          <div v-for="skill in groupedSkills.global" :key="skill.name" class="skill-item">
            <div class="skill-info">
              <div class="skill-name">{{ skill.name }}</div>
              <div class="skill-desc">{{ skill.description }}</div>
            </div>
            <div class="skill-actions">
              <button class="btn-icon small" @click="openEditDialog(skill)" title="编辑">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                  <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                </svg>
              </button>
              <button class="btn-icon small danger" @click="askDeleteSkill(skill)" title="删除">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-if="skills.length === 0" class="empty-state">
        <div class="empty-icon">🎯</div>
        <div class="empty-title">还没有技能</div>
        <div class="empty-desc">技能是 AI 可复用的指令集，帮助 Agent 更好地完成特定任务</div>
        <div class="empty-actions">
          <button class="btn-primary" @click="openTemplateDialog">从模板创建</button>
          <button class="btn-secondary" @click="openAddDialog">手动创建</button>
        </div>
      </div>
    </div>

    <!-- 添加对话框 -->
    <div v-if="showAddDialog" class="dialog-overlay" @click.self="showAddDialog = false">
      <div class="dialog">
        <div class="dialog-header">创建技能</div>
        <div class="dialog-content">
          <div class="form-group">
            <label>技能名称 <span class="required">*</span></label>
            <input v-model="skillForm.name" type="text" placeholder="小写字母、数字、连字符，如 code-review">
            <div class="form-hint">1-64 字符，只能包含小写字母、数字和连字符</div>
          </div>
          <div class="form-group">
            <label>描述 <span class="required">*</span></label>
            <input v-model="skillForm.description" type="text" placeholder="简短描述技能的功能">
            <div class="form-hint">1-1024 字符，帮助 Agent 理解何时使用此技能</div>
          </div>
          <div class="form-group">
            <label>技能内容</label>
            <textarea v-model="skillForm.content" rows="10" placeholder="技能的详细指令..."></textarea>
          </div>
          <div class="form-group checkbox-group">
            <label>
              <input v-model="skillForm.global" type="checkbox">
              保存为全局技能（所有项目可用）
            </label>
          </div>
        </div>
        <div class="dialog-footer">
          <button class="btn-cancel" @click="showAddDialog = false">取消</button>
          <button class="btn-save" @click="saveSkill">创建</button>
        </div>
      </div>
    </div>

    <!-- 编辑对话框 -->
    <div v-if="showEditDialog" class="dialog-overlay" @click.self="showEditDialog = false">
      <div class="dialog">
        <div class="dialog-header">编辑技能: {{ editingSkill?.name }}</div>
        <div class="dialog-content">
          <div class="form-group">
            <label>描述 <span class="required">*</span></label>
            <input v-model="skillForm.description" type="text">
          </div>
          <div class="form-group">
            <label>技能内容</label>
            <textarea v-model="skillForm.content" rows="12"></textarea>
          </div>
        </div>
        <div class="dialog-footer">
          <button class="btn-cancel" @click="showEditDialog = false">取消</button>
          <button class="btn-save" @click="updateSkill">保存</button>
        </div>
      </div>
    </div>

    <!-- 模板对话框 -->
    <div v-if="showTemplateDialog" class="dialog-overlay" @click.self="showTemplateDialog = false">
      <div class="dialog template-dialog">
        <div class="dialog-header">从模板创建技能</div>
        <div class="dialog-content">
          <div class="template-list">
            <template v-for="(items, category) in groupedTemplates" :key="category">
              <div class="template-category">{{ categoryNames[category] || category }}</div>
              <div 
                v-for="template in items" 
                :key="template.id" 
                :class="['template-item', { selected: selectedTemplate?.id === template.id }]"
                @click="selectTemplate(template)"
              >
                <div class="template-name">{{ template.name }}</div>
                <div class="template-desc">{{ template.description }}</div>
              </div>
            </template>
          </div>
          
          <div v-if="selectedTemplate" class="template-config">
            <div class="form-group">
              <label>技能名称</label>
              <input v-model="templateCustomName" type="text" :placeholder="selectedTemplate.id">
            </div>
            <div class="form-group checkbox-group">
              <label>
                <input v-model="templateGlobal" type="checkbox">
                保存为全局技能
              </label>
            </div>
          </div>
        </div>
        <div class="dialog-footer">
          <button class="btn-cancel" @click="showTemplateDialog = false">取消</button>
          <button class="btn-save" @click="createFromTemplate" :disabled="!selectedTemplate">创建</button>
        </div>
      </div>
    </div>

    <!-- 删除确认对话框 -->
    <div v-if="showConfirmDialog" class="dialog-overlay" @click.self="showConfirmDialog = false">
      <div class="dialog confirm-dialog">
        <div class="dialog-header">确认删除</div>
        <div class="dialog-content">
          <p class="confirm-message">确定要删除技能 "{{ confirmTarget?.name }}" 吗？此操作不可恢复。</p>
        </div>
        <div class="dialog-footer">
          <button class="btn-cancel" @click="showConfirmDialog = false">取消</button>
          <button class="btn-danger" @click="confirmDeleteSkill">删除</button>
        </div>
      </div>
    </div>
  </div>
</template>


<style scoped>
.skills-manager {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.skills-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.header-actions {
  display: flex;
  gap: 4px;
}

.btn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: 4px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.btn-icon:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.btn-icon.small {
  width: 24px;
  height: 24px;
}

.btn-icon.danger:hover {
  background: var(--red);
  color: white;
  border-color: var(--red);
}

.loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}

.skills-content {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.skill-group {
  margin-bottom: 16px;
}

.group-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 8px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.group-icon {
  font-size: 14px;
}

.group-count {
  background: var(--bg-elevated);
  padding: 2px 6px;
  border-radius: 10px;
  font-size: 10px;
}

.skill-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.skill-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  background: var(--bg-elevated);
  border-radius: 6px;
  border: 1px solid var(--border-subtle);
}

.skill-info {
  flex: 1;
  min-width: 0;
}

.skill-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--accent-primary);
  font-family: monospace;
}

.skill-desc {
  font-size: 11px;
  color: var(--text-secondary);
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.skill-actions {
  display: flex;
  gap: 4px;
  margin-left: 8px;
}

/* 空状态 */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  text-align: center;
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.empty-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.empty-desc {
  font-size: 13px;
  color: var(--text-secondary);
  max-width: 300px;
  margin-bottom: 20px;
}

.empty-actions {
  display: flex;
  gap: 8px;
}

.btn-primary {
  padding: 8px 16px;
  background: var(--accent-primary);
  border: none;
  border-radius: 6px;
  color: white;
  font-size: 13px;
  cursor: pointer;
  transition: opacity 0.15s;
}

.btn-primary:hover {
  opacity: 0.9;
}

.btn-secondary {
  padding: 8px 16px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-secondary:hover {
  background: var(--bg-hover);
}

/* 对话框 */
.dialog-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.dialog {
  width: 480px;
  max-height: 80vh;
  background: var(--bg-surface);
  border-radius: 8px;
  border: 1px solid var(--border-default);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  display: flex;
  flex-direction: column;
}

.template-dialog {
  width: 560px;
}

.confirm-dialog {
  width: 360px;
}

.dialog-header {
  padding: 16px;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border-subtle);
}

.dialog-content {
  padding: 16px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 12px;
  color: var(--text-secondary);
}

.required {
  color: var(--red);
}

.form-hint {
  font-size: 11px;
  color: var(--text-muted);
}

.form-group input[type="text"],
.form-group textarea {
  padding: 8px 10px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 13px;
  outline: none;
  font-family: inherit;
}

.form-group input:focus,
.form-group textarea:focus {
  border-color: var(--accent-primary);
}

.form-group textarea {
  resize: vertical;
  min-height: 100px;
  font-family: monospace;
  font-size: 12px;
  line-height: 1.5;
}

.checkbox-group label {
  flex-direction: row;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.checkbox-group input[type="checkbox"] {
  width: 16px;
  height: 16px;
}

.dialog-footer {
  padding: 12px 16px;
  border-top: 1px solid var(--border-subtle);
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.btn-cancel,
.btn-save,
.btn-danger {
  padding: 6px 16px;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-cancel {
  background: transparent;
  border: 1px solid var(--border-default);
  color: var(--text-secondary);
}

.btn-cancel:hover {
  background: var(--bg-hover);
}

.btn-save {
  background: var(--accent-primary);
  border: none;
  color: white;
}

.btn-save:hover {
  opacity: 0.9;
}

.btn-save:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-danger {
  background: var(--red);
  border: none;
  color: white;
}

.btn-danger:hover {
  opacity: 0.9;
}

.confirm-message {
  font-size: 13px;
  color: var(--text-primary);
  text-align: center;
  margin: 0;
}

/* 模板列表 */
.template-list {
  max-height: 300px;
  overflow-y: auto;
  margin-bottom: 12px;
}

.template-category {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-top: 12px;
  margin-bottom: 6px;
}

.template-category:first-child {
  margin-top: 0;
}

.template-item {
  padding: 10px 12px;
  background: var(--bg-elevated);
  border-radius: 6px;
  border: 1px solid var(--border-subtle);
  margin-bottom: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.template-item:hover {
  border-color: var(--text-muted);
}

.template-item.selected {
  border-color: var(--accent-primary);
  background: rgba(176, 128, 255, 0.1);
}

.template-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
}

.template-desc {
  font-size: 11px;
  color: var(--text-secondary);
  margin-top: 2px;
}

.template-config {
  padding-top: 12px;
  border-top: 1px solid var(--border-subtle);
}
</style>
