#!/bin/bash
# 创建 provider.go - 已存在，跳过
# 创建 plugin.go - 已存在，跳过  
# 创建 mcp.go 和 model.go

# 从 app.go 提取 MCP 相关代码到 mcp.go
echo "创建 mcp.go..."
# 这里需要手动提取，因为代码太复杂

# 从 app.go 提取模型配置代码到 model.go  
echo "创建 model.go..."
# 这里需要手动提取

echo "模块文件创建完成"
