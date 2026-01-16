#!/bin/bash

# 测试 OpenCode API

PORT=4096
BASE_URL="http://localhost:$PORT"

echo "=== 测试 OpenCode API ==="
echo ""

# 1. 检查连接
echo "1. 检查服务器连接..."
curl -s "$BASE_URL/session" | jq '.' || echo "连接失败"
echo ""

# 2. 创建会话
echo "2. 创建新会话..."
SESSION_RESPONSE=$(curl -s -X POST "$BASE_URL/session" -H "Content-Type: application/json" -d '{}')
echo "响应: $SESSION_RESPONSE"
SESSION_ID=$(echo $SESSION_RESPONSE | jq -r '.info.id')
echo "会话 ID: $SESSION_ID"

if [ "$SESSION_ID" == "null" ] || [ -z "$SESSION_ID" ]; then
  echo "创建会话失败，使用现有会话"
  SESSION_ID=$(curl -s "$BASE_URL/session" | jq -r '.[0].id')
  echo "使用会话: $SESSION_ID"
fi
echo ""

# 3. 发送消息（异步）
echo "3. 发送测试消息..."
curl -s -X POST "$BASE_URL/session/$SESSION_ID/prompt_async" \
  -H "Content-Type: application/json" \
  -d '{
    "parts": [{"type": "text", "text": "Hello, please respond with a simple greeting in Chinese."}],
    "model": {"providerID": "opencode", "modelID": "big-pickle"}
  }'
echo ""
echo "消息已发送，等待响应..."
echo ""

# 4. 订阅事件流（等待 10 秒）
echo "4. 监听事件流（10秒）..."
# macOS 没有 timeout 命令，使用 perl 替代
perl -e 'alarm 10; exec @ARGV' curl -s -N "$BASE_URL/event" 2>/dev/null | while IFS= read -r line; do
  if [[ $line == data:* ]]; then
    DATA=$(echo "$line" | sed 's/^data://')
    echo "$DATA" | jq -c '.' 2>/dev/null || echo "$DATA"
  fi
done
echo ""

# 5. 获取会话消息
echo "5. 获取会话历史消息..."
curl -s "$BASE_URL/session/$SESSION_ID/message" | jq '.'
echo ""

echo "=== 测试完成 ==="
