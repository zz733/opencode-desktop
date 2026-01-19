#!/bin/bash

echo "=== 检查存储的账号数据 ==="
echo ""

# 查找可能的存储位置
echo "1. 查找 Wails 应用数据目录..."
find ~/Library/Application\ Support -name "myapp" -type d 2>/dev/null | head -3

echo ""
echo "2. 查找可能的账号数据文件..."
find ~/Library/Application\ Support -name "*account*" -o -name "*kiro*" 2>/dev/null | grep -i myapp | head -10

echo ""
echo "3. 当前 OpenCode 配置文件中的账号："
cat ~/.config/opencode/kiro-accounts.json | python3 -c "
import json, sys
data = json.load(sys.stdin)
for acc in data.get('accounts', []):
    print(f\"  - ID: {acc.get('id', 'N/A')[:20]}...\")
    print(f\"    Email: {acc.get('email', 'N/A')}\")
    print(f\"    RealEmail: {acc.get('realEmail', 'N/A')}\")
    print(f\"    AuthMethod: {acc.get('authMethod', 'N/A')}\")
    print()
"

echo "=== 检查完成 ==="
