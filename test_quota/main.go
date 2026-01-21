package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run main.go <refresh_token>")
		os.Exit(1)
	}

	refreshToken := os.Args[1]
	
	fmt.Println("=== 测试 Kiro 账号配额 ===")
	fmt.Println()
	
	// Step 1: 刷新 Token 获取 Access Token
	fmt.Println("步骤 1: 刷新 Token...")
	authURL := "https://prod.us-east-1.auth.desktop.kiro.dev/refreshToken"
	
	payload := map[string]string{
		"refreshToken": refreshToken,
	}
	
	payloadBytes, _ := json.Marshal(payload)
	
	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Printf("✗ 创建请求失败: %v\n", err)
		os.Exit(1)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	client := &http.Client{Timeout: 30 * time.Second}
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("✗ 请求失败: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	bodyBytes, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != 200 {
		fmt.Printf("✗ Token 刷新失败 (状态: %d)\n", resp.StatusCode)
		fmt.Printf("  响应: %s\n", string(bodyBytes))
		os.Exit(1)
	}
	
	var tokenResp struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresIn    int64  `json:"expiresIn"`
		ProfileArn   string `json:"profileArn"`
	}
	
	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		fmt.Printf("✗ 解析响应失败: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("✓ Token 刷新成功\n")
	fmt.Printf("  Access Token 长度: %d\n", len(tokenResp.AccessToken))
	fmt.Printf("  过期时间: %d 秒\n", tokenResp.ExpiresIn)
	fmt.Printf("  Profile ARN: %s\n\n", tokenResp.ProfileArn)
	
	// Step 2: 获取使用量
	fmt.Println("步骤 2: 获取使用量...")
	
	profileArn := url.QueryEscape(tokenResp.ProfileArn)
	usageURL := fmt.Sprintf(
		"https://codewhisperer.us-east-1.amazonaws.com/getUsageLimits?isEmailRequired=true&origin=AI_EDITOR&profileArn=%s",
		profileArn,
	)
	
	req2, err := http.NewRequest("GET", usageURL, nil)
	if err != nil {
		fmt.Printf("✗ 创建请求失败: %v\n", err)
		os.Exit(1)
	}
	
	req2.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	req2.Header.Set("Accept", "application/json")
	
	resp2, err := client.Do(req2)
	if err != nil {
		fmt.Printf("✗ 请求失败: %v\n", err)
		os.Exit(1)
	}
	defer resp2.Body.Close()
	
	bodyBytes2, _ := io.ReadAll(resp2.Body)
	
	if resp2.StatusCode != 200 {
		fmt.Printf("✗ 获取使用量失败 (状态: %d)\n", resp2.StatusCode)
		fmt.Printf("  响应: %s\n", string(bodyBytes2))
		os.Exit(1)
	}
	
	fmt.Printf("✓ 获取使用量成功\n\n")
	
	// 解析并显示使用量
	var usageResp map[string]interface{}
	if err := json.Unmarshal(bodyBytes2, &usageResp); err != nil {
		fmt.Printf("✗ 解析响应失败: %v\n", err)
		os.Exit(1)
	}
	
	// 格式化输出
	prettyJSON, _ := json.MarshalIndent(usageResp, "", "  ")
	fmt.Println("=== 账号信息 ===")
	fmt.Println(string(prettyJSON))
	
	// 提取关键信息
	fmt.Println("\n=== 配额摘要 ===")
	
	if userInfo, ok := usageResp["userInfo"].(map[string]interface{}); ok {
		if email, ok := userInfo["email"].(string); ok {
			fmt.Printf("邮箱: %s\n", email)
		}
	}
	
	if subInfo, ok := usageResp["subscriptionInfo"].(map[string]interface{}); ok {
		if title, ok := subInfo["subscriptionTitle"].(string); ok {
			fmt.Printf("订阅类型: %s\n", title)
		}
	}
	
	if usageList, ok := usageResp["usageBreakdownList"].([]interface{}); ok && len(usageList) > 0 {
		if usage, ok := usageList[0].(map[string]interface{}); ok {
			current := int(usage["currentUsage"].(float64))
			limit := int(usage["usageLimit"].(float64))
			fmt.Printf("使用量: %d / %d\n", current, limit)
			fmt.Printf("剩余: %d\n", limit-current)
			
			// 显示试用配额
			if freeTrialInfo, ok := usage["freeTrialInfo"].(map[string]interface{}); ok {
				trialCurrent := int(freeTrialInfo["currentUsage"].(float64))
				trialLimit := int(freeTrialInfo["usageLimit"].(float64))
				fmt.Printf("试用配额: %d / %d (剩余: %d)\n", trialCurrent, trialLimit, trialLimit-trialCurrent)
			}
		}
	}
}
