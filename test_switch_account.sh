#!/bin/bash

echo "========================================="
echo "测试账号切换功能"
echo "========================================="
echo ""

echo "1. 当前 OpenCode 配置文件内容："
echo "-----------------------------------"
cat ~/.config/opencode/kiro-accounts.json | jq '.accounts[0].email, .accounts[0].realEmail' 2>/dev/null || cat ~/.config/opencode/kiro-accounts.json | head -20
echo ""

echo "2. myapp 数据目录："
echo "-----------------------------------"
ls -la ~/Library/Application\ Support/OpenCode/KiroAccountManager/data/
echo ""

echo "3. 检查日志文件："
echo "-----------------------------------"
if [ -f /tmp/kiro-account-manager.log ]; then
    echo "最近的日志："
    tail -50 /tmp/kiro-account-manager.log
else
    echo "日志文件不存在"
fi
echo ""

echo "========================================="
echo "请在 myapp 中点击切换账号按钮"
echo "然后按回车键查看结果..."
echo "========================================="
read

echo ""
echo "切换后的 OpenCode 配置："
echo "-----------------------------------"
cat ~/.config/opencode/kiro-accounts.json | jq '.accounts[0].email, .accounts[0].realEmail' 2>/dev/null || cat ~/.config/opencode/kiro-accounts.json | head -20
echo ""

echo "检查日志更新："
echo "-----------------------------------"
if [ -f /tmp/kiro-account-manager.log ]; then
    echo "最新的日志："
    tail -50 /tmp/kiro-account-manager.log
else
    echo "日志文件不存在"
fi
