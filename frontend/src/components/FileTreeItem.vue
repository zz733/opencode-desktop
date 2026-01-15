<script setup>
import { ref } from 'vue'
import { ListDir } from '../../wailsjs/go/main/App'

const props = defineProps({
  item: Object,
  depth: { type: Number, default: 0 },
  expandedFolders: Set
})

const emit = defineEmits(['openFile', 'toggleFolder'])

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
  
  // 特殊文件名
  if (lowerName === 'dockerfile') return '🐳'
  if (lowerName === 'makefile') return '⚙️'
  if (lowerName.includes('license')) return '📜'
  if (lowerName.includes('readme')) return 'ℹ️'
  if (lowerName === '.gitignore') return '◆'
  if (lowerName === '.env' || lowerName.startsWith('.env.')) return '⚡'
  
  // Go 测试文件
  if (ext === 'go' && baseName.endsWith('_test')) return 'T'
  
  // Python 测试文件
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

const handleClick = async () => {
  if (props.item.type === 'folder') {
    emit('toggleFolder', props.item)
  }
}

const handleDblClick = () => {
  if (props.item.type === 'file') {
    emit('openFile', props.item)
  }
}

const paddingLeft = () => 16 + props.depth * 16 + 'px'
</script>

<template>
  <div>
    <div 
      :class="['tree-item', { folder: item.type === 'folder' }]" 
      :style="{ paddingLeft: paddingLeft() }"
      @click="handleClick"
      @dblclick="handleDblClick"
    >
      <svg v-if="item.type === 'folder'" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" :class="{ rotated: isExpanded() }">
        <path d="M9 18l6-6-6-6"/>
      </svg>
      <span v-else class="spacer"></span>
      
      <!-- 文件夹图标 - 蓝色 -->
      <svg v-if="item.type === 'folder'" width="16" height="16" viewBox="0 0 24 24" fill="#6b9eff" stroke="#5a8af0" stroke-width="0.5">
        <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
      </svg>
      <!-- 文件图标 - 圆圈字母 -->
      <span v-else class="file-icon-circle" :style="{ borderColor: getFileIcon(item.name).color, color: getFileIcon(item.name).color }">
        {{ getFileIconText(item.name) }}
      </span>
      
      <span class="name">{{ item.name }}</span>
      <span v-if="item.type === 'file' && isImageFile(item.name)" class="type-badge image">IMG</span>
      <span v-else-if="item.type === 'file' && isBinaryFile(item.name)" class="type-badge binary">BIN</span>
    </div>
    
    <!-- 递归渲染子项 -->
    <template v-if="item.type === 'folder' && isExpanded() && item.children">
      <FileTreeItem
        v-for="child in item.children"
        :key="child.path"
        :item="child"
        :depth="depth + 1"
        :expandedFolders="expandedFolders"
        @openFile="emit('openFile', $event)"
        @toggleFolder="emit('toggleFolder', $event)"
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
  display: inline-flex; 
  align-items: center; 
  justify-content: center;
  width: 16px; 
  height: 16px; 
  font-size: 10px; 
  font-weight: 600; 
  border: 1.5px solid; 
  border-radius: 50%; 
  font-family: system-ui, -apple-system, sans-serif;
  flex-shrink: 0;
}
.name { font-size: 13px; color: var(--text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; flex: 1; }
.type-badge { font-size: 9px; padding: 1px 4px; border-radius: 3px; font-weight: 600; margin-left: auto; }
.type-badge.image { background: rgba(137, 207, 240, 0.2); color: #89CFF0; }
.type-badge.binary { background: rgba(255, 107, 107, 0.2); color: #FF6B6B; }
</style>
