#!/bin/bash

echo "直接测试账号切换功能"
echo "========================================"
echo ""

# 先恢复 OpenCode 配置文件
echo "→ 恢复 OpenCode 配置文件..."
if [ -f "$HOME/.config/opencode/kiro-accounts.json.bak" ]; then
    cp "$HOME/.config/opencode/kiro-accounts.json.bak" "$HOME/.config/opencode/kiro-accounts.json"
    echo "✓ 配置文件已恢复"
else
    echo "⚠ 备份文件不存在，跳过"
fi

echo ""
echo "→ 当前 OpenCode 配置:"
if [ -f "$HOME/.config/opencode/kiro-accounts.json" ]; then
    cat "$HOME/.config/opencode/kiro-accounts.json" | jq -r '.accounts[] | "  邮箱: \(.email)"'
else
    echo "  文件不存在"
fi

echo ""
echo "========================================"
echo "现在请："
echo "1. 在应用中点击切换按钮"
echo "2. 查看终端输出"
echo "3. 查看浏览器控制台"
echo "========================================"
