#!/bin/bash
# 捕获 OpenCode SSE 事件
echo "监听 OpenCode 事件..."
curl -s "http://localhost:4096/event" | while read line; do
  if [[ "$line" == data:* ]]; then
    echo "$line"
  fi
done
