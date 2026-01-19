// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run cmd_add_kiro_account.go <refresh_token>")
		os.Exit(1)
	}

	refreshToken := os.Args[1]

	fmt.Println("=== 添加 Kiro 账号 ===")
	fmt.Printf("Refresh Token 长度: %d\n\n", len(refreshToken))

	// 初始化服务
	fmt.Println("1. 初始化服务...")
	crypto := NewCryptoService("opencode-kiro-master-key-v1")
	configMgr, err := NewConfigManager(crypto)
	if err != nil {
		log.Fatalf("创建配置管理器失败: %v", err)
	}

	if err := configMgr.Initialize(); err != nil {
		log.Fatalf("初始化配置失败: %v", err)
	}

	dataDir := configMgr.GetDataDirectory()
	fmt.Printf("   数据目录: %s\n", dataDir)

	storage := NewStorageService(dataDir, crypto)
	accountMgr := NewAccountManager(storage, crypto)
	fmt.Println("   ✓ 初始化完成")

	// 创建 API 客户端
	fmt.Println("\n2. 调用 Kiro API...")
	kiroClient := NewKiroAPIClient()

	// 刷新 Token
	tokenResp, err := kiroClient.RefreshKiroToken(refreshToken)
	if err != nil {
		log.Fatalf("   ✗ 刷新 Token 失败: %v", err)
	}
	fmt.Println("   ✓ Token 刷新成功")

	// 获取配额信息
	usageResp, err := kiroClient.GetKiroUsageLimits(tokenResp.AccessToken)
	if err != nil {
		log.Fatalf("   ✗ 获取配额失败: %v", err)
	}
	fmt.Println("   ✓ 配额信息获取成功")

	// 显示用户信息
	if usageResp.UserInfo != nil {
		fmt.Printf("   邮箱: %s\n", usageResp.UserInfo.Email)
	}
	if usageResp.SubscriptionInfo != nil {
		fmt.Printf("   订阅: %s\n", usageResp.SubscriptionInfo.SubscriptionTitle)
	}

	// 转换为账号对象
	fmt.Println("\n3. 创建账号...")
	account := ConvertKiroResponseToAccount(tokenResp, usageResp, accountMgr)
	fmt.Printf("   ID: %s\n", account.ID)
	fmt.Printf("   邮箱: %s\n", account.Email)
	fmt.Printf("   主配额: %d/%d\n", account.Quota.Main.Used, account.Quota.Main.Total)
	fmt.Printf("   试用配额: %d/%d\n", account.Quota.Trial.Used, account.Quota.Trial.Total)

	// 添加账号
	fmt.Println("\n4. 保存账号...")
	if err := accountMgr.AddAccount(account); err != nil {
		log.Fatalf("   ✗ 添加失败: %v", err)
	}
	fmt.Println("   ✓ 账号已保存")

	// 验证
	fmt.Println("\n5. 验证...")
	accounts := accountMgr.ListAccounts()
	fmt.Printf("   账号总数: %d\n", len(accounts))
	for i, acc := range accounts {
		fmt.Printf("   [%d] %s - %s (主: %d/%d, 试用: %d/%d)\n",
			i+1, acc.Email, acc.SubscriptionType,
			acc.Quota.Main.Used, acc.Quota.Main.Total,
			acc.Quota.Trial.Used, acc.Quota.Trial.Total)
	}

	fmt.Println("\n✓ 完成！")
}
