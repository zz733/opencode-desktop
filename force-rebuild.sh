#!/bin/bash

echo "=== 强制重新编译 ==="

# 1. 杀死所有相关进程
echo "→ 停止所有进程..."
pkill -9 myapp 2>/dev/null
pkill -9 vite 2>/dev/null
pkill -9 node 2>/dev/null

# 2. 清理前端缓存
echo "→ 清理前端缓存..."
rm -rf frontend/dist
rm -rf frontend/node_modules/.vite
rm -rf frontend/.vite

# 3. 清理 Wails 缓存
echo "→ 清理 Wails 缓存..."
rm -rf build

# 4. 重新生成绑定
echo "→ 重新生成绑定..."
wails generate module

# 5. 重新编译
echo "→ 重新编译..."
wails build

echo "✓ 完成！"
echo ""
echo "现在运行: ./build/bin/myapp.app/Contents/MacOS/myapp"
