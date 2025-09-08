package main

import (
	"github.com/gorilla/websocket"
)

// Account 代表測試帳號
//
// Responsible for:
// - 存儲帳號的基本資訊
// - 關聯帳號與頻道的對應關係
//
// Design considerations:
// - 使用結構體標籤支援 JSON 序列化
// - 密碼欄位包含在結構體中，但在 API 回應時會被過濾
//
// Usage context:
// - 帳號驗證時比對用戶輸入
// - API 回應時提供帳號資訊（不含密碼）
type Account struct {
	Username string `json:"username"` // 用戶名稱
	Password string `json:"password"` // 登入密碼
	Channel  string `json:"channel"`  // 所屬頻道
}


// Client 代表 WebSocket 客戶端連接
//
// Responsible for:
// - 管理單個 WebSocket 連接的狀態
// - 處理客戶端的訊息發送佇列
// - 關聯客戶端與用戶帳號和頻道
//
// Design considerations:
// - send channel 使用緩衝區避免阻塞
// - 包含用戶名和頻道資訊便於訊息路由
// - 直接持有 WebSocket 連接的引用
//
// Process flow:
// 1. WebSocket 連接建立時創建 Client 實例
// 2. 註冊到 Hub 進行集中管理
// 3. 啟動 readPump 和 writePump goroutines
// 4. 連接斷開時從 Hub 取消註冊並清理資源
//
// Usage context:
// - WebSocket 連接處理器建立客戶端
// - Hub 管理所有活躍客戶端
// - 訊息廣播時遍歷相關客戶端
type Client struct {
	conn     *websocket.Conn // WebSocket 連接
	send     chan Message    // 訊息發送佇列
	username string          // 用戶名稱
	channel  string          // 所屬頻道
}

// Hub 管理所有 WebSocket 連接
//
// Responsible for:
// - 集中管理所有活躍的客戶端連接
// - 處理客戶端的註冊和取消註冊
// - 廣播訊息給指定頻道的客戶端
// - 維護連接狀態和生命週期
//
// Design considerations:
// - 使用 channels 進行 goroutine 間的安全通訊
// - broadcast channel 使用緩衝區提高效能
// - clients map 用於快速查找和管理連接
//
// Process flow:
// 1. 啟動時開始運行事件迴圈
// 2. 監聽 register、unregister、broadcast 三個 channel
// 3. 註冊時將客戶端加入 clients map 並發送歡迎訊息
// 4. 取消註冊時移除客戶端並發送離開訊息
// 5. 廣播時只發送給相同頻道的客戶端
//
// Usage context:
// - 程式啟動時在獨立 goroutine 中運行
// - WebSocket 連接建立/斷開時進行註冊操作
// - 收到新訊息時進行廣播
type Hub struct {
	clients    map[*Client]bool // 已註冊的客戶端
	broadcast  chan Message     // 廣播訊息佇列
	register   chan *Client     // 客戶端註冊佇列
	unregister chan *Client     // 客戶端取消註冊佇列
}
