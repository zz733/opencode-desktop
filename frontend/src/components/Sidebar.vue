<script setup>
import { ref, onMounted, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ListDir, OpenFolder, ReadFileContent } from '../../wailsjs/go/main/App'
import FileTreeItem from './FileTreeItem.vue'

const { t } = useI18n()

const props = defineProps({
  activeTab: String,
  workDir: String
})

const emit = defineEmits(['openFile', 'update:workDir', 'runCommand'])

const localWorkDir = ref('')
const files = ref([])
const loading = ref(false)
const expandedFolders = ref(new Set())

// Maven 项目检测
const isMavenProject = ref(false)
const mavenExpanded = ref(true) // Maven 面板展开状态
const mavenInfo = ref({
  groupId: '',
  artifactId: '',
  version: '',
  name: ''
})

// 项目类型检测
const projectType = ref('')

// 检测项目类型
const detectProjectType = async () => {
  if (!localWorkDir.value) {
    projectType.value = ''
    return
  }
  
  try {
    // 检测各种项目类型
    const checks = [
      { file: 'pom.xml', type: 'maven' },
      { file: 'build.gradle', type: 'gradle' },
      { file: 'build.gradle.kts', type: 'gradle' },
      { file: 'go.mod', type: 'go' },
      { file: 'Cargo.toml', type: 'rust' },
      { file: 'package.json', type: 'node' },
      { file: 'requirements.txt', type: 'python' },
      { file: 'pyproject.toml', type: 'python' },
      { file: 'setup.py', type: 'python' },
      { file: '*.csproj', type: 'csharp' },
      { file: '*.sln', type: 'dotnet' },
    ]
    
    for (const check of checks) {
      try {
        const content = await ReadFileContent(localWorkDir.value + '/' + check.file)
        if (content) {
          projectType.value = check.type
          
          // 进一步检测 Node 项目类型
          if (check.type === 'node') {
            if (content.includes('"vue"')) projectType.value = 'vue'
            else if (content.includes('"react"')) projectType.value = 'react'
          }
          
          return
        }
      } catch (e) {
        // 文件不存在，继续检测
      }
    }
    
    projectType.value = ''
  } catch (e) {
    projectType.value = ''
  }
}

// 检测是否是 Maven 项目
const checkMavenProject = async () => {
  if (!localWorkDir.value) {
    isMavenProject.value = false
    return
  }
  
  try {
    const pomPath = localWorkDir.value + '/pom.xml'
    const content = await ReadFileContent(pomPath)
    if (content && content.includes('<project')) {
      isMavenProject.value = true
      // 简单解析 pom.xml
      const groupIdMatch = content.match(/<groupId>([^<]+)<\/groupId>/)
      const artifactIdMatch = content.match(/<artifactId>([^<]+)<\/artifactId>/)
      const versionMatch = content.match(/<version>([^<]+)<\/version>/)
      const nameMatch = content.match(/<name>([^<]+)<\/name>/)
      
      mavenInfo.value = {
        groupId: groupIdMatch?.[1] || '',
        artifactId: artifactIdMatch?.[1] || '',
        version: versionMatch?.[1] || '',
        name: nameMatch?.[1] || ''
      }
    } else {
      isMavenProject.value = false
    }
  } catch (e) {
    isMavenProject.value = false
  }
}

// Maven 命令列表
const mavenCommands = computed(() => [
  { id: 'clean', label: t('maven.clean'), cmd: 'mvn clean', icon: 'trash' },
  { id: 'compile', label: t('maven.compile'), cmd: 'mvn compile', icon: 'build' },
  { id: 'test', label: t('maven.test'), cmd: 'mvn test', icon: 'test' },
  { id: 'package', label: t('maven.package'), cmd: 'mvn package -DskipTests', icon: 'package' },
  { id: 'install', label: t('maven.install'), cmd: 'mvn install -DskipTests', icon: 'install' },
  { id: 'clean-install', label: t('maven.cleanInstall'), cmd: 'mvn clean install -DskipTests', icon: 'refresh' },
  { id: 'spring-run', label: t('maven.springRun'), cmd: 'mvn spring-boot:run', icon: 'play' },
  { id: 'tree', label: t('maven.tree'), cmd: 'mvn dependency:tree', icon: 'tree' },
])

// 执行 Maven 命令（需要 cd 到工作目录）
const runMavenCommand = (cmd) => {
  // 在工作目录执行命令
  const fullCmd = `cd "${localWorkDir.value}" && ${cmd}`
  emit('runCommand', fullCmd)
}

// 切换 Maven 面板展开状态
const toggleMavenPanel = () => {
  mavenExpanded.value = !mavenExpanded.value
}

const loadDir = async (dir = '') => {
  loading.value = true
  try {
    const targetDir = dir || localWorkDir.value || ''
    const items = await ListDir(targetDir)
    if (dir === '' || dir === localWorkDir.value) {
      files.value = items || []
    }
    return items || []
  } catch (e) {
    console.error('加载目录失败:', e)
    return []
  } finally {
    loading.value = false
  }
}

const refreshDir = async (dir) => {
  localWorkDir.value = dir
  expandedFolders.value.clear()
  await loadDir()
  await checkMavenProject()
}

defineExpose({ refreshDir })

watch(() => props.workDir, async (newDir) => {
  if (newDir && newDir !== localWorkDir.value) {
    localWorkDir.value = newDir
    expandedFolders.value.clear()
    await loadDir()
    await checkMavenProject()
    await detectProjectType()
  }
})

onMounted(async () => {
  if (props.workDir) {
    localWorkDir.value = props.workDir
    await loadDir()
    await checkMavenProject()
    await detectProjectType()
  }
})

const toggleFolder = async (item) => {
  if (item.type !== 'folder') return
  const path = item.path
  if (expandedFolders.value.has(path)) {
    expandedFolders.value.delete(path)
  } else {
    expandedFolders.value.add(path)
    if (!item.children || item.children.length === 0) {
      item.children = await loadDir(path)
    }
  }
  // 触发响应式更新
  expandedFolders.value = new Set(expandedFolders.value)
}

const openFile = (item) => {
  if (item.type === 'file') {
    emit('openFile', item)
  }
}

// 刷新文件树
const refreshFileTree = async () => {
  expandedFolders.value.clear()
  await loadDir()
}

const selectFolder = async () => {
  const dir = await OpenFolder()
  if (dir) {
    localWorkDir.value = dir
    expandedFolders.value.clear()
    emit('update:workDir', dir)
    await loadDir()
  }
}

const getDirName = () => {
  if (!localWorkDir.value) return ''
  return localWorkDir.value.split('/').pop() || localWorkDir.value.split('\\').pop() || 'ROOT'
}
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <span>{{ activeTab === 'files' ? t('sidebar.explorer').toUpperCase() : t('sidebar.' + activeTab).toUpperCase() }}</span>
    </div>
    
    <div v-if="activeTab === 'files'" class="file-tree">
      <div v-if="!localWorkDir" class="empty-tree">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
        </svg>
        <p>{{ t('sidebar.noFolder') }}</p>
        <button @click="selectFolder">{{ t('editor.openFolder') }}</button>
      </div>
      
      <div v-else class="file-tree-content">
        <div class="section-header">
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 9l6 6 6-6"/></svg>
          <span class="dir-name">{{ getDirName().toUpperCase() }}</span>
          <button class="btn-open-folder" @click="selectFolder">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
              <line x1="12" y1="11" x2="12" y2="17"/><line x1="9" y1="14" x2="15" y2="14"/>
            </svg>
          </button>
        </div>
      
        <div class="tree-items" v-if="files.length">
          <FileTreeItem
            v-for="item in files"
            :key="item.path"
            :item="item"
            :depth="0"
            :expandedFolders="expandedFolders"
            :projectType="projectType"
            @openFile="openFile"
            @toggleFolder="toggleFolder"
            @refresh="refreshFileTree"
          />
        </div>
        <div v-else-if="!loading" class="empty-files"><p>{{ t('sidebar.emptyFolder') }}</p></div>
        
        <!-- Maven 面板 -->
        <div v-if="isMavenProject" class="maven-panel">
          <div class="section-header maven-header" @click="toggleMavenPanel">
            <svg :class="['collapse-icon', { collapsed: !mavenExpanded }]" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M6 9l6 6 6-6"/></svg>
            <svg class="maven-icon" width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>
            </svg>
            <span>MAVEN</span>
          </div>
          
          <template v-if="mavenExpanded">
            <div class="maven-info" v-if="mavenInfo.artifactId">
              <span class="artifact-name">{{ mavenInfo.artifactId }}</span>
              <span class="artifact-version">{{ mavenInfo.version }}</span>
            </div>
            
            <div class="maven-commands">
              <div 
                v-for="cmd in mavenCommands" 
                :key="cmd.id" 
                class="maven-cmd"
                @click="runMavenCommand(cmd.cmd)"
                :title="cmd.cmd"
              >
                <!-- 图标 -->
                <svg v-if="cmd.icon === 'trash'" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                </svg>
                <svg v-else-if="cmd.icon === 'build'" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/>
                </svg>
                <svg v-else-if="cmd.icon === 'test'" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/><path d="m9 15 2 2 4-4"/>
                </svg>
                <svg v-else-if="cmd.icon === 'package'" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M16.5 9.4l-9-5.19M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/>
                </svg>
                <svg v-else-if="cmd.icon === 'install'" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
                </svg>
                <svg v-else-if="cmd.icon === 'refresh'" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
                </svg>
                <svg v-else-if="cmd.icon === 'play'" width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M8 5v14l11-7z"/>
                </svg>
                <svg v-else-if="cmd.icon === 'tree'" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M17 18a2 2 0 0 0-2-2H9a2 2 0 0 0-2 2"/><rect x="3" y="4" width="18" height="18" rx="2"/><circle cx="9" cy="10" r="2"/><path d="M15 8h.01"/>
                </svg>
                <span>{{ cmd.label }}</span>
              </div>
            </div>
          </template>
        </div>
      </div>
    </div>
    <div v-else class="placeholder"><p>{{ activeTab }}</p></div>
  </aside>
</template>

<style scoped>
.sidebar { flex: 1; background: var(--bg-surface); display: flex; flex-direction: column; overflow: hidden; }
.sidebar-header { padding: 10px 20px; font-size: 11px; font-weight: 400; letter-spacing: 1.2px; color: var(--text-secondary); }
.file-tree { flex: 1; overflow-y: auto; display: flex; flex-direction: column; }
.file-tree-content { flex: 1; display: flex; flex-direction: column; }
.section-header { display: flex; align-items: center; gap: 4px; padding: 4px 8px; font-size: 11px; font-weight: 700; color: var(--text-primary); }
.dir-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.btn-open-folder { background: transparent; border: none; color: var(--text-muted); cursor: pointer; padding: 2px; border-radius: 4px; }
.btn-open-folder:hover { background: var(--bg-active); color: var(--text-primary); }
.tree-items { padding-left: 8px; flex: 1; overflow-y: auto; }
.empty-tree { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 40px 20px; color: var(--text-muted); font-size: 13px; }
.empty-tree svg { opacity: 0.3; margin-bottom: 12px; }
.empty-tree button { margin-top: 12px; padding: 6px 12px; background: var(--accent-primary); border: none; border-radius: 4px; color: white; font-size: 12px; cursor: pointer; }
.empty-tree button:hover { background: var(--accent-hover); }
.empty-files { padding: 20px; text-align: center; color: var(--text-muted); font-size: 12px; }
.placeholder { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--text-muted); font-size: 13px; }

/* Maven 面板样式 */
.maven-panel {
  border-top: 1px solid var(--border-default);
  margin-top: 8px;
}

.maven-header {
  padding: 8px;
  gap: 6px;
  cursor: pointer;
}

.maven-header:hover {
  background: var(--bg-hover);
}

.collapse-icon {
  transition: transform 0.15s;
}

.collapse-icon.collapsed {
  transform: rotate(-90deg);
}

.maven-icon {
  color: var(--yellow);
}

.maven-info {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 12px 8px;
  font-size: 11px;
}

.artifact-name {
  color: var(--text-primary);
  font-weight: 500;
}

.artifact-version {
  color: var(--text-muted);
  background: var(--bg-elevated);
  padding: 1px 6px;
  border-radius: 3px;
}

.maven-commands {
  padding: 0 4px 8px;
}

.maven-cmd {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 8px;
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.1s;
}

.maven-cmd:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.maven-cmd svg {
  flex-shrink: 0;
  opacity: 0.7;
}

.maven-cmd:hover svg {
  opacity: 1;
}
</style>
