#!/bin/bash

FILE=~/.config/opencode/kiro-accounts.json

echo "开始监控 $FILE"
echo "当前内容:"
cat $FILE | jq -r '.accounts[0].email'
echo ""
echo "等待文件变化..."
echo ""

# 使用 fswatch 监控文件变化
fswatch -o $FILE | while read change; do
    echo "========================================="
    echo "检测到文件变化！时间: $(date)"
    echo "新的账号:"
    cat $FILE | jq -r '.accounts[0].email'
    echo ""
    echo "是谁修改的？检查进程:"
    lsof $FILE 2>/dev/null || echo "没有进程打开此文件"
    echo "========================================="
    echo ""
done
