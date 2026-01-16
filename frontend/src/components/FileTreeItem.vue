<script setup>
import { ref, computed, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { DeletePath, RenamePath, OpenInFinder, CopyToClipboard, CreateNewFile, CreateNewFolder, CopyPath, MovePath } from '../../wailsjs/go/main/App'

const { t } = useI18n()

const props = defineProps({
  item: Object,
  depth: { type: Number, default: 0 },
  expandedFolders: Set
})

const emit = defineEmits(['openFile', 'toggleFolder', 'refresh'])

// 右键菜单状态
const showContextMenu = ref(false)
const contextMenuX = ref(0)
const contextMenuY = ref(0)

// 重命名状态
const isRenaming = ref(false)
const renameValue = ref('')

// 新建文件/文件夹状态
const isCreating = ref(false)
const createType = ref('')
const createValue = ref('')

// 文件图标配置
const fileIcons = {
  'go': { color: '#00ADD8' }, 'js': { color: '#F7DF1E' }, 'jsx': { color: '#61DAFB' },
  'ts': { color: '#3178C6' }, 'tsx': { color: '#61DAFB' }, 'vue': { color: '#42B883' },
  'py': { color: '#3776AB' }, 'java': { color: '#E76F00' }, 'kt': { color: '#7F52FF' },
  'swift': { color: '#FA7343' }, 'rs': { color: '#DEA584' }, 'rb': { color: '#CC342D' },
  'php': { color: '#777BB4' }, 'c': { color: '#A8B9CC' }, 'cpp': { color: '#00599C' },
  'h': { color: '#A8B9CC' }, 'cs': { color: '#239120' }, 'lua': { color: '#000080' },
  'sh': { color: '#4EAA25' }, 'html': { color: '#E34F26' }, 'css': { color: '#1572B6' },
  'scss': { color: '#CC6699' }, 'json': { color: '#F5A623' }, 'yaml': { color: '#CB171E' },
  'yml': { color: '#CB171E' }, 'xml': { color: '#E34F26' }, 'md': { color: '#083FA1' },
  'txt': { color: '#6B6B6B' }, 'png': { color: '#89CFF0' }, 'jpg': { color: '#89CFF0' },
  'svg': { color: '#FFB13B' }, 'sql': { color: '#336791' }, 'lock': { color: '#6B6B6B' },
}

const getFileIcon = (name) => {
  const ext = name.toLowerCase().split('.').pop()
  return fileIcons[ext] || { color: '#6B6B6B' }
}

const getFileIconText = (name) => {
  const lowerName = name.toLowerCase()
  const ext = lowerName.split('.').pop()
  const baseName = name.replace(/\.[^.]+$/, '')
  
  if (lowerName === 'dockerfile') return '🐳'
  if (lowerName === 'makefile') return '⚙️'
  if (lowerName.includes('license')) return '📜'
  if (lowerName.includes('readme')) return 'ℹ️'
  if (lowerName === '.gitignore') return '◆'
  if (lowerName === '.env' || lowerName.startsWith('.env.')) return '⚡'
  if (ext === 'go' && baseName.endsWith('_test')) return 'T'
  if (ext === 'py') {
    if (baseName === '__init__') return 'P'
    if (baseName.startsWith('test_') || baseName.endsWith('_test')) return 'T'
  }
  
  const iconMap = {
    'go': 'Go', 'js': 'JS', 'jsx': '⚛', 'ts': 'TS', 'tsx': '⚛', 'vue': 'V',
    'py': '🐍', 'java': '☕', 'kt': 'K', 'swift': '🦅', 'rs': '🦀', 'rb': '💎',
    'php': '🐘', 'c': 'C', 'cpp': 'C+', 'h': 'H', 'cs': 'C#', 'lua': '🌙',
    'sh': '$', 'bash': '$', 'zsh': '$', 'ps1': '>_',
    'html': '<>', 'htm': '<>', 'css': '#', 'scss': 'S#', 'sass': 'S#', 'less': 'L#',
    'json': '{}', 'yaml': 'Y', 'yml': 'Y', 'xml': 'X', 'toml': 'T', 'ini': '⚙',
    'md': 'M', 'mdx': 'M', 'txt': 'T', 'pdf': 'P', 'doc': 'W', 'docx': 'W',
    'png': '🖼', 'jpg': '🖼', 'jpeg': '🖼', 'gif': '🖼', 'svg': 'S', 'ico': '🖼', 'webp': '🖼',
    'ttf': 'F', 'otf': 'F', 'woff': 'F', 'woff2': 'F',
    'zip': '📦', 'tar': '📦', 'gz': '📦', 'rar': '📦', '7z': '📦',
    'sql': 'Q', 'db': 'D', 'sqlite': 'D',
    'lock': '🔒', 'sum': '∑', 'mod': 'Go',
  }
  return iconMap[ext] || '?'
}

const isImageFile = (name) => {
  const ext = name.split('.').pop()?.toLowerCase()
  return ['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'ico', 'bmp'].includes(ext)
}

const isBinaryFile = (name) => {
  const ext = name.split('.').pop()?.toLowerCase()
  const binaryExts = ['exe', 'dll', 'so', 'dylib', 'bin', 'dat', 'zip', 'tar', 'gz', 'rar', '7z',
    'pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'mp3', 'mp4', 'avi', 'mov', 'mkv',
    'ttf', 'otf', 'woff', 'woff2', 'db', 'sqlite', 'class', 'jar', 'war', 'o', 'a', 'pyc']
  return binaryExts.includes(ext)
}

const isExpanded = () => props.expandedFolders.has(props.item.path)
const paddingLeft = () => 16 + props.depth * 16 + 'px'
const hasClipboard = computed(() => !!window.__fileClipboard?.path)

const handleClick = () => {
  if (props.item.type === 'folder') {
    emit('toggleFolder', props.item)
  }
}

const handleDblClick = () => {
  if (props.item.type === 'file') {
    emit('openFile', props.item)
  }
}

// 右键菜单
const handleContextMenu = (e) => {
  e.preventDefault()
  e.stopPropagation()
  contextMenuX.value = e.clientX
  contextMenuY.value = e.clientY
  showContextMenu.value = true
  setTimeout(() => {
    document.addEventListener('click', closeContextMenu, { once: true })
  }, 0)
}

const closeContextMenu = () => {
  showContextMenu.value = false
}

// 菜单操作
const doOpenInFinder = async () => {
  closeContextMenu()
  try {
    await OpenInFinder(props.item.path)
  } catch (e) {
    console.error('打开失败:', e)
  }
}

const doCopyPath = async () => {
  closeContextMenu()
  await CopyToClipboard(props.item.path)
}

const doCopy = () => {
  closeContextMenu()
  window.__fileClipboard = { path: props.item.path, action: 'copy' }
}

const doCut = () => {
  closeContextMenu()
  window.__fileClipboard = { path: props.item.path, action: 'cut' }
}

const doPaste = async () => {
  closeContextMenu()
  const cb = window.__fileClipboard
  if (!cb?.path) return
  try {
    if (cb.action === 'copy') {
      await CopyPath(cb.path, props.item.path)
    } else {
      await MovePath(cb.path, props.item.path)
      window.__fileClipboard = null
    }
    emit('refresh')
  } catch (e) {
    console.error('粘贴失败:', e)
  }
}

const startRename = () => {
  closeContextMenu()
  renameValue.value = props.item.name
  isRenaming.value = true
  nextTick(() => {
    const input = document.querySelector('.rename-input')
    if (input) {
      input.focus()
      input.select()
    }
  })
}

const confirmRename = async () => {
  if (!renameValue.value || renameValue.value === props.item.name) {
    isRenaming.value = false
    return
  }
  try {
    await RenamePath(props.item.path, renameValue.value)
    emit('refresh')
  } catch (e) {
    console.error('重命名失败:', e)
  }
  isRenaming.value = false
}

const cancelRename = () => {
  isRenaming.value = false
}

const doDelete = async () => {
  closeContextMenu()
  try {
    await DeletePath(props.item.path)
    emit('refresh')
  } catch (e) {
    console.error('删除失败:', e)
  }
}

const startNewFile = () => {
  closeContextMenu()
  if (props.item.type !== 'folder') return
  createType.value = 'file'
  createValue.value = ''
  isCreating.value = true
  if (!props.expandedFolders.has(props.item.path)) {
    emit('toggleFolder', props.item)
  }
  nextTick(() => {
    const input = document.querySelector('.create-input')
    if (input) input.focus()
  })
}

const startNewFolder = () => {
  closeContextMenu()
  if (props.item.type !== 'folder') return
  createType.value = 'folder'
  createValue.value = ''
  isCreating.value = true
  if (!props.expandedFolders.has(props.item.path)) {
    emit('toggleFolder', props.item)
  }
  nextTick(() => {
    const input = document.querySelector('.create-input')
    if (input) input.focus()
  })
}

const confirmCreate = async () => {
  if (!createValue.value) {
    isCreating.value = false
    return
  }
  try {
    if (createType.value === 'file') {
      await CreateNewFile(props.item.path, createValue.value)
    } else {
      await CreateNewFolder(props.item.path, createValue.value)
    }
    emit('refresh')
  } catch (e) {
    console.error('创建失败:', e)
  }
  isCreating.value = false
}

const cancelCreate = () => {
  isCreating.value = false
}
</script>

<template>
  <div>
    <!-- 正常显示 -->
    <div 
      v-if="!isRenaming"
      :class="['tree-item', { folder: item.type === 'folder' }]" 
      :style="{ paddingLeft: paddingLeft() }"
      @click="handleClick"
      @dblclick="handleDblClick"
      @contextmenu="handleContextMenu"
    >
      <svg v-if="item.type === 'folder'" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" :class="{ rotated: isExpanded() }">
        <path d="M9 18l6-6-6-6"/>
      </svg>
      <span v-else class="spacer"></span>
      
      <svg v-if="item.type === 'folder'" width="16" height="16" viewBox="0 0 24 24" fill="#6b9eff" stroke="#5a8af0" stroke-width="0.5">
        <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
      </svg>
      <span v-else class="file-icon-circle" :style="{ borderColor: getFileIcon(item.name).color, color: getFileIcon(item.name).color }">
        {{ getFileIconText(item.name) }}
      </span>
      
      <span class="name">{{ item.name }}</span>
      <span v-if="item.type === 'file' && isImageFile(item.name)" class="type-badge image">IMG</span>
      <span v-else-if="item.type === 'file' && isBinaryFile(item.name)" class="type-badge binary">BIN</span>
    </div>
    
    <!-- 重命名输入框 -->
    <div v-else class="tree-item rename-item" :style="{ paddingLeft: paddingLeft() }">
      <span class="spacer"></span>
      <input 
        class="rename-input"
        v-model="renameValue"
        @keyup.enter="confirmRename"
        @keyup.escape="cancelRename"
        @blur="confirmRename"
        autocomplete="off"
        spellcheck="false"
      />
    </div>
    
    <!-- 右键菜单 -->
    <Teleport to="body">
      <div 
        v-if="showContextMenu" 
        class="context-menu"
        :style="{ left: contextMenuX + 'px', top: contextMenuY + 'px' }"
      >
        <!-- 文件夹专属 -->
        <template v-if="item.type === 'folder'">
          <div class="menu-item" @click="startNewFile">{{ t('contextMenu.newFile') }}</div>
          <div class="menu-item" @click="startNewFolder">{{ t('contextMenu.newFolder') }}</div>
          <div class="menu-divider"></div>
        </template>
        
        <div class="menu-item" @click="doCopy">{{ t('contextMenu.copy') }}</div>
        <div class="menu-item" @click="doCut">{{ t('contextMenu.cut') }}</div>
        <div v-if="item.type === 'folder' && hasClipboard" class="menu-item" @click="doPaste">{{ t('contextMenu.paste') }}</div>
        <div class="menu-divider"></div>
        <div class="menu-item" @click="startRename">{{ t('contextMenu.rename') }}</div>
        <div class="menu-item danger" @click="doDelete">{{ t('contextMenu.delete') }}</div>
        <div class="menu-divider"></div>
        <div class="menu-item" @click="doCopyPath">{{ t('contextMenu.copyPath') }}</div>
        <div class="menu-item" @click="doOpenInFinder">{{ t('contextMenu.openInFinder') }}</div>
      </div>
    </Teleport>
    
    <!-- 递归渲染子项 -->
    <template v-if="item.type === 'folder' && isExpanded() && item.children">
      <!-- 新建文件/文件夹输入框 -->
      <div v-if="isCreating" class="tree-item create-item" :style="{ paddingLeft: (16 + (depth + 1) * 16) + 'px' }">
        <span class="create-icon">{{ createType === 'file' ? '📄' : '📁' }}</span>
        <input 
          class="create-input"
          v-model="createValue"
          :placeholder="createType === 'file' ? t('contextMenu.newFileName') : t('contextMenu.newFolderName')"
          @keyup.enter="confirmCreate"
          @keyup.escape="cancelCreate"
          @blur="confirmCreate"
          autocomplete="off"
          spellcheck="false"
        />
      </div>
      
      <FileTreeItem
        v-for="child in item.children"
        :key="child.path"
        :item="child"
        :depth="depth + 1"
        :expandedFolders="expandedFolders"
        @openFile="emit('openFile', $event)"
        @toggleFolder="emit('toggleFolder', $event)"
        @refresh="emit('refresh')"
      />
    </template>
  </div>
</template>

<style scoped>
.tree-item { display: flex; align-items: center; gap: 4px; padding: 2px 8px; cursor: pointer; user-select: none; }
.tree-item:hover { background: var(--bg-hover); }
.tree-item svg.rotated { transform: rotate(90deg); }
.spacer { width: 12px; }
.file-icon-circle { 
  display: inline-flex; align-items: center; justify-content: center;
  width: 16px; height: 16px; font-size: 10px; font-weight: 600; 
  border: 1.5px solid; border-radius: 50%; 
  font-family: system-ui, -apple-system, sans-serif; flex-shrink: 0;
}
.name { font-size: 13px; color: var(--text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; flex: 1; }
.type-badge { font-size: 9px; padding: 1px 4px; border-radius: 3px; font-weight: 600; margin-left: auto; }
.type-badge.image { background: rgba(137, 207, 240, 0.2); color: #89CFF0; }
.type-badge.binary { background: rgba(255, 107, 107, 0.2); color: #FF6B6B; }

.context-menu {
  position: fixed; background: var(--bg-elevated); border: 1px solid var(--border-default);
  border-radius: 6px; padding: 4px 0; min-width: 160px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3); z-index: 10000;
}
.menu-item { padding: 6px 12px; font-size: 12px; color: var(--text-primary); cursor: pointer; }
.menu-item:hover { background: var(--bg-hover); }
.menu-item.danger { color: #ff6b6b; }
.menu-item.danger:hover { background: rgba(255, 107, 107, 0.1); }
.menu-divider { height: 1px; background: var(--border-default); margin: 4px 0; }

.rename-item, .create-item { background: var(--bg-hover); }
.rename-input, .create-input {
  flex: 1; background: var(--bg-surface); border: 1px solid var(--accent-primary);
  border-radius: 3px; padding: 2px 6px; font-size: 13px; color: var(--text-primary); outline: none;
}
.create-icon { font-size: 14px; margin-right: 4px; }
</style>
