package main

import (
	"log"
	"net/http"
	"time"
)

// handleWebSocket 處理 WebSocket 連接請求
//
// Responsible for:
// - 升級 HTTP 連接為 WebSocket
// - 驗證客戶端提供的帳號憑證
// - 建立新的客戶端連接並註冊到 Hub
//
// Design considerations:
// - 在連接升級前進行帳號驗證以提高安全性
// - 驗證失敗時發送錯誤訊息並關閉連接
// - 成功連接後啟動讀寫 goroutines 處理訊息
//
// Process flow:
// 1. 升級 HTTP 連接為 WebSocket
// 2. 從查詢參數獲取用戶名和密碼
// 3. 驗證帳號憑證是否有效
// 4. 建立 Client 實例並設置相關資訊
// 5. 註冊客戶端到 Hub 進行管理
// 6. 啟動 readPump 和 writePump goroutines
//
// Usage context:
// - 客戶端建立 WebSocket 連接時調用
// - 路由器將 /ws 端點對應到此處理器
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := WebSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf(LogWebSocketUpgradeError, err)
		return
	}

	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	// 驗證帳號
	account, valid := validateAccount(username, password)
	if !valid {
		log.Printf(LogInvalidAccount, username)
		conn.WriteJSON(map[string]string{
			"error": ErrorInvalidAuth,
		})
		conn.Close()
		return
	}

	client := &Client{
		conn:     conn,
		send:     make(chan Message, DefaultClientSendBuffer),
		username: account.Username,
		channel:  account.Channel,
	}

	hub.register <- client

	go client.writePump()
	go client.readPump()
}

// readPump 處理客戶端發送的訊息
//
// Responsible for:
// - 持續讀取客戶端發送的 WebSocket 訊息
// - 設置適當的讀取限制和超時時間
// - 處理連接斷開和清理工作
//
// Design considerations:
// - 使用 defer 確保連接斷開時進行清理
// - 設置讀取限制防止過大訊息攻擊
// - 設置超時和 Pong 處理器維持連接活性
//
// Process flow:
// 1. 設置連接參數（讀取限制、超時、Pong 處理器）
// 2. 進入無限迴圈讀取訊息
// 3. 解析 JSON 格式的訊息
// 4. 設置訊息屬性（ID、時間戳、用戶、頻道）
// 5. 存儲訊息到對應頻道
// 6. 廣播訊息給其他客戶端
// 7. 發生錯誤時退出迴圈並清理連接
//
// Usage context:
// - 客戶端連接建立後在獨立 goroutine 中運行
func (c *Client) readPump() {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(DefaultReadLimit)
	c.conn.SetReadDeadline(time.Now().Add(DefaultReadTimeout * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(DefaultReadTimeout * time.Second))
		return nil
	})

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			log.Printf(LogReadJSONError, err)
			break
		}

		msg.ID = generateMessageID()
		msg.Timestamp = time.Now()
		msg.User = c.username
		msg.Channel = c.channel

		// 儲存訊息到對應 channel 的 messageStore
		messageStore.AddMessage(msg)

		// 廣播訊息到所有客戶端
		hub.broadcast <- msg
	}
}

// writePump 處理發送給客戶端的訊息
//
// Responsible for:
// - 持續監聽客戶端的發送佇列
// - 將訊息序列化為 JSON 並發送給客戶端
// - 處理發送失敗和連接清理
//
// Design considerations:
// - 使用 select 語句監聽 send channel
// - 發送失敗時直接返回，讓 Hub 處理客戶端移除
// - 使用 defer 確保連接正確關閉
//
// Process flow:
// 1. 進入無限迴圈監聽 send channel
// 2. 收到訊息時序列化為 JSON 並發送
// 3. 發送失敗時記錄錯誤並退出
// 4. 退出時關閉 WebSocket 連接
//
// Usage context:
// - 客戶端連接建立後在獨立 goroutine 中運行
// - Hub 廣播訊息時透過 send channel 發送
func (c *Client) writePump() {
	defer c.conn.Close()

	for message := range c.send {
		if err := c.conn.WriteJSON(message); err != nil {
			log.Printf(LogWriteJSONError, err)
			return
		}
	}
}

// Hub.run 運行 WebSocket 連接管理中心
//
// Responsible for:
// - 管理所有客戶端連接的生命週期
// - 處理客戶端註冊、取消註冊和訊息廣播
// - 維護連接狀態和發送系統通知
//
// Design considerations:
// - 使用 select 語句處理多個 channel 的事件
// - 註冊和取消註冊時發送系統通知
// - 廣播時只發送給相同頻道的客戶端
// - 發送失敗時自動清理斷開的連接
//
// Process flow:
// 1. 進入無限迴圈監聽三個主要 channel
// 2. 處理客戶端註冊：加入 clients map，發送歡迎訊息
// 3. 處理客戶端取消註冊：移除並發送離開訊息
// 4. 處理訊息廣播：只發送給相同頻道的客戶端
// 5. 發送失敗時自動清理斷開的客戶端
//
// Usage context:
// - 程式啟動時在獨立 goroutine 中運行
// - 整個程式生命週期中持續運行
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf(LogUserConnected, client.username, client.channel)

			// 發送歡迎消息
			welcomeMsg := NewJoinMessage(client.username, client.channel)

			// 儲存系統訊息到對應 channel
			messageStore.AddMessage(welcomeMsg)
			h.broadcast <- welcomeMsg

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf(LogUserDisconnected, client.username, client.channel)

				// 發送離線消息
				leaveMsg := NewLeaveMessage(client.username, client.channel)
				// 儲存系統訊息到對應 channel
				messageStore.AddMessage(leaveMsg)
				h.broadcast <- leaveMsg
			}

		case message := <-h.broadcast:
			// 只廣播給相同 channel 的客戶端
			log.Printf(LogBroadcastToChannel, message.Channel, message.User, message.Content)
			broadcastCount := 0
			for client := range h.clients {
				if client.channel == message.Channel {
					select {
					case client.send <- message:
						broadcastCount++
						log.Printf(LogMessageSentToUser, client.username, client.channel)
					default:
						close(client.send)
						delete(h.clients, client)
						log.Printf(LogClientRemoved, client.username)
					}
				}
			}
			log.Printf(LogBroadcastComplete, broadcastCount)
		}
	}
}
