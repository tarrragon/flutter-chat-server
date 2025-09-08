package main

import (
	"fmt"
	"log"
	"net/http"
)

// 全域變數
//
// Responsible for:
// - 維護應用程式的核心狀態
// - 提供各模組間的資料共享
//
// Design considerations:
// - messageStore 使用 map 結構按頻道分類存儲
// - hub 作為 WebSocket 連接的中央管理器
//
// Usage context:
// - 整個應用程式生命週期中使用
// - 各模組透過這些變數進行資料存取和通訊
var (
	// messageStore 按 channel 分類存儲訊息
	messageStore = make(MessageStore)

	// hub WebSocket 連接管理中心
	hub = Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message, DefaultHubBroadcastBuffer),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
)

// printStartupBanner 顯示伺服器啟動資訊
//
// Responsible for:
// - 顯示伺服器啟動成功的資訊
// - 列出所有可用的 API 端點
// - 顯示測試帳號資訊
//
// Design considerations:
// - 使用統一的格式化字串確保輸出一致性
// - 包含所有開發者需要的關鍵資訊
// - 便於快速瞭解服務可用性
//
// Usage context:
// - 伺服器成功啟動後調用
// - 提供開發者和使用者快速參考
func printStartupBanner() {
	fmt.Printf(DefaultStartupBanner, DefaultServerHost, DefaultServerPort, DefaultServerPort, DefaultServerHost, DefaultServerPort)
	fmt.Println()

	testAccounts := getTestAccounts()
	for _, account := range testAccounts {
		fmt.Printf(DefaultAccountInfoTemplate, account.Username, account.Password, account.Channel)
		fmt.Println()
	}
}

// main 主程式入口點
//
// Responsible for:
// - 初始化應用程式的核心組件
// - 啟動 WebSocket 管理中心
// - 設置 HTTP 路由和啟動伺服器
//
// Design considerations:
// - Hub 在獨立 goroutine 中運行避免阻塞主執行緒
// - 使用設定檔中的常數確保配置一致性
// - 啟動資訊清楚顯示服務狀態
//
// Process flow:
// 1. 啟動 Hub 的事件迴圈處理 WebSocket 連接
// 2. 設置所有 HTTP 路由（API 和 WebSocket 端點）
// 3. 顯示啟動成功資訊和可用端點
// 4. 啟動 HTTP 伺服器並監聽指定埠
//
// Usage context:
// - 程式啟動時的主要入口點
// - 協調各個模組的初始化和啟動
func main() {
	// 啟動 Hub
	go hub.run()

	// 設置路由
	router := setupRoutes()

	// 顯示啟動資訊
	printStartupBanner()

	// 啟動伺服器
	serverAddr := fmt.Sprintf(":%d", DefaultServerPort)
	log.Fatal(http.ListenAndServe(serverAddr, router))
}
