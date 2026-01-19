package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	KiroAuthAPIBase  = "https://prod.us-east-1.auth.desktop.kiro.dev"
	KiroUsageAPIBase = "https://codewhisperer.us-east-1.amazonaws.com"
	KiroProfileARN   = "arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run test.go <refresh_token>")
		os.Exit(1)
	}

	refreshToken := os.Args[1]
	
	fmt.Println("=== 测试 Kiro Token 刷新 ===")
	fmt.Printf("Refresh Token 长度: %d\n", len(refreshToken))
	fmt.Println()

	// 步骤 1: 刷新 Token
	fmt.Println("步骤 1: 调用 RefreshKiroToken...")
	accessToken, err := refreshKiroToken(refreshToken)
	if err != nil {
		fmt.Printf("❌ 刷新失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Token 刷新成功")
	fmt.Printf("  Access Token 长度: %d\n", len(accessToken))
	fmt.Println()

	// 步骤 2: 获取配额信息
	fmt.Println("步骤 2: 调用 GetKiroUsageLimits...")
	err = getKiroUsageLimits(accessToken)
	if err != nil {
		fmt.Printf("❌ 获取配额失败: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println()
	fmt.Println("=== 测试完成 ===")
}

func refreshKiroToken(refreshToken string) (string, error) {
	urlStr := fmt.Sprintf("%s/refreshToken", KiroAuthAPIBase)

	body := map[string]string{
		"refreshToken": refreshToken,
	}

	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", urlStr, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("网络错误: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("  HTTP 状态码: %d\n", resp.StatusCode)

	if resp.StatusCode == 401 {
		return "", fmt.Errorf("RefreshToken 已过期或无效")
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("刷新失败 (状态码: %d)", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	accessToken, ok := result["accessToken"].(string)
	if !ok {
		return "", fmt.Errorf("响应中没有 accessToken")
	}

	return accessToken, nil
}

func getKiroUsageLimits(accessToken string) error {
	urlStr := fmt.Sprintf("%s/getUsageLimits?isEmailRequired=true&origin=AI_EDITOR&profileArn=%s",
		KiroUsageAPIBase,
		url.QueryEscape(KiroProfileARN))

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("网络错误: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("  HTTP 状态码: %d\n", resp.StatusCode)

	if resp.StatusCode != 200 {
		return fmt.Errorf("获取配额失败 (状态码: %d)", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	fmt.Println("✓ 配额信息获取成功")
	
	// 打印响应
	jsonBytes, _ := json.MarshalIndent(result, "  ", "  ")
	fmt.Printf("  响应内容:\n%s\n", string(jsonBytes))

	return nil
}
