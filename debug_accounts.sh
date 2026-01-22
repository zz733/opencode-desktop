#!/bin/bash

echo "========================================"
echo "调试账号管理器状态"
echo "========================================"
echo ""

# 1. 检查数据目录
DATA_DIR="$HOME/Library/Application Support/OpenCode/KiroAccountManager/data"
echo "→ 数据目录: $DATA_DIR"

if [ -d "$DATA_DIR" ]; then
    echo "✓ 数据目录存在"
    ls -lh "$DATA_DIR"
else
    echo "✗ 数据目录不存在"
fi

echo ""

# 2. 检查加密文件
ENC_FILE="$DATA_DIR/accounts.json.enc"
if [ -f "$ENC_FILE" ]; then
    echo "✓ 加密账号文件存在"
    echo "  大小: $(ls -lh "$ENC_FILE" | awk '{print $5}')"
    echo "  修改时间: $(stat -f "%Sm" "$ENC_FILE")"
else
    echo "✗ 加密账号文件不存在"
fi

echo ""

# 3. 检查 OpenCode 配置
OPENCODE_CONFIG="$HOME/.config/opencode/kiro-accounts.json"
echo "→ OpenCode 配置: $OPENCODE_CONFIG"

if [ -f "$OPENCODE_CONFIG" ]; then
    echo "✓ OpenCode 配置文件存在"
    echo ""
    echo "当前账号:"
    cat "$OPENCODE_CONFIG" | jq -r '.accounts[] | "  邮箱: \(.email)\n  ID: \(.id)"'
else
    echo "✗ OpenCode 配置文件不存在"
fi

echo ""

# 4. 检查数据库
DB_FILE="$HOME/.config/opencode/kiro.db"
if [ -f "$DB_FILE" ]; then
    echo "✓ 数据库文件存在"
    echo ""
    echo "数据库中的账号:"
    sqlite3 "$DB_FILE" "SELECT email, id FROM accounts" | while read line; do
        echo "  $line"
    done
else
    echo "✗ 数据库文件不存在"
fi

echo ""
echo "========================================"
echo "建议："
echo "1. 查看应用终端输出，搜索 'account manager not initialized'"
echo "2. 查看浏览器控制台，查看完整错误信息"
echo "3. 重启应用: wails dev"
echo "========================================"
