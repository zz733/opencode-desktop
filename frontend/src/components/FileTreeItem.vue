<script setup>
import { ref, computed, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { DeletePath, RenamePath, OpenInFinder, CopyToClipboard, CreateNewFile, CreateNewFolder, CopyPath, MovePath, WriteFileContent } from '../../wailsjs/go/main/App'

const { t } = useI18n()

const props = defineProps({
  item: Object,
  depth: { type: Number, default: 0 },
  expandedFolders: Set,
  projectType: { type: String, default: '' }
})

const emit = defineEmits(['openFile', 'toggleFolder', 'refresh', 'refreshFolder'])

const showContextMenu = ref(false)
const contextMenuX = ref(0)
const contextMenuY = ref(0)
const showNewSubmenu = ref(false)
const isRenaming = ref(false)
const renameValue = ref('')
const isCreating = ref(false)
const createType = ref('')
const createValue = ref('')

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
  if (ext === 'py' && baseName === '__init__') return 'P'
  if (ext === 'py' && (baseName.startsWith('test_') || baseName.endsWith('_test'))) return 'T'
  
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
    'sql': 'Q', 'db': 'D', 'sqlite': 'D', 'lock': '🔒', 'sum': '∑', 'mod': 'Go',
  }
  return iconMap[ext] || '?'
}

const isImageFile = (name) => ['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'ico', 'bmp'].includes(name.split('.').pop()?.toLowerCase())
const isBinaryFile = (name) => ['exe', 'dll', 'so', 'dylib', 'bin', 'dat', 'zip', 'tar', 'gz', 'rar', '7z', 'pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'mp3', 'mp4', 'avi', 'mov', 'mkv', 'ttf', 'otf', 'woff', 'woff2', 'db', 'sqlite', 'class', 'jar', 'war', 'o', 'a', 'pyc'].includes(name.split('.').pop()?.toLowerCase())
const isExpanded = () => props.expandedFolders.has(props.item.path)
const paddingLeft = () => 16 + props.depth * 16 + 'px'
const hasClipboard = computed(() => !!window.__fileClipboard?.path)

const newMenuItems = computed(() => {
  const items = [
    { id: 'file', label: t('contextMenu.newFile'), icon: '📄' },
    { id: 'folder', label: t('contextMenu.newFolder'), icon: '📁' },
  ]
  const pt = props.projectType
  if (pt === 'java' || pt === 'maven' || pt === 'gradle') {
    items.push({ divider: true })
    items.push({ id: 'java-class', label: t('contextMenu.javaClass'), icon: '☕' })
    items.push({ id: 'java-interface', label: t('contextMenu.javaInterface'), icon: 'I' })
    items.push({ id: 'java-enum', label: t('contextMenu.javaEnum'), icon: 'E' })
    items.push({ id: 'java-record', label: t('contextMenu.javaRecord'), icon: 'R' })
    items.push({ id: 'java-annotation', label: t('contextMenu.javaAnnotation'), icon: '@' })
    items.push({ id: 'java-package', label: t('contextMenu.javaPackage'), icon: '📦' })
  }
  if (pt === 'go') {
    items.push({ divider: true })
    items.push({ id: 'go-file', label: t('contextMenu.goFile'), icon: 'Go' })
    items.push({ id: 'go-test', label: t('contextMenu.goTest'), icon: 'T' })
  }
  if (pt === 'python') {
    items.push({ divider: true })
    items.push({ id: 'py-file', label: t('contextMenu.pyFile'), icon: '🐍' })
    items.push({ id: 'py-package', label: t('contextMenu.pyPackage'), icon: '📦' })
    items.push({ id: 'py-test', label: t('contextMenu.pyTest'), icon: 'T' })
  }
  if (pt === 'node' || pt === 'vue' || pt === 'react') {
    items.push({ divider: true })
    if (pt === 'vue') {
      items.push({ id: 'vue-component', label: t('contextMenu.vueComponent'), icon: 'V' })
      items.push({ id: 'vue-composable', label: t('contextMenu.vueComposable'), icon: '⚡' })
    }
    if (pt === 'react') {
      items.push({ id: 'react-component', label: t('contextMenu.reactComponent'), icon: '⚛' })
      items.push({ id: 'react-hook', label: t('contextMenu.reactHook'), icon: '🪝' })
    }
    items.push({ id: 'ts-file', label: t('contextMenu.tsFile'), icon: 'TS' })
    items.push({ id: 'js-file', label: t('contextMenu.jsFile'), icon: 'JS' })
  }
  if (pt === 'rust') {
    items.push({ divider: true })
    items.push({ id: 'rs-file', label: t('contextMenu.rsFile'), icon: '🦀' })
    items.push({ id: 'rs-mod', label: t('contextMenu.rsMod'), icon: '📦' })
  }
  if (pt === 'csharp' || pt === 'dotnet') {
    items.push({ divider: true })
    items.push({ id: 'cs-class', label: t('contextMenu.csClass'), icon: 'C#' })
    items.push({ id: 'cs-interface', label: t('contextMenu.csInterface'), icon: 'I' })
    items.push({ id: 'cs-enum', label: t('contextMenu.csEnum'), icon: 'E' })
  }
  return items
})

const getFileTemplate = (type, name) => {
  const templates = {
    'java-class': `public class ${name} {\n    \n}\n`,
    'java-interface': `public interface ${name} {\n    \n}\n`,
    'java-enum': `public enum ${name} {\n    \n}\n`,
    'java-record': `public record ${name}() {\n    \n}\n`,
    'java-annotation': `import java.lang.annotation.*;\n\n@Retention(RetentionPolicy.RUNTIME)\n@Target(ElementType.TYPE)\npublic @interface ${name} {\n    \n}\n`,
    'go-file': `package main\n\n`,
    'go-test': `package main\n\nimport "testing"\n\nfunc Test${name}(t *testing.T) {\n    \n}\n`,
    'py-file': `#!/usr/bin/env python3\n# -*- coding: utf-8 -*-\n\n`,
    'py-test': `import unittest\n\nclass Test${name}(unittest.TestCase):\n    def test_example(self):\n        pass\n\nif __name__ == '__main__':\n    unittest.main()\n`,
    'vue-component': `<script setup>\n\n<\/script>\n\n<template>\n  <div>\n    \n  </div>\n</template>\n\n<style scoped>\n\n</style>\n`,
    'vue-composable': `import { ref } from 'vue'\n\nexport function use${name}() {\n  const state = ref(null)\n  return { state }\n}\n`,
    'react-component': `import React from 'react'\n\nexport function ${name}() {\n  return <div></div>\n}\n`,
    'react-hook': `import { useState } from 'react'\n\nexport function use${name}() {\n  const [state, setState] = useState(null)\n  return { state, setState }\n}\n`,
    'rs-file': `\n`,
    'rs-mod': `pub mod ${name};\n`,
    'cs-class': `namespace MyNamespace\n{\n    public class ${name}\n    {\n        \n    }\n}\n`,
    'cs-interface': `namespace MyNamespace\n{\n    public interface I${name}\n    {\n        \n    }\n}\n`,
    'cs-enum': `namespace MyNamespace\n{\n    public enum ${name}\n    {\n        \n    }\n}\n`,
  }
  return templates[type] || ''
}

const getFileExtension = (type) => {
  const ext = { 'java-class': '.java', 'java-interface': '.java', 'java-enum': '.java', 'java-record': '.java', 'java-annotation': '.java', 'go-file': '.go', 'go-test': '_test.go', 'py-file': '.py', 'py-test': '_test.py', 'vue-component': '.vue', 'vue-composable': '.js', 'react-component': '.jsx', 'react-hook': '.js', 'ts-file': '.ts', 'js-file': '.js', 'rs-file': '.rs', 'rs-mod': '.rs', 'cs-class': '.cs', 'cs-interface': '.cs', 'cs-enum': '.cs' }
  return ext[type] || ''
}

const handleClick = () => { if (props.item.type === 'folder') emit('toggleFolder', props.item) }
const handleDblClick = () => { if (props.item.type === 'file') emit('openFile', props.item) }

const handleContextMenu = (e) => {
  e.preventDefault()
  e.stopPropagation()
  contextMenuX.value = e.clientX
  contextMenuY.value = e.clientY
  showContextMenu.value = true
  showNewSubmenu.value = false
  setTimeout(() => document.addEventListener('click', closeContextMenu, { once: true }), 0)
}

const closeContextMenu = () => { showContextMenu.value = false; showNewSubmenu.value = false }

const doOpenInFinder = async () => { closeContextMenu(); try { await OpenInFinder(props.item.path) } catch (e) { console.error(e) } }
const doCopyPath = async () => { closeContextMenu(); await CopyToClipboard(props.item.path) }
const doCopy = () => { closeContextMenu(); window.__fileClipboard = { path: props.item.path, action: 'copy' } }
const doCut = () => { closeContextMenu(); window.__fileClipboard = { path: props.item.path, action: 'cut' } }
const doPaste = async () => {
  closeContextMenu()
  const cb = window.__fileClipboard
  if (!cb?.path) return
  try {
    if (cb.action === 'copy') await CopyPath(cb.path, props.item.path)
    else { await MovePath(cb.path, props.item.path); window.__fileClipboard = null }
    emit('refreshFolder', props.item.path)
  } catch (e) { console.error(e) }
}

const startRename = () => {
  closeContextMenu()
  renameValue.value = props.item.name
  isRenaming.value = true
  nextTick(() => { const input = document.querySelector('.rename-input'); if (input) { input.focus(); input.select() } })
}
// 获取父文件夹路径
const getParentPath = (path) => {
  const parts = path.split('/')
  parts.pop()
  return parts.join('/') || '/'
}

const confirmRename = async () => {
  if (!renameValue.value || renameValue.value === props.item.name) { isRenaming.value = false; return }
  try { 
    await RenamePath(props.item.path, renameValue.value)
    emit('refreshFolder', getParentPath(props.item.path))
  } catch (e) { console.error(e) }
  isRenaming.value = false
}
const cancelRename = () => { isRenaming.value = false }
const doDelete = async () => { 
  closeContextMenu()
  try { 
    await DeletePath(props.item.path)
    emit('refreshFolder', getParentPath(props.item.path))
  } catch (e) { console.error(e) } 
}

const startCreate = (type) => {
  closeContextMenu()
  if (props.item.type !== 'folder') return
  createType.value = type
  createValue.value = ''
  isCreating.value = true
  if (!props.expandedFolders.has(props.item.path)) emit('toggleFolder', props.item)
  nextTick(() => { const input = document.querySelector('.create-input'); if (input) input.focus() })
}

const confirmCreate = async () => {
  if (!createValue.value) { isCreating.value = false; return }
  try {
    let fileName = createValue.value
    const type = createType.value
    if (type === 'folder' || type === 'java-package' || type === 'py-package') {
      if (type === 'java-package') fileName = fileName.replace(/\./g, '/')
      await CreateNewFolder(props.item.path, fileName)
      if (type === 'py-package') await CreateNewFile(props.item.path + '/' + fileName, '__init__.py')
    } else if (type === 'file') {
      await CreateNewFile(props.item.path, fileName)
    } else {
      const ext = getFileExtension(type)
      if (!fileName.includes('.')) fileName = fileName + ext
      const filePath = props.item.path + '/' + fileName
      const baseName = fileName.replace(/\.[^.]+$/, '')
      const template = getFileTemplate(type, baseName)
      await CreateNewFile(props.item.path, fileName)
      if (template) await WriteFileContent(filePath, template)
    }
    // 只刷新当前文件夹，不刷新整个树
    emit('refreshFolder', props.item.path)
  } catch (e) { console.error(e) }
  isCreating.value = false
}
const cancelCreate = () => { isCreating.value = false }

const getCreatePlaceholder = computed(() => {
  const type = createType.value
  if (type === 'folder') return t('contextMenu.newFolderName')
  if (type === 'file') return t('contextMenu.newFileName')
  if (type === 'java-package') return 'com.example.package'
  if (type === 'py-package') return 'package_name'
  if (type.startsWith('java-') || type.startsWith('vue-') || type.startsWith('react-')) return 'ClassName'
  return t('contextMenu.newFileName')
})

const getCreateIcon = computed(() => {
  const item = newMenuItems.value.find(i => i.id === createType.value)
  return item?.icon || '📄'
})
</script>

<template>
  <div>
    <div v-if="!isRenaming" :class="['tree-item', { folder: item.type === 'folder' }]" :style="{ paddingLeft: paddingLeft() }" @click="handleClick" @dblclick="handleDblClick" @contextmenu="handleContextMenu">
      <svg v-if="item.type === 'folder'" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" :class="{ rotated: isExpanded() }"><path d="M9 18l6-6-6-6"/></svg>
      <span v-else class="spacer"></span>
      <svg v-if="item.type === 'folder'" width="16" height="16" viewBox="0 0 24 24" fill="#6b9eff" stroke="#5a8af0" stroke-width="0.5"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
      <span v-else class="file-icon-circle" :style="{ borderColor: getFileIcon(item.name).color, color: getFileIcon(item.name).color }">{{ getFileIconText(item.name) }}</span>
      <span class="name">{{ item.name }}</span>
      <span v-if="item.type === 'file' && isImageFile(item.name)" class="type-badge image">IMG</span>
      <span v-else-if="item.type === 'file' && isBinaryFile(item.name)" class="type-badge binary">BIN</span>
    </div>
    <div v-else class="tree-item rename-item" :style="{ paddingLeft: paddingLeft() }">
      <span class="spacer"></span>
      <input class="rename-input" v-model="renameValue" @keyup.enter="confirmRename" @keyup.escape="cancelRename" @blur="confirmRename" autocomplete="off" spellcheck="false" />
    </div>
    
    <Teleport to="body">
      <div v-if="showContextMenu" class="context-menu" :style="{ left: contextMenuX + 'px', top: contextMenuY + 'px' }">
        <template v-if="item.type === 'folder'">
          <div class="menu-item has-submenu" @mouseenter="showNewSubmenu = true" @mouseleave="showNewSubmenu = false">
            <span>{{ t('contextMenu.new') }}</span>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 18l6-6-6-6"/></svg>
            <div v-if="showNewSubmenu" class="submenu">
              <template v-for="mi in newMenuItems" :key="mi.id || 'div'">
                <div v-if="mi.divider" class="menu-divider"></div>
                <div v-else class="menu-item" @click="startCreate(mi.id)"><span class="menu-icon">{{ mi.icon }}</span><span>{{ mi.label }}</span></div>
              </template>
            </div>
          </div>
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

    <template v-if="item.type === 'folder' && isExpanded() && item.children">
      <div v-if="isCreating" class="tree-item create-item" :style="{ paddingLeft: (16 + (depth + 1) * 16) + 'px' }">
        <span class="create-icon">{{ getCreateIcon }}</span>
        <input class="create-input" v-model="createValue" :placeholder="getCreatePlaceholder" @keyup.enter="confirmCreate" @keyup.escape="cancelCreate" @blur="confirmCreate" autocomplete="off" spellcheck="false" />
      </div>
      <FileTreeItem v-for="child in item.children" :key="child.path" :item="child" :depth="depth + 1" :expandedFolders="expandedFolders" :projectType="projectType" @openFile="emit('openFile', $event)" @toggleFolder="emit('toggleFolder', $event)" @refresh="emit('refresh')" @refreshFolder="emit('refreshFolder', $event)" />
    </template>
  </div>
</template>

<style scoped>
.tree-item { display: flex; align-items: center; gap: 4px; padding: 2px 8px; cursor: pointer; user-select: none; }
.tree-item:hover { background: var(--bg-hover); }
.tree-item svg.rotated { transform: rotate(90deg); }
.spacer { width: 12px; }
.file-icon-circle { display: inline-flex; align-items: center; justify-content: center; width: 16px; height: 16px; font-size: 10px; font-weight: 600; border: 1.5px solid; border-radius: 50%; font-family: system-ui, -apple-system, sans-serif; flex-shrink: 0; }
.name { font-size: 13px; color: var(--text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; flex: 1; }
.type-badge { font-size: 9px; padding: 1px 4px; border-radius: 3px; font-weight: 600; margin-left: auto; }
.type-badge.image { background: rgba(137, 207, 240, 0.2); color: #89CFF0; }
.type-badge.binary { background: rgba(255, 107, 107, 0.2); color: #FF6B6B; }
.context-menu { position: fixed; background: var(--bg-elevated); border: 1px solid var(--border-default); border-radius: 6px; padding: 4px 0; min-width: 180px; box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3); z-index: 10000; }
.menu-item { display: flex; align-items: center; gap: 8px; padding: 6px 12px; font-size: 12px; color: var(--text-primary); cursor: pointer; position: relative; }
.menu-item:hover { background: var(--bg-hover); }
.menu-item.danger { color: #ff6b6b; }
.menu-item.danger:hover { background: rgba(255, 107, 107, 0.1); }
.menu-item.has-submenu { justify-content: space-between; }
.menu-icon { width: 16px; text-align: center; font-size: 12px; }
.menu-divider { height: 1px; background: var(--border-default); margin: 4px 0; }
.submenu { position: absolute; left: 100%; top: -4px; background: var(--bg-elevated); border: 1px solid var(--border-default); border-radius: 6px; padding: 4px 0; min-width: 180px; box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3); }
.rename-item, .create-item { background: var(--bg-hover); }
.rename-input, .create-input { flex: 1; background: var(--bg-surface); border: 1px solid var(--accent-primary); border-radius: 3px; padding: 2px 6px; font-size: 13px; color: var(--text-primary); outline: none; }
.create-icon { width: 16px; text-align: center; font-size: 12px; }
</style>
