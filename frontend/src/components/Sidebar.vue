<script setup>
import { ref, onMounted, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ListDir, OpenFolder, ReadFileContent, SearchInFiles, ReplaceInFiles, GetGitStatus, GitAdd, GitCommit, GitPush, GitPull, GitDiscard } from '../../wailsjs/go/main/App'
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

// 搜索相关
const searchQuery = ref('')
const searchResults = ref([])
const searching = ref(false)
const searchCaseSensitive = ref(false)
const searchRegex = ref(false)
const showReplace = ref(false)
const replaceText = ref('')
const replacing = ref(false)

// Git 相关
const gitStatus = ref(null)
const gitBranch = ref('')
const gitChanges = ref([])
const loadingGit = ref(false)

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

// 检测项目类型（异步，不阻塞）
const detectProjectType = async () => {
  if (!localWorkDir.value) {
    projectType.value = ''
    return
  }
  
  // 简单检测，不阻塞
  const checks = [
    { file: 'pom.xml', type: 'maven' },
    { file: 'build.gradle', type: 'gradle' },
    { file: 'go.mod', type: 'go' },
    { file: 'Cargo.toml', type: 'rust' },
    { file: 'package.json', type: 'node' },
    { file: 'requirements.txt', type: 'python' },
    { file: 'pyproject.toml', type: 'python' },
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
      // 忽略错误，继续检测
    }
  }
  projectType.value = ''
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

// 刷新指定文件夹（保持展开状态）
const refreshFolder = async (folderPath) => {
  // 递归查找并刷新文件夹
  const refreshItem = async (items, targetPath) => {
    for (const item of items) {
      if (item.path === targetPath && item.type === 'folder') {
        // 重新加载这个文件夹的内容
        item.children = await loadDir(targetPath)
        return true
      }
      if (item.children && item.children.length > 0) {
        if (await refreshItem(item.children, targetPath)) {
          return true
        }
      }
    }
    return false
  }
  
  await refreshItem(files.value, folderPath)
  // 触发响应式更新
  files.value = [...files.value]
}

// 刷新文件树（完全刷新）
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

// 搜索功能
const performSearch = async () => {
  if (!searchQuery.value || !localWorkDir.value) {
    searchResults.value = []
    return
  }
  
  searching.value = true
  try {
    const results = await SearchInFiles(
      localWorkDir.value,
      searchQuery.value,
      searchCaseSensitive.value,
      searchRegex.value
    )
    searchResults.value = results || []
  } catch (e) {
    console.error('搜索失败:', e)
    searchResults.value = []
  } finally {
    searching.value = false
  }
}

const openSearchResult = (result) => {
  emit('openFile', {
    path: result.path,
    name: result.path.split('/').pop() || result.path.split('\\').pop(),
    type: 'file'
  })
}

// 替换功能
const performReplace = async () => {
  if (!searchQuery.value || !localWorkDir.value) {
    return
  }
  
  if (!confirm(t('search.replaceConfirm', { 
    search: searchQuery.value, 
    replace: replaceText.value 
  }))) {
    return
  }
  
  replacing.value = true
  try {
    const count = await ReplaceInFiles(
      localWorkDir.value,
      searchQuery.value,
      replaceText.value,
      searchCaseSensitive.value
    )
    alert(t('search.replaceSuccess', { count }))
    // 重新搜索以更新结果
    await performSearch()
  } catch (e) {
    console.error('替换失败:', e)
    alert(t('search.replaceFailed') + ': ' + e)
  } finally {
    replacing.value = false
  }
}

// Git 功能
const loadGitStatus = async () => {
  if (!localWorkDir.value) {
    gitStatus.value = null
    return
  }
  
  loadingGit.value = true
  try {
    const status = await GetGitStatus(localWorkDir.value)
    gitStatus.value = status
    gitBranch.value = status.branch || ''
    gitChanges.value = status.changes || []
  } catch (e) {
    console.error('获取 Git 状态失败:', e)
    gitStatus.value = null
  } finally {
    loadingGit.value = false
  }
}

const gitStageFile = async (path) => {
  try {
    await GitAdd(localWorkDir.value, path)
    await loadGitStatus()
  } catch (e) {
    console.error('暂存文件失败:', e)
  }
}

const gitCommitChanges = async () => {
  const message = prompt(t('git.commitMessage'))
  if (!message) return
  
  try {
    await GitCommit(localWorkDir.value, message)
    await loadGitStatus()
  } catch (e) {
    console.error('提交失败:', e)
    alert(t('git.commitFailed') + ': ' + e)
  }
}

const gitPushChanges = async () => {
  try {
    await GitPush(localWorkDir.value)
    await loadGitStatus()
  } catch (e) {
    console.error('推送失败:', e)
    alert(t('git.pushFailed') + ': ' + e)
  }
}

const gitPullChanges = async () => {
  try {
    await GitPull(localWorkDir.value)
    await loadGitStatus()
  } catch (e) {
    console.error('拉取失败:', e)
    alert(t('git.pullFailed') + ': ' + e)
  }
}

const gitDiscardFile = async (path) => {
  if (!confirm(t('git.discardConfirm'))) return
  
  try {
    await GitDiscard(localWorkDir.value, path)
    await loadGitStatus()
  } catch (e) {
    console.error('丢弃更改失败:', e)
  }
}

const getStatusIcon = (status) => {
  switch (status) {
    case 'M': return 'M'
    case 'A': return 'A'
    case 'D': return 'D'
    case 'R': return 'R'
    case '??': return 'U'
    default: return '?'
  }
}

const getStatusColor = (status) => {
  switch (status) {
    case 'M': return 'var(--yellow)'
    case 'A': return 'var(--green)'
    case 'D': return 'var(--red)'
    case 'R': return 'var(--blue)'
    case '??': return 'var(--text-muted)'
    default: return 'var(--text-secondary)'
  }
}

// 监听 activeTab 变化，加载对应数据
watch(() => props.activeTab, async (newTab) => {
  if (newTab === 'git') {
    await loadGitStatus()
  }
})
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
            @refreshFolder="refreshFolder"
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
    
    <!-- 搜索面板 -->
    <div v-else-if="activeTab === 'search'" class="search-panel">
      <div class="search-input-group">
        <input 
          v-model="searchQuery" 
          type="text" 
          :placeholder="t('search.placeholder')"
          @keyup.enter="performSearch"
          class="search-input"
        />
        <div v-if="showReplace" class="replace-input-wrapper">
          <input 
            v-model="replaceText" 
            type="text" 
            :placeholder="t('search.replacePlaceholder')"
            class="search-input"
          />
        </div>
        <div class="search-options">
          <label class="search-option" :title="t('search.caseSensitive')">
            <input type="checkbox" v-model="searchCaseSensitive" />
            <span>Aa</span>
          </label>
          <label class="search-option" :title="t('search.useRegex')">
            <input type="checkbox" v-model="searchRegex" />
            <span>.*</span>
          </label>
          <button 
            @click="showReplace = !showReplace" 
            class="btn-toggle-replace"
            :class="{ active: showReplace }"
            :title="t('search.toggleReplace')"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
              <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
            </svg>
          </button>
          <button @click="performSearch" class="btn-search" :disabled="!searchQuery || searching">
            <svg v-if="!searching" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
            </svg>
            <span v-else class="spinner"></span>
          </button>
          <button 
            v-if="showReplace" 
            @click="performReplace" 
            class="btn-replace" 
            :disabled="!searchQuery || !searchResults.length || replacing"
          >
            <svg v-if="!replacing" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="20 6 9 17 4 12"/>
            </svg>
            <span v-else class="spinner"></span>
          </button>
        </div>
      </div>
      
      <div v-if="searchResults.length > 0" class="search-results">
        <div class="results-header">{{ searchResults.length }} {{ t('search.results') }}</div>
        <div 
          v-for="(result, idx) in searchResults" 
          :key="idx"
          class="search-result-item"
          @click="openSearchResult(result)"
        >
          <div class="result-path">{{ result.path }}</div>
          <div class="result-line">
            <span class="line-number">{{ result.line }}</span>
            <span class="line-content">{{ result.content }}</span>
          </div>
        </div>
      </div>
      <div v-else-if="!searching && searchQuery" class="empty-results">
        {{ t('search.noResults') }}
      </div>
    </div>
    
    <!-- Git 面板 -->
    <div v-else-if="activeTab === 'git'" class="git-panel">
      <div v-if="!localWorkDir" class="empty-git">
        <p>{{ t('git.noFolder') }}</p>
      </div>
      <div v-else-if="!gitStatus || !gitStatus.hasRepo" class="empty-git">
        <p>{{ t('git.noRepo') }}</p>
      </div>
      <div v-else class="git-content">
        <!-- Git 工具栏 -->
        <div class="git-toolbar">
          <div class="git-branch">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="6" y1="3" x2="6" y2="15"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/>
              <path d="M18 9a9 9 0 0 1-9 9"/>
            </svg>
            <span>{{ gitBranch || 'main' }}</span>
          </div>
          <div class="git-actions">
            <button @click="gitPullChanges" :title="t('git.pull')" class="git-btn">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="8 18 12 22 16 18"/><polyline points="8 6 12 2 16 6"/>
                <line x1="12" y1="2" x2="12" y2="22"/>
              </svg>
            </button>
            <button @click="gitPushChanges" :title="t('git.push')" class="git-btn">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="17 11 12 6 7 11"/><polyline points="17 18 12 13 7 18"/>
                <line x1="12" y1="6" x2="12" y2="18"/>
              </svg>
            </button>
            <button @click="gitCommitChanges" :title="t('git.commit')" class="git-btn" :disabled="gitChanges.length === 0">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="9 11 12 14 22 4"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/>
              </svg>
            </button>
            <button @click="loadGitStatus" :title="t('git.refresh')" class="git-btn">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
              </svg>
            </button>
          </div>
        </div>
        
        <!-- 变更列表 -->
        <div v-if="gitChanges.length > 0" class="git-changes">
          <div class="changes-header">{{ t('git.changes') }} ({{ gitChanges.length }})</div>
          <div 
            v-for="change in gitChanges" 
            :key="change.path"
            class="git-change-item"
          >
            <div class="change-status" :style="{ color: getStatusColor(change.status) }">
              {{ getStatusIcon(change.status) }}
            </div>
            <div class="change-path" @click="emit('openFile', { path: localWorkDir + '/' + change.path, name: change.path.split('/').pop(), type: 'file' })">
              {{ change.path }}
            </div>
            <div class="change-actions">
              <button 
                v-if="!change.staged" 
                @click="gitStageFile(change.path)" 
                :title="t('git.stage')"
                class="change-btn"
              >
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="20 6 9 17 4 12"/>
                </svg>
              </button>
              <button 
                v-if="change.status !== 'A' && change.status !== '??'" 
                @click="gitDiscardFile(change.path)" 
                :title="t('git.discard')"
                class="change-btn"
              >
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/>
                </svg>
              </button>
            </div>
          </div>
        </div>
        <div v-else class="no-changes">
          <p>{{ t('git.noChanges') }}</p>
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

/* 搜索面板样式 */
.search-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.search-input-group {
  padding: 12px;
  border-bottom: 1px solid var(--border-default);
}

.search-input {
  width: 100%;
  padding: 6px 8px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 4px;
  color: var(--text-primary);
  font-size: 13px;
  margin-bottom: 8px;
}

.search-input:focus {
  outline: none;
  border-color: var(--accent-primary);
}

.search-options {
  display: flex;
  align-items: center;
  gap: 8px;
}

.search-option {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--text-secondary);
  cursor: pointer;
}

.search-option input {
  cursor: pointer;
}

.btn-toggle-replace {
  padding: 4px 8px;
  background: transparent;
  border: 1px solid var(--border-default);
  border-radius: 4px;
  color: var(--text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
}

.btn-toggle-replace:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.btn-toggle-replace.active {
  background: var(--accent-primary);
  border-color: var(--accent-primary);
  color: white;
}

.replace-input-wrapper {
  margin-top: 8px;
}

.btn-search {
  margin-left: auto;
  padding: 4px 12px;
  background: var(--accent-primary);
  border: none;
  border-radius: 4px;
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
}

.btn-search:hover:not(:disabled) {
  background: var(--accent-hover);
}

.btn-search:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-replace {
  padding: 4px 12px;
  background: var(--green);
  border: none;
  border-radius: 4px;
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
}

.btn-replace:hover:not(:disabled) {
  opacity: 0.9;
}

.btn-replace:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.search-results {
  flex: 1;
  overflow-y: auto;
}

.results-header {
  padding: 8px 12px;
  font-size: 11px;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border-default);
}

.search-result-item {
  padding: 8px 12px;
  cursor: pointer;
  border-bottom: 1px solid var(--border-subtle);
}

.search-result-item:hover {
  background: var(--bg-hover);
}

.result-path {
  font-size: 11px;
  color: var(--text-muted);
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.result-line {
  display: flex;
  gap: 8px;
  font-size: 12px;
}

.line-number {
  color: var(--text-muted);
  min-width: 30px;
}

.line-content {
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-results {
  padding: 40px 20px;
  text-align: center;
  color: var(--text-muted);
  font-size: 13px;
}

/* Git 面板样式 */
.git-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.empty-git {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  text-align: center;
  color: var(--text-muted);
  font-size: 13px;
}

.git-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.git-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-default);
}

.git-branch {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-primary);
}

.git-actions {
  display: flex;
  gap: 4px;
}

.git-btn {
  padding: 4px;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
}

.git-btn:hover:not(:disabled) {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.git-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.git-changes {
  flex: 1;
  overflow-y: auto;
}

.changes-header {
  padding: 8px 12px;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border-default);
}

.git-change-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  border-bottom: 1px solid var(--border-subtle);
}

.git-change-item:hover {
  background: var(--bg-hover);
}

.change-status {
  font-size: 11px;
  font-weight: 700;
  width: 16px;
  text-align: center;
}

.change-path {
  flex: 1;
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.change-path:hover {
  color: var(--text-primary);
}

.change-actions {
  display: flex;
  gap: 4px;
}

.change-btn {
  padding: 2px;
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  border-radius: 3px;
}

.change-btn:hover {
  background: var(--bg-active);
  color: var(--text-primary);
}

.no-changes {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  text-align: center;
  color: var(--text-muted);
  font-size: 13px;
}
</style>
