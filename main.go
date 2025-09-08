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

// 消息結構
type Message struct {
	ID        string    `json:"id"`
	User      string    `json:"user"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // text, image, file etc.
}

// 用戶連接管理
type Client struct {
	conn     *websocket.Conn
	send     chan Message
	username string
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
}

var messageStore []Message

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允許跨域，測試用
	},
}

var hub = Hub{
	clients:    make(map[*Client]bool),
	broadcast:  make(chan Message),
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
	if username == "" {
		username = "Anonymous"
	}

	client := &Client{
		conn:     conn,
		send:     make(chan Message, 256),
		username: username,
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

		msg.Timestamp = time.Now()
		msg.User = c.username

		// 儲存訊息到 messageStore
		messageStore = append(messageStore, msg)

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
			log.Printf("User %s connected", client.username)

			// 發送歡迎消息
			welcomeMsg := Message{
				ID:        fmt.Sprintf("%d", time.Now().Unix()),
				User:      "System",
				Content:   fmt.Sprintf("%s 加入了聊天室", client.username),
				Timestamp: time.Now(),
				Type:      "system",
			}

			// 儲存系統訊息
			messageStore = append(messageStore, welcomeMsg)
			h.broadcast <- welcomeMsg

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("User %s disconnected", client.username)

				// 發送離線消息
				leaveMsg := Message{
					ID:        fmt.Sprintf("%d", time.Now().Unix()),
					User:      "System",
					Content:   fmt.Sprintf("%s 離開了聊天室", client.username),
					Timestamp: time.Now(),
					Type:      "system",
				}
				// 儲存系統訊息
				messageStore = append(messageStore, leaveMsg)
				h.broadcast <- leaveMsg
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

// REST API 端點
func getMessages(w http.ResponseWriter, r *http.Request) {
	log.Printf("收到 GET /api/messages 請求")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 返回實際存儲的消息，如果沒有則返回歡迎消息
	if len(messageStore) == 0 {
		log.Printf("messageStore 為空，返回歡迎訊息")
		welcomeMessages := []Message{
			{
				ID:        "welcome",
				User:      "System",
				Content:   "歡迎使用聊天室！開始你的第一條消息吧 👋",
				Timestamp: time.Now(),
				Type:      "system",
			},
		}
		json.NewEncoder(w).Encode(welcomeMessages)
		return
	}

	// 返回最近的消息（限制數量避免一次返回太多）
	limit := 50
	start := 0
	if len(messageStore) > limit {
		start = len(messageStore) - limit
	}

	recentMessages := messageStore[start:]
	log.Printf("返回 %d 條訊息 (總共 %d 條)", len(recentMessages), len(messageStore))
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

	log.Printf("解析到訊息: %+v", msg)

	msg.ID = fmt.Sprintf("%d", time.Now().Unix())
	msg.Timestamp = time.Now()
	msg.User = "API User" // 設定預設用戶名

	// 儲存訊息到 messageStore
	messageStore = append(messageStore, msg)
	log.Printf("訊息已儲存，目前共有 %d 條訊息", len(messageStore))

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
		select {
		case hub.broadcast <- msg:
			log.Printf("訊息已廣播到 WebSocket 客戶端")
		default:
			// 如果廣播 channel 滿了，記錄但不阻塞
			log.Printf("Broadcast channel full, message not sent to WebSocket clients")
		}
	}()
}

func getOnlineUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var users []string
	for client := range hub.clients {
		users = append(users, client.username)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": users,
		"count": len(users),
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

	// WebSocket 路由
	r.HandleFunc("/ws", handleWebSocket)

	// 靜態文件服務（可選，用於測試前端）
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	fmt.Println("🚀 服務器啟動在 http://localhost:8080")
	fmt.Println("📱 手機端可連接: http://你的內網IP:8080")
	fmt.Println("💻 WebSocket 端點: ws://localhost:8080/ws")
	fmt.Println("📡 API 端點:")
	fmt.Println("   GET  /api/messages - 獲取歷史消息")
	fmt.Println("   POST /api/messages - 發送消息")
	fmt.Println("   GET  /api/users    - 獲取在線用戶")

	log.Fatal(http.ListenAndServe(":8080", r))
}
