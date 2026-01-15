<script setup>
import { WindowMinimise, WindowToggleMaximise, Quit } from '../../wailsjs/runtime/runtime'

const minimize = () => WindowMinimise()
const toggleMaximize = () => WindowToggleMaximise()
const close = () => Quit()
</script>

<template>
  <div class="titlebar" style="--wails-draggable:drag">
    <!-- 左侧菜单 -->
    <div class="titlebar-left">
      <div class="menu-item">File</div>
      <div class="menu-item">Edit</div>
      <div class="menu-item">View</div>
      <div class="menu-item">Help</div>
    </div>
    
    <!-- 中间标题 -->
    <div class="titlebar-center">
      OpenCode Desktop
    </div>
    
    <!-- 右侧窗口控制按钮 -->
    <div class="titlebar-right" style="--wails-draggable:no-drag">
      <button class="window-btn" @click="minimize">
        <svg width="10" height="10" viewBox="0 0 10 10">
          <path d="M0 5h10" stroke="currentColor" stroke-width="1"/>
        </svg>
      </button>
      <button class="window-btn" @click="toggleMaximize">
        <svg width="10" height="10" viewBox="0 0 10 10">
          <rect x="0.5" y="0.5" width="9" height="9" stroke="currentColor" stroke-width="1" fill="none"/>
        </svg>
      </button>
      <button class="window-btn close" @click="close">
        <svg width="10" height="10" viewBox="0 0 10 10">
          <path d="M0 0l10 10M10 0L0 10" stroke="currentColor" stroke-width="1"/>
        </svg>
      </button>
    </div>
  </div>
</template>

<style scoped>
.titlebar {
  height: 30px;
  background: #211d25;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 8px;
  user-select: none;
  border-bottom: 1px solid #28242e;
}

.titlebar-left {
  display: flex;
  align-items: center;
  gap: 4px;
}

.menu-item {
  padding: 4px 10px;
  font-size: 12px;
  color: #938f9b;
  cursor: pointer;
  border-radius: 4px;
}

.menu-item:hover {
  background: #322e3a;
  color: #ffffff;
}

.titlebar-center {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  font-size: 12px;
  color: #938f9b;
}

.titlebar-right {
  display: flex;
  align-items: center;
}

.window-btn {
  width: 46px;
  height: 30px;
  background: transparent;
  border: none;
  color: #938f9b;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.window-btn:hover {
  background: #322e3a;
  color: #ffffff;
}

.window-btn.close:hover {
  background: #e81123;
  color: #ffffff;
}
</style>
