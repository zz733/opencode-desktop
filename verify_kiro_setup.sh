#!/bin/bash

echo "=== Kiro 账号管理器环境检查 ==="
echo ""

# 检查账号管理器数据目录
echo "1. 检查账号管理器数据目录..."
DATA_DIR="$HOME/.config/opencode/data"
if [ -d "$DATA_DIR" ]; then
    echo "   ✓ 数据目录存在: $DATA_DIR"
    if [ -f "$DATA_DIR/accounts.json.enc" ]; then
        echo "   ✓ 账号数据文件存在"
        echo "   文件大小: $(ls -lh "$DATA_DIR/accounts.json.enc" | awk '{print $5}')"
        echo "   修改时间: $(ls -l "$DATA_DIR/accounts.json.enc" | awk '{print $6, $7, $8}')"
    else
        echo "   ✗ 账号数据文件不存在"
        echo "   → 还没有添加过账号"
    fi
else
    echo "   ✗ 数据目录不存在: $DATA_DIR"
    echo "   → 还没有添加过账号"
fi

echo ""

# 检查 OpenCode Token 文件
echo "2. 检查 OpenCode Token 文件..."
TOKEN_FILE="$HOME/.aws/sso/cache/kiro-auth-token.json"
if [ -f "$TOKEN_FILE" ]; then
    echo "   ✓ Token 文件存在: $TOKEN_FILE"
    echo "   文件大小: $(ls -lh "$TOKEN_FILE" | awk '{print $5}')"
    echo "   修改时间: $(ls -l "$TOKEN_FILE" | awk '{print $6, $7, $8}')"
    echo ""
    echo "   Token 内容预览:"
    if command -v jq &> /dev/null; then
        cat "$TOKEN_FILE" | jq '{authMethod, provider, expiresAt}'
    else
        cat "$TOKEN_FILE" | grep -E '"(authMethod|provider|expiresAt)"'
    fi
else
    echo "   ✗ Token 文件不存在"
    echo "   → 还没有切换过账号"
fi

echo ""

# 检查备份目录
echo "3. 检查备份目录..."
BACKUP_DIR="$HOME/.config/opencode/backups"
if [ -d "$BACKUP_DIR" ]; then
    echo "   ✓ 备份目录存在: $BACKUP_DIR"
    BACKUP_COUNT=$(ls -1 "$BACKUP_DIR" 2>/dev/null | wc -l)
    echo "   备份文件数量: $BACKUP_COUNT"
else
    echo "   ✗ 备份目录不存在"
fi

echo ""

# 检查 Kiro 数据目录
echo "4. 检查 Kiro 数据目录..."
KIRO_DIR="$HOME/Library/Application Support/Kiro"
if [ -d "$KIRO_DIR" ]; then
    echo "   ✓ Kiro 目录存在: $KIRO_DIR"
    if [ -f "$KIRO_DIR/User/globalStorage/storage.json" ]; then
        echo "   ✓ storage.json 存在"
    fi
    if [ -f "$KIRO_DIR/User/globalStorage/state.vscdb" ]; then
        echo "   ✓ state.vscdb 存在"
    fi
else
    echo "   ✗ Kiro 目录不存在"
    echo "   → Kiro IDE 可能未安装或未运行过"
fi

echo ""

# 检查旧的 AWS Builder ID 账号
echo "5. 检查旧的 AWS Builder ID 账号..."
OLD_ACCOUNTS="$HOME/.config/opencode/kiro-accounts.json"
if [ -f "$OLD_ACCOUNTS" ]; then
    echo "   ⚠ 发现旧的账号文件: $OLD_ACCOUNTS"
    echo "   → 这是 AWS Builder ID 账号，不是 Kiro 账号"
    if command -v jq &> /dev/null; then
        ACCOUNT_COUNT=$(cat "$OLD_ACCOUNTS" | jq '.accounts | length')
        echo "   AWS Builder ID 账号数量: $ACCOUNT_COUNT"
    fi
else
    echo "   ✓ 没有旧的账号文件"
fi

echo ""
echo "=== 检查完成 ==="
echo ""

# 提供建议
if [ ! -d "$DATA_DIR" ] || [ ! -f "$DATA_DIR/accounts.json.enc" ]; then
    echo "📝 建议："
    echo "   1. 启动应用: cd myapp && wails dev"
    echo "   2. 打开设置 → Kiro 账号"
    echo "   3. 点击"添加账号"按钮"
    echo "   4. 输入 Refresh Token 并保存"
    echo "   5. 再次运行此脚本验证"
fi
