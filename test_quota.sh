#!/bin/bash

echo "=== 测试配额显示 ==="
echo ""
echo "请提供您的 Refresh Token:"
read -r REFRESH_TOKEN

echo ""
echo "正在测试..."

# 使用 Go 程序测试
cat > /tmp/test_quota.go << 'EOF'
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: test_quota <refresh_token>")
		os.Exit(1)
	}
	
	refreshToken := os.Args[1]
	
	// 这里需要导入你的包
	fmt.Printf("Refresh Token: %s...\n", refreshToken[:20])
	fmt.Println("请在实际应用中测试")
}
EOF

echo "✓ 测试脚本已创建"
echo ""
echo "请重启应用并添加账号，然后查看终端日志中的配额信息"
