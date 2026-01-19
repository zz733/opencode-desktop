package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run . <refresh_token>")
		os.Exit(1)
	}

	refreshToken := os.Args[1]
	
	fmt.Println("=== 测试 Kiro Token 刷新 ===")
	fmt.Printf("Refresh Token 长度: %d\n", len(refreshToken))
	fmt.Println()

	// 创建 API 客户端
	client := NewKiroAPIClient()

	// 步骤 1: 刷新 Token
	fmt.Println("步骤 1: 调用 RefreshKiroToken...")
	tokenResp, err := client.RefreshKiroToken(refreshToken)
	if err != nil {
		fmt.Printf("❌ 刷新失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Token 刷新成功")
	fmt.Printf("  Access Token 长度: %d\n", len(tokenResp.AccessToken))
	fmt.Printf("  Refresh Token 长度: %d\n", len(tokenResp.RefreshToken))
	fmt.Printf("  过期时间: %d 秒\n", tokenResp.ExpiresIn)
	fmt.Println()

	// 步骤 2: 获取配额信息
	fmt.Println("步骤 2: 调用 GetKiroUsageLimits...")
	usageResp, err := client.GetKiroUsageLimits(tokenResp.AccessToken)
	if err != nil {
		fmt.Printf("❌ 获取配额失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ 配额信息获取成功")
	
	if usageResp.UserInfo != nil {
		fmt.Printf("  邮箱: %s\n", usageResp.UserInfo.Email)
		fmt.Printf("  用户ID: %s\n", usageResp.UserInfo.UserID)
	}
	
	if usageResp.SubscriptionInfo != nil {
		fmt.Printf("  订阅类型: %s\n", usageResp.SubscriptionInfo.Type)
		fmt.Printf("  订阅标题: %s\n", usageResp.SubscriptionInfo.SubscriptionTitle)
	}
	
	if len(usageResp.UsageBreakdownList) > 0 {
		breakdown := usageResp.UsageBreakdownList[0]
		fmt.Printf("  主配额: %d/%d\n", breakdown.CurrentUsage, breakdown.UsageLimit)
		
		if breakdown.FreeTrialInfo != nil {
			fmt.Printf("  试用配额: %d/%d\n", breakdown.FreeTrialInfo.CurrentUsage, breakdown.FreeTrialInfo.UsageLimit)
		}
		
		if len(breakdown.Bonuses) > 0 {
			totalReward := 0
			totalRewardUsed := 0
			for _, bonus := range breakdown.Bonuses {
				totalReward += int(bonus.UsageLimit)
				totalRewardUsed += int(bonus.CurrentUsage)
			}
			fmt.Printf("  奖励配额: %d/%d\n", totalRewardUsed, totalReward)
		}
	}
	
	fmt.Println()
	fmt.Println("=== 测试完成 ===")
}
