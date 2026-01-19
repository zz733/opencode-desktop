#!/bin/bash

echo "=== 测试账号切换 ==="
echo ""
echo "1. 当前 kiro-accounts.json 内容："
cat ~/.config/opencode/kiro-accounts.json | jq '.accounts[] | {email, id}'
echo ""
echo "2. 请在应用中切换账号..."
echo "3. 按回车键查看切换后的结果..."
read
echo ""
echo "切换后的 kiro-accounts.json 内容："
cat ~/.config/opencode/kiro-accounts.json | jq '.accounts[] | {email, id}'
echo ""
echo "=== 测试完成 ==="
