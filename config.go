package main

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// 應用程式設定常數 - 單一來源，集中管理
//
// Responsible for:
// - 定義所有應用程式設定的單一來源
// - 確保設定在整個應用程式中保持一致
// - 遵循 DRY 原則，避免重複定義
//
// Design considerations:
// - 所有設定集中在這裡定義
// - 其他模組直接引用這些常數
// - 簡潔明瞭，易於維護
//
// Usage context:
// - 各模組直接使用這些常數
// - 測試中可以使用這些常數進行驗證

const (
	// 網路設定預設值
	DefaultServerPort = 8080
	DefaultServerHost = "localhost"

	// WebSocket 設定預設值
	DefaultReadLimit       = 512
	DefaultReadTimeout     = 60
	DefaultWriteTimeout    = 10
	DefaultPongWait        = 60
	DefaultPingPeriod      = 54
	DefaultAllowAllOrigins = true

	// 訊息處理設定預設值
	DefaultHistoryLimit       = 50
	DefaultClientSendBuffer   = 256
	DefaultHubBroadcastBuffer = 256

	// 預設使用者和類型
	DefaultUsername    = "Anonymous"
	DefaultAPIUser     = "Web User"
	DefaultMessageType = "text"

	// 訊息類型
	MessageTypeText   = "text"
	MessageTypeSystem = "system"
	MessageTypeImage  = "image"
	MessageTypeFile   = "file"

	// HTTP 回應訊息
	StatusSent = "sent"

	// 錯誤訊息
	ErrorInvalidJSON     = "Invalid JSON"
	ErrorChannelRequired = "channel is required"
	ErrorInvalidAuth     = "Invalid username or password"

	// 系統訊息模板
	SystemMessageJoinTemplate  = "%s 加入了 %s 頻道"
	SystemMessageLeaveTemplate = "%s 離開了 %s 頻道"
	WelcomeMessageTemplate     = "歡迎來到 %s 頻道！開始你的第一條消息吧 👋"

	// 日誌訊息模板
	LogWebSocketUpgradeError = "WebSocket upgrade error: %v"
	LogInvalidAccount        = "Invalid account: %s"
	LogUserConnected         = "User %s connected to channel %s"
	LogUserDisconnected      = "User %s disconnected from channel %s"
	LogReadJSONError         = "ReadJSON error: %v"
	LogWriteJSONError        = "WriteJSON error: %v"
	LogClientRemoved         = "客戶端 %s 發送失敗，已移除"

	LogAPIMessageReceived = "收到 GET /api/messages 請求，channel: %s"
	LogAPIMessagePost     = "收到 POST /api/messages 請求"
	LogMessageStored      = "訊息已儲存到 channel %s，該頻道目前共有 %d 條訊息"
	LogResponseSent       = "已回應客戶端"
	LogBroadcastStart     = "開始廣播訊息到 WebSocket 客戶端"
	LogBroadcastSuccess   = "訊息已廣播到 WebSocket 客戶端"
	LogBroadcastComplete  = "廣播完成，共發送給 %d 個客戶端"
	LogBroadcastToChannel = "廣播訊息到頻道 %s: %s 說 '%s'"
	LogMessageSentToUser  = "訊息已發送給用戶 %s (頻道: %s)"
)

// 預設測試帳號
var DefaultTestAccounts = []Account{
	{Username: "alice", Password: "password123", Channel: "general"},
	{Username: "bob", Password: "password123", Channel: "tech"},
	{Username: "charlie", Password: "password123", Channel: "random"},
}

// 啟動訊息模板
const DefaultStartupBanner = `🚀 服務器啟動在 http://%s:%d
📱 手機端可連接: http://你的內網IP:%d
💻 WebSocket 端點: ws://%s:%d/ws?username=帳號&password=密碼
📡 API 端點:
   GET  /api/messages?channel=頻道 - 獲取指定頻道的歷史消息
   POST /api/messages - 發送消息
   GET  /api/users - 獲取按頻道分組的在線用戶
   GET  /api/accounts - 獲取可用的測試帳號
   POST /api/login - 驗證帳號登入

🧪 測試帳號:`

const DefaultAccountInfoTemplate = "   用戶: %s, 密碼: %s, 頻道: %s"

// WebSocket 升級器設定
var WebSocketUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允許跨域，測試用
	},
}
