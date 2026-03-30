package main

import "testing"

func TestCCOpenAIBridgeResolveModel(t *testing.T) {
	bridge := &CCOpenAIBridge{
		platform:      "wechat",
		selectedModel: "qwen/qwen3.5-plus",
	}

	cases := []struct {
		name      string
		requested string
		expected  string
	}{
		{name: "empty", requested: "", expected: "qwen/qwen3.5-plus"},
		{name: "project alias", requested: "qclaw-project", expected: "qwen/qwen3.5-plus"},
		{name: "platform alias", requested: "wechat", expected: "qwen/qwen3.5-plus"},
		{name: "explicit model", requested: "openai/gpt-4o-mini", expected: "openai/gpt-4o-mini"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if actual := bridge.resolveModel(tc.requested); actual != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}

func TestCCOpenAIBridgeBuildPrompt(t *testing.T) {
	bridge := &CCOpenAIBridge{}

	messages := []bridgeChatMessage{
		{Role: "system", Content: "你是一个代码助手"},
		{Role: "user", Content: "先分析问题"},
		{Role: "assistant", Content: "我正在分析"},
		{Role: "user", Content: "给出最终方案"},
	}

	actual := bridge.buildPrompt(messages)
	expected := "系统指令:\n你是一个代码助手\n\n用户:\n先分析问题\n\n助手:\n我正在分析\n\n用户:\n给出最终方案\n\n请基于以上对话继续回复最后一条用户消息，保持上下文一致。"

	if actual != expected {
		t.Fatalf("expected prompt %q, got %q", expected, actual)
	}
}

func TestCCOpenAIBridgeExtractMessageText(t *testing.T) {
	bridge := &CCOpenAIBridge{}

	content := []interface{}{
		map[string]interface{}{"type": "text", "text": "第一段"},
		map[string]interface{}{"type": "input_text", "text": "第二段"},
		map[string]interface{}{"type": "image_url", "image_url": "ignored"},
	}

	actual := bridge.extractMessageText(content)
	if actual != "第一段\n第二段" {
		t.Fatalf("expected merged text, got %q", actual)
	}
}

func TestCCOpenAIBridgeMatchSessionEvent(t *testing.T) {
	bridge := &CCOpenAIBridge{}

	event := bridgeEvent{
		Type: "message.part.updated",
		Properties: map[string]interface{}{
			"part": map[string]interface{}{
				"sessionID": "ses_123",
				"type":      "text",
				"text":      "hello",
			},
		},
	}

	if !bridge.matchSessionEvent("ses_123", event) {
		t.Fatalf("expected event to match session")
	}
	if bridge.matchSessionEvent("ses_456", event) {
		t.Fatalf("expected event not to match different session")
	}
}

func TestDiffText(t *testing.T) {
	if actual := diffText("hello", "hello world"); actual != " world" {
		t.Fatalf("expected incremental delta, got %q", actual)
	}
	if actual := diffText("hello", "world"); actual != "world" {
		t.Fatalf("expected full replacement delta, got %q", actual)
	}
}
