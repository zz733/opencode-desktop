#!/bin/bash

echo "========================================"
echo "测试账号切换自动化修复"
echo "========================================"
echo ""

# 1. 检查当前配置文件状态
echo "→ 检查 OpenCode 配置文件..."
KIRO_CONFIG="$HOME/.config/opencode/kiro-accounts.json"

if [ -f "$KIRO_CONFIG" ]; then
    echo "✓ 配置文件存在: $KIRO_CONFIG"
    
    # 检查文件内容
    echo ""
    echo "→ 当前配置内容:"
    cat "$KIRO_CONFIG" | jq -r '.accounts[] | "  邮箱: \(.email)\n  ID: \(.id)\n  过期时间: \(.expiresAt)"'
    
    # 检查 token 是否过期
    EXPIRES_AT=$(cat "$KIRO_CONFIG" | jq -r '.accounts[0].expiresAt')
    CURRENT_TIME=$(date +%s)000  # 转换为毫秒
    
    echo ""
    if [ "$EXPIRES_AT" -lt "$CURRENT_TIME" ]; then
        echo "⚠ Token 已过期"
        echo "  过期时间: $(date -r $(($EXPIRES_AT / 1000)) '+%Y-%m-%d %H:%M:%S')"
        echo "  当前时间: $(date '+%Y-%m-%d %H:%M:%S')"
    else
        echo "✓ Token 有效"
        REMAINING=$(( ($EXPIRES_AT - $CURRENT_TIME) / 1000 / 60 ))
        echo "  剩余时间: ${REMAINING} 分钟"
    fi
else
    echo "✗ 配置文件不存在: $KIRO_CONFIG"
fi

echo ""
echo "========================================"
echo "测试说明："
echo "1. 启动应用: wails dev"
echo "2. 打开 Kiro 账户管理"
echo "3. 点击切换按钮"
echo "4. 观察终端日志，应该看到："
echo "   - Token 过期检查"
echo "   - 自动刷新 Token"
echo "   - 配置文件创建/更新"
echo "========================================"
