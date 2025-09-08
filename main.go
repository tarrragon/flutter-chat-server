package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// 預設測試帳號
type Account struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Channel  string `json:"channel"`
}

// 預設的三個測試帳號
var testAccounts = []Account{
	{Username: "alice", Password: "password123", Channel: "general"},
	{Username: "bob", Password: "password123", Channel: "tech"},
	{Username: "charlie", Password: "password123", Channel: "random"},
}

// 消息結構
type Message struct {
	ID        string    `json:"id"`
	User      string    `json:"user"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // text, image, file etc.
	Channel   string    `json:"channel"`
}

// 用戶連接管理
type Client struct {
	conn     *websocket.Conn
	send     chan Message
	username string
	channel  string
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
}

// 按 channel 分類存儲訊息
var messageStore = make(map[string][]Message)

// 驗證帳號函式
func validateAccount(username, password string) (*Account, bool) {
	for _, account := range testAccounts {
		if account.Username == username && account.Password == password {
			return &account, true
		}
	}
	return nil, false
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允許跨域，測試用
	},
}

var hub = Hub{
	clients:    make(map[*Client]bool),
	broadcast:  make(chan Message, 256), // 增加緩衝區
	register:   make(chan *Client),
	unregister: make(chan *Client),
}

// 處理 WebSocket 連接
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	// 驗證帳號
	account, valid := validateAccount(username, password)
	if !valid {
		log.Printf("Invalid account: %s", username)
		conn.WriteJSON(map[string]string{
			"error": "Invalid username or password",
		})
		conn.Close()
		return
	}

	client := &Client{
		conn:     conn,
		send:     make(chan Message, 256),
		username: account.Username,
		channel:  account.Channel,
	}

	hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("ReadJSON error: %v", err)
			break
		}

		msg.ID = fmt.Sprintf("%d_%d", time.Now().Unix(), time.Now().UnixNano())
		msg.Timestamp = time.Now()
		msg.User = c.username
		msg.Channel = c.channel

		// 儲存訊息到對應 channel 的 messageStore
		if messageStore[c.channel] == nil {
			messageStore[c.channel] = []Message{}
		}
		messageStore[c.channel] = append(messageStore[c.channel], msg)

		// 廣播訊息到所有客戶端
		hub.broadcast <- msg
	}

}

func (c *Client) writePump() {
	defer c.conn.Close()

	for {
		select {
		case message := <-c.send:
			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("WriteJSON error: %v", err)
				return
			}
		}
	}
}

// Hub 運行
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("User %s connected to channel %s", client.username, client.channel)

			// 發送歡迎消息
			welcomeMsg := Message{
				ID:        fmt.Sprintf("%d", time.Now().Unix()),
				User:      "System",
				Content:   fmt.Sprintf("%s 加入了 %s 頻道", client.username, client.channel),
				Timestamp: time.Now(),
				Type:      "system",
				Channel:   client.channel,
			}

			// 儲存系統訊息到對應 channel
			if messageStore[client.channel] == nil {
				messageStore[client.channel] = []Message{}
			}
			messageStore[client.channel] = append(messageStore[client.channel], welcomeMsg)
			h.broadcast <- welcomeMsg

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("User %s disconnected from channel %s", client.username, client.channel)

				// 發送離線消息
				leaveMsg := Message{
					ID:        fmt.Sprintf("%d", time.Now().Unix()),
					User:      "System",
					Content:   fmt.Sprintf("%s 離開了 %s 頻道", client.username, client.channel),
					Timestamp: time.Now(),
					Type:      "system",
					Channel:   client.channel,
				}
				// 儲存系統訊息到對應 channel
				if messageStore[client.channel] == nil {
					messageStore[client.channel] = []Message{}
				}
				messageStore[client.channel] = append(messageStore[client.channel], leaveMsg)
				h.broadcast <- leaveMsg
			}

		case message := <-h.broadcast:
			// 只廣播給相同 channel 的客戶端
			log.Printf("廣播訊息到頻道 %s: %s 說 '%s'", message.Channel, message.User, message.Content)
			broadcastCount := 0
			for client := range h.clients {
				if client.channel == message.Channel {
					select {
					case client.send <- message:
						broadcastCount++
						log.Printf("訊息已發送給用戶 %s (頻道: %s)", client.username, client.channel)
					default:
						close(client.send)
						delete(h.clients, client)
						log.Printf("客戶端 %s 發送失敗，已移除", client.username)
					}
				}
			}
			log.Printf("廣播完成，共發送給 %d 個客戶端", broadcastCount)
		}
	}
}

// REST API 端點
func getMessages(w http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("channel")
	log.Printf("收到 GET /api/messages 請求，channel: %s", channel)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 如果沒有指定 channel，返回錯誤
	if channel == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "channel parameter is required"})
		return
	}

	// 返回指定 channel 的消息，如果沒有則返回歡迎消息
	channelMessages := messageStore[channel]
	if len(channelMessages) == 0 {
		log.Printf("channel %s 的 messageStore 為空，返回歡迎訊息", channel)
		welcomeMessages := []Message{
			{
				ID:        "welcome",
				User:      "System",
				Content:   fmt.Sprintf("歡迎來到 %s 頻道！開始你的第一條消息吧 👋", channel),
				Timestamp: time.Now(),
				Type:      "system",
				Channel:   channel,
			},
		}
		json.NewEncoder(w).Encode(welcomeMessages)
		return
	}

	// 返回最近的消息（限制數量避免一次返回太多）
	limit := 50
	start := 0
	if len(channelMessages) > limit {
		start = len(channelMessages) - limit
	}

	recentMessages := channelMessages[start:]
	log.Printf("返回 channel %s 的 %d 條訊息 (總共 %d 條)", channel, len(recentMessages), len(channelMessages))
	json.NewEncoder(w).Encode(recentMessages)
}

func sendMessage(w http.ResponseWriter, r *http.Request) {
	log.Printf("收到 POST /api/messages 請求")

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
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	// 檢查是否有指定 channel
	if msg.Channel == "" {
		log.Printf("缺少 channel 參數")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "channel is required"})
		return
	}

	log.Printf("解析到訊息: %+v", msg)

	msg.ID = fmt.Sprintf("%d_%d", time.Now().Unix(), time.Now().UnixNano())
	msg.Timestamp = time.Now()

	// 如果沒有指定用戶名，設定為預設值
	if msg.User == "" {
		msg.User = "Web User"
	}

	// 儲存訊息到對應 channel 的 messageStore
	if messageStore[msg.Channel] == nil {
		messageStore[msg.Channel] = []Message{}
	}
	messageStore[msg.Channel] = append(messageStore[msg.Channel], msg)
	log.Printf("訊息已儲存到 channel %s，該頻道目前共有 %d 條訊息", msg.Channel, len(messageStore[msg.Channel]))

	// 先回應客戶端
	log.Printf("準備回應客戶端")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "sent"}); err != nil {
		log.Printf("回應編碼錯誤: %v", err)
		return
	}
	log.Printf("已回應客戶端")

	// 然後廣播到所有 WebSocket 客戶端（使用 goroutine 避免阻塞）
	go func() {
		log.Printf("開始廣播訊息到 WebSocket 客戶端")
		hub.broadcast <- msg
		log.Printf("訊息已廣播到 WebSocket 客戶端")
	}()
}

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

// 獲取可用的測試帳號
func getAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 只返回公開資訊，不包含密碼
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

// 驗證帳號登入
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
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	account, valid := validateAccount(loginData.Username, loginData.Password)
	if !valid {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid username or password"})
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

func main() {
	// 啟動 Hub
	go hub.run()

	r := mux.NewRouter()

	// REST API 路由
	r.HandleFunc("/api/messages", getMessages).Methods("GET")
	r.HandleFunc("/api/messages", sendMessage).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/users", getOnlineUsers).Methods("GET")
	r.HandleFunc("/api/accounts", getAccounts).Methods("GET")
	r.HandleFunc("/api/login", loginAccount).Methods("POST", "OPTIONS")

	// WebSocket 路由
	r.HandleFunc("/ws", handleWebSocket)

	// 靜態文件服務（可選，用於測試前端）
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	fmt.Println("🚀 服務器啟動在 http://localhost:8080")
	fmt.Println("📱 手機端可連接: http://你的內網IP:8080")
	fmt.Println("💻 WebSocket 端點: ws://localhost:8080/ws?username=帳號&password=密碼")
	fmt.Println("📡 API 端點:")
	fmt.Println("   GET  /api/messages?channel=頻道 - 獲取指定頻道的歷史消息")
	fmt.Println("   POST /api/messages - 發送消息")
	fmt.Println("   GET  /api/users - 獲取按頻道分組的在線用戶")
	fmt.Println("   GET  /api/accounts - 獲取可用的測試帳號")
	fmt.Println("   POST /api/login - 驗證帳號登入")
	fmt.Println()
	fmt.Println("🧪 測試帳號:")
	for _, account := range testAccounts {
		fmt.Printf("   用戶: %s, 密碼: %s, 頻道: %s\n", account.Username, account.Password, account.Channel)
	}

	log.Fatal(http.ListenAndServe(":8080", r))
}
