package main

import (
	"fmt"
	"log"
	"testing"
)

// TestAddKiroAccountManual 手动测试添加 Kiro 账号
func TestAddKiroAccountManual(t *testing.T) {
	fmt.Println("=== Kiro 账号添加测试 ===")

	// 1. 初始化服务
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
	fmt.Println("   ✓ 服务初始化完成")

	// 2. 测试 Refresh Token
	refreshToken := "aorAAAAAGnLGQgY_JGCEaAu31zq9-VgAlp1em-13e_w6H4pNt4aq17R2Ot_1LNtlrZASD8jR6JKRg5NVUWG5HZRdgBkc0:MGUCMQCuOP9WeTpHKpyyFSo/Q6M0NDCBKOvnnkPq15udRiV6EsyXa5lDxb+beSdukMSZ7s4CMCcIfnOZwQkXyBtPAT5sFPQdyBl8iMDZv7VBM/3l99RKBeOVSGKbqtVU6aIAik539A"

	fmt.Println("2. 测试 Refresh Token...")
	fmt.Printf("   Token 长度: %d\n", len(refreshToken))

	kiroClient := NewKiroAPIClient()

	// 3. 刷新 Token
	fmt.Println("")
	fmt.Println("3. 刷新 Token 获取 Access Token...")
	tokenResp, err := kiroClient.RefreshKiroToken(refreshToken)
	if err != nil {
		log.Fatalf("   ✗ 刷新 Token 失败: %v", err)
	}
	fmt.Println("   ✓ Token 刷新成功")
	fmt.Printf("   Access Token 长度: %d\n", len(tokenResp.AccessToken))
	fmt.Printf("   过期时间: %d 秒\n", tokenResp.ExpiresIn)

	// 4. 获取用户信息和配额
	fmt.Println("")
	fmt.Println("4. 获取用户信息和配额...")
	usageResp, err := kiroClient.GetKiroUsageLimits(tokenResp.AccessToken)
	if err != nil {
		log.Fatalf("   ✗ 获取配额失败: %v", err)
	}
	fmt.Println("   ✓ 配额信息获取成功")

	if usageResp.UserInfo != nil {
		fmt.Printf("   邮箱: %s\n", usageResp.UserInfo.Email)
		fmt.Printf("   用户 ID: %s\n", usageResp.UserInfo.UserID)
	}

	if usageResp.SubscriptionInfo != nil {
		fmt.Printf("   订阅类型: %s\n", usageResp.SubscriptionInfo.SubscriptionTitle)
	}

	if len(usageResp.UsageBreakdownList) > 0 {
		breakdown := usageResp.UsageBreakdownList[0]
		fmt.Printf("   主配额: %d / %d\n", breakdown.CurrentUsage, breakdown.UsageLimit)
		if breakdown.FreeTrialInfo != nil {
			fmt.Printf("   试用配额: %d / %d\n", breakdown.FreeTrialInfo.CurrentUsage, breakdown.FreeTrialInfo.UsageLimit)
		}
	}

	// 5. 转换为账号对象
	fmt.Println("")
	fmt.Println("5. 创建账号对象...")
	account := ConvertKiroResponseToAccount(tokenResp, usageResp, accountMgr)
	fmt.Printf("   账号 ID: %s\n", account.ID)
	fmt.Printf("   邮箱: %s\n", account.Email)
	fmt.Printf("   显示名称: %s\n", account.DisplayName)
	fmt.Printf("   订阅类型: %s\n", account.SubscriptionType)
	fmt.Printf("   主配额: %d / %d\n", account.Quota.Main.Used, account.Quota.Main.Total)
	fmt.Printf("   试用配额: %d / %d\n", account.Quota.Trial.Used, account.Quota.Trial.Total)

	// 6. 添加账号
	fmt.Println("")
	fmt.Println("6. 添加账号到管理器...")
	if err := accountMgr.AddAccount(account); err != nil {
		log.Fatalf("   ✗ 添加账号失败: %v", err)
	}
	fmt.Println("   ✓ 账号添加成功")

	// 7. 验证账号列表
	fmt.Println("")
	fmt.Println("7. 验证账号列表...")
	accounts := accountMgr.ListAccounts()
	fmt.Printf("   账号总数: %d\n", len(accounts))
	for i, acc := range accounts {
		fmt.Printf("   [%d] %s (%s) - %s\n", i+1, acc.Email, acc.DisplayName, acc.SubscriptionType)
		fmt.Printf("       主配额: %d/%d, 试用: %d/%d\n",
			acc.Quota.Main.Used, acc.Quota.Main.Total,
			acc.Quota.Trial.Used, acc.Quota.Trial.Total)
		fmt.Printf("       激活状态: %v\n", acc.IsActive)
	}

	// 8. 获取存储路径
	fmt.Println("")
	fmt.Println("8. 存储信息...")
	paths := configMgr.GetPaths()
	fmt.Printf("   数据目录: %s\n", paths.DataDir)
	fmt.Printf("   账号文件: %s\n", paths.AccountsFile)
	fmt.Printf("   备份目录: %s\n", paths.BackupDir)

	fmt.Println("")
	fmt.Println("=== 测试完成 ===")
}
