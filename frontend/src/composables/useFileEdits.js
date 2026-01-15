import { ref } from 'vue'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { ReadFileContent, WriteFileContent } from '../../wailsjs/go/main/App'

// 文件编辑记录
const fileEdits = ref([]) // { id, path, filename, oldContent, newContent, timestamp, reverted }

let editId = 0
let initialized = false

// 初始化监听
function initFileEditListener() {
  if (initialized) return
  initialized = true
  
  EventsOn('file-changed', async (changedPath) => {
    // 检查是否已有该文件的待处理记录
    const existing = fileEdits.value.find(e => e.path === changedPath && !e.reverted)
    if (existing) {
      // 更新新内容
      try {
        existing.newContent = await ReadFileContent(changedPath)
      } catch (e) {}
      return
    }
  })
}

// 记录文件编辑（在文件被修改前调用）
async function recordEdit(path, oldContent) {
  const filename = path.split('/').pop()
  
  // 读取新内容
  let newContent = ''
  try {
    newContent = await ReadFileContent(path)
  } catch (e) {
    return null
  }
  
  // 如果内容相同，不记录
  if (oldContent === newContent) return null
  
  const edit = {
    id: ++editId,
    path,
    filename,
    oldContent,
    newContent,
    timestamp: Date.now(),
    reverted: false
  }
  
  fileEdits.value.push(edit)
  return edit
}

// 添加编辑记录（外部调用）
function addEdit(path, oldContent, newContent) {
  if (oldContent === newContent) return null
  
  const filename = path.split('/').pop()
  const edit = {
    id: ++editId,
    path,
    filename,
    oldContent,
    newContent,
    timestamp: Date.now(),
    reverted: false
  }
  
  fileEdits.value.push(edit)
  return edit
}

// 撤销编辑
async function revertEdit(editId) {
  const edit = fileEdits.value.find(e => e.id === editId)
  if (!edit || edit.reverted) return false
  
  try {
    await WriteFileContent(edit.path, edit.oldContent)
    edit.reverted = true
    return true
  } catch (e) {
    console.error('撤销失败:', e)
    return false
  }
}

// 清除编辑记录
function clearEdit(editId) {
  const index = fileEdits.value.findIndex(e => e.id === editId)
  if (index > -1) {
    fileEdits.value.splice(index, 1)
  }
}

// 清除所有记录
function clearAllEdits() {
  fileEdits.value = []
}

// 获取文件的最新编辑记录
function getLatestEdit(path) {
  return fileEdits.value.filter(e => e.path === path && !e.reverted).pop()
}

export function useFileEdits() {
  initFileEditListener()
  
  return {
    fileEdits,
    addEdit,
    recordEdit,
    revertEdit,
    clearEdit,
    clearAllEdits,
    getLatestEdit
  }
}
