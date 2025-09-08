package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// getMessages 處理獲取歷史訊息的 API 請求
//
// Responsible for:
// - 處理 GET /api/messages 的 HTTP 請求
// - 根據 channel 參數返回對應頻道的歷史訊息
// - 提供預設歡迎訊息當頻道為空時
//
// Design considerations:
// - 要求必須提供 channel 參數以確保頻道隔離
// - 限制返回訊息數量避免一次載入過多資料
// - 空頻道時提供友好的歡迎訊息
//
// Process flow:
// 1. 設置 CORS 標頭支援跨域請求
// 2. 檢查 channel 參數是否存在
// 3. 獲取指定頻道的訊息列表
// 4. 如果頻道為空則返回歡迎訊息
// 5. 限制返回最近的訊息數量
// 6. 序列化為 JSON 並返回
//
// Usage context:
// - 客戶端載入聊天歷史時調用
// - 支援分頁載入以提高效能
func getMessages(w http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("channel")
	log.Print("收到 GET /api/messages 請求，channel: " + channel)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 如果沒有指定 channel，返回錯誤
	if channel == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": ErrorChannelRequired})
		return
	}

	// 使用新的便利方法獲取最近訊息
	recentMessages := messageStore.GetRecentMessages(channel, DefaultHistoryLimit)
	log.Printf("返回 channel %s 的 %d 條訊息 (總共 %d 條)", channel, len(recentMessages), messageStore.GetChannelMessageCount(channel))
	json.NewEncoder(w).Encode(recentMessages)
}

// sendMessage 處理發送新訊息的 API 請求
//
// Responsible for:
// - 處理 POST /api/messages 的 HTTP 請求
// - 解析 JSON 格式的訊息內容
// - 驗證必要欄位並存儲訊息
// - 廣播訊息給 WebSocket 客戶端
//
// Design considerations:
// - 支援 CORS 和 OPTIONS 預檢請求
// - 要求必須指定 channel 參數
// - 自動設置訊息 ID、時間戳等系統欄位
// - 使用 goroutine 進行異步廣播避免阻塞回應
//
// Process flow:
// 1. 設置 CORS 標頭並處理 OPTIONS 請求
// 2. 解析 JSON 請求主體為 Message 結構
// 3. 驗證必要的 channel 欄位
// 4. 設置系統生成的欄位（ID、時間戳、用戶）
// 5. 存儲訊息到對應頻道
// 6. 立即回應客戶端表示成功
// 7. 異步廣播訊息給 WebSocket 客戶端
//
// Usage context:
// - 客戶端透過 REST API 發送訊息時調用
// - 支援非 WebSocket 客戶端的訊息發送
func sendMessage(w http.ResponseWriter, r *http.Request) {
	log.Print("收到 POST /api/messages 請求")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		log.Printf("處理 OPTIONS 請求")
		w.WriteHeader(http.StatusOK)
		return
	}

	var msg Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		log.Printf("JSON 解析錯誤: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": ErrorInvalidJSON})
		return
	}

	// 檢查是否有指定 channel
	if msg.Channel == "" {
		log.Printf("缺少 channel 參數")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": ErrorChannelRequired})
		return
	}

	log.Printf("解析到訊息: %+v", msg)

	msg.ID = generateMessageID()
	msg.Timestamp = time.Now()

	// 如果沒有指定用戶名，設定為預設值
	if msg.User == "" {
		msg.User = DefaultAPIUser
	}

	// 儲存訊息到對應 channel 的 messageStore
	messageStore.AddMessage(msg)
	log.Printf("訊息已儲存到 channel %s，該頻道目前共有 %d 條訊息", msg.Channel, messageStore.GetChannelMessageCount(msg.Channel))

	// 先回應客戶端
	log.Printf("準備回應客戶端")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": StatusSent}); err != nil {
		log.Printf("回應編碼錯誤: %v", err)
		return
	}
	log.Print("已回應客戶端")

	// 然後廣播到所有 WebSocket 客戶端（使用 goroutine 避免阻塞）
	go func() {
		log.Print("開始廣播訊息到 WebSocket 客戶端")
		hub.broadcast <- msg
		log.Print("訊息已廣播到 WebSocket 客戶端")
	}()
}

// getOnlineUsers 處理獲取在線用戶的 API 請求
//
// Responsible for:
// - 處理 GET /api/users 的 HTTP 請求
// - 統計目前在線的用戶並按頻道分組
// - 返回詳細的用戶分佈資訊
//
// Design considerations:
// - 按頻道分組顯示用戶分佈情況
// - 提供總用戶數和各頻道用戶數
// - 即時反映當前連接狀態
//
// Process flow:
// 1. 設置 CORS 標頭支援跨域請求
// 2. 遍歷 Hub 中的所有客戶端連接
// 3. 按頻道將用戶名稱分組
// 4. 統計總用戶數和各頻道用戶數
// 5. 序列化為 JSON 並返回
//
// Usage context:
// - 客戶端查看在線用戶狀態時調用
// - 監控系統瞭解用戶分佈情況
func getOnlineUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 按 channel 分組用戶
	channelUsers := make(map[string][]string)
	totalCount := 0

	for client := range hub.clients {
		if channelUsers[client.channel] == nil {
			channelUsers[client.channel] = []string{}
		}
		channelUsers[client.channel] = append(channelUsers[client.channel], client.username)
		totalCount++
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"channelUsers": channelUsers,
		"totalCount":   totalCount,
	})
}

// getAccounts 處理獲取可用帳號的 API 請求
//
// Responsible for:
// - 處理 GET /api/accounts 的 HTTP 請求
// - 返回所有可用的測試帳號資訊
// - 過濾敏感資訊（如密碼）
//
// Design considerations:
// - 不包含密碼等敏感資訊在回應中
// - 提供足夠資訊供客戶端顯示選項
// - 保持 API 回應結構的一致性
//
// Process flow:
// 1. 設置 CORS 標頭支援跨域請求
// 2. 遍歷所有預設測試帳號
// 3. 建立只包含公開資訊的帳號列表
// 4. 序列化為 JSON 並返回
//
// Usage context:
// - 客戶端登入頁面載入可用帳號選項
// - 提供用戶選擇和瞭解可用帳號
func getAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 只返回公開資訊，不包含密碼
	testAccounts := getTestAccounts()
	publicAccounts := make([]map[string]string, len(testAccounts))
	for i, account := range testAccounts {
		publicAccounts[i] = map[string]string{
			"username": account.Username,
			"channel":  account.Channel,
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"accounts": publicAccounts,
	})
}

// loginAccount 處理帳號登入驗證的 API 請求
//
// Responsible for:
// - 處理 POST /api/login 的 HTTP 請求
// - 驗證用戶提供的帳號憑證
// - 返回驗證結果和帳號資訊
//
// Design considerations:
// - 支援 CORS 和 OPTIONS 預檢請求
// - 驗證失敗時返回適當的 HTTP 狀態碼
// - 成功時返回帳號資訊但不包含密碼
//
// Process flow:
// 1. 設置 CORS 標頭並處理 OPTIONS 請求
// 2. 解析 JSON 請求主體獲取帳號憑證
// 3. 調用帳號驗證函式檢查憑證
// 4. 根據驗證結果返回對應的回應
// 5. 成功時包含帳號資訊，失敗時包含錯誤訊息
//
// Usage context:
// - 客戶端登入頁面驗證用戶憑證
// - 提供統一的帳號驗證入口
func loginAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var loginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": ErrorInvalidJSON})
		return
	}

	account, valid := validateAccount(loginData.Username, loginData.Password)
	if !valid {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": ErrorInvalidAuth})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"account": map[string]string{
			"username": account.Username,
			"channel":  account.Channel,
		},
	})
}
