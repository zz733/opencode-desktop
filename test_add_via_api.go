package main

import (
	"fmt"
	"testing"
)

// TestAddAccountViaAPI 测试通过 API 添加账号
func TestAddAccountViaAPI(t *testing.T) {
	// 创建 App 实例
	app := NewApp()
	
	// 模拟前端调用
	refreshToken := "aorAAAAAGnLGQgY_JGCEaAu31zq9-VgAlp1em-13e_w6H4pNt4aq17R2Ot_1LNtlrZASD8jR6JKRg5NVUWG5HZRdgBkc0:MGUCMQCuOP9WeTpHKpyyFSo/Q6M0NDCBKOvnnkPq15udRiV6EsyXa5lDxb+beSdukMSZ7s4CMCcIfnOZwQkXyBtPAT5sFPQdyBl8iMDZv7VBM/3l99RKBeOVSGKbqtVU6aIAik539A"
	
	data := map[string]interface{}{
		"refreshToken": refreshToken,
		"displayName":  "测试账号",
		"notes":        "通过 API 测试添加",
	}
	
	fmt.Println("=== 测试添加 Kiro 账号 ===")
	fmt.Println("1. 调用 AddKiroAccount...")
	
	err := app.AddKiroAccount("token", data)
	if err != nil {
		t.Fatalf("添加账号失败: %v", err)
	}
	
	fmt.Println("   ✓ 添加成功")
	
	// 获取账号列表
	fmt.Println("\n2. 获取账号列表...")
	accounts, err := app.GetKiroAccounts()
	if err != nil {
		t.Fatalf("获取账号列表失败: %v", err)
	}
	
	fmt.Printf("   账号数量: %d\n", len(accounts))
	
	if len(accounts) == 0 {
		t.Fatal("账号列表为空！")
	}
	
	// 显示账号信息
	for i, acc := range accounts {
		fmt.Printf("\n   账号 %d:\n", i+1)
		fmt.Printf("     ID: %s\n", acc.ID)
		fmt.Printf("     邮箱: %s\n", acc.Email)
		fmt.Printf("     显示名称: %s\n", acc.DisplayName)
		fmt.Printf("     订阅类型: %s\n", acc.SubscriptionType)
		fmt.Printf("     主配额: %d/%d\n", acc.Quota.Main.Used, acc.Quota.Main.Total)
		fmt.Printf("     试用配额: %d/%d\n", acc.Quota.Trial.Used, acc.Quota.Trial.Total)
		fmt.Printf("     激活状态: %v\n", acc.IsActive)
	}
	
	fmt.Println("\n=== 测试完成 ===")
}
