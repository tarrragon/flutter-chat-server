package main

import (
	"crypto/rand"
	"fmt"
	"sync/atomic"
	"time"
)

// Message 代表聊天訊息
//
// Responsible for:
// - 存儲單條聊天訊息的完整資訊
// - 支援不同類型的訊息（文字、系統通知等）
// - 提供時間戳記和頻道分類
//
// Design considerations:
// - ID 使用字串格式以支援複合 ID（時間戳 + 毫秒）
// - Type 欄位預留擴展性，支援未來的多媒體訊息
// - Channel 欄位確保訊息的頻道隔離
//
// Usage context:
// - WebSocket 接收訊息時建立
// - REST API 接收訊息時建立
// - 載入歷史訊息時從存儲中讀取
type Message struct {
	ID        string    `json:"id"`        // 訊息唯一識別碼
	User      string    `json:"user"`      // 發送者用戶名
	Content   string    `json:"content"`   // 訊息內容
	Timestamp time.Time `json:"timestamp"` // 發送時間
	Type      string    `json:"type"`      // 訊息類型（text, system, image, file）
	Channel   string    `json:"channel"`   // 所屬頻道
}

// messageIDCounter 用於生成唯一 ID 的計數器
var messageIDCounter int64

// generateMessageID 生成唯一的訊息 ID
//
// Responsible for:
// - 生成全域唯一的訊息識別碼
// - 結合時間戳、原子計數器和隨機數確保唯一性
// - 即使在高併發情況下也能保證 ID 不重複
//
// Returns:
// - string: 唯一的訊息 ID
func generateMessageID() string {
	timestamp := time.Now().UnixNano()
	counter := atomic.AddInt64(&messageIDCounter, 1)
	
	// 添加一個小的隨機數增加唯一性
	randomBytes := make([]byte, 2)
	rand.Read(randomBytes)
	randomNum := int64(randomBytes[0])<<8 | int64(randomBytes[1])
	
	return fmt.Sprintf("%d_%d_%d", timestamp, counter, randomNum)
}

// NewMessage 建立新的一般訊息
//
// Responsible for:
// - 建立具有基本屬性的文字訊息
// - 自動設置 ID 和時間戳
// - 設置預設的訊息類型為文字
//
// Parameters:
// - user: 發送者用戶名
// - content: 訊息內容
// - channel: 所屬頻道
//
// Returns:
// - Message: 完整的訊息物件
func NewMessage(user, content, channel string) Message {
	now := time.Now()
	return Message{
		ID:        generateMessageID(),
		User:      user,
		Content:   content,
		Timestamp: now,
		Type:      MessageTypeText,
		Channel:   channel,
	}
}

// NewSystemMessage 建立系統訊息
//
// Responsible for:
// - 建立系統通知類型的訊息
// - 自動設置系統用戶名和訊息類型
// - 用於用戶加入/離開等系統事件
//
// Parameters:
// - content: 系統訊息內容
// - channel: 所屬頻道
//
// Returns:
// - Message: 系統訊息物件
func NewSystemMessage(content, channel string) Message {
	now := time.Now()
	return Message{
		ID:        generateMessageID(),
		User:      "System",
		Content:   content,
		Timestamp: now,
		Type:      MessageTypeSystem,
		Channel:   channel,
	}
}

// NewJoinMessage 建立用戶加入頻道的系統訊息
//
// Responsible for:
// - 建立標準格式的用戶加入通知
// - 使用統一的訊息模板
//
// Parameters:
// - username: 加入的用戶名
// - channel: 頻道名稱
//
// Returns:
// - Message: 加入通知訊息
func NewJoinMessage(username, channel string) Message {
	content := fmt.Sprintf(SystemMessageJoinTemplate, username, channel)
	return NewSystemMessage(content, channel)
}

// NewLeaveMessage 建立用戶離開頻道的系統訊息
//
// Responsible for:
// - 建立標準格式的用戶離開通知
// - 使用統一的訊息模板
//
// Parameters:
// - username: 離開的用戶名
// - channel: 頻道名稱
//
// Returns:
// - Message: 離開通知訊息
func NewLeaveMessage(username, channel string) Message {
	content := fmt.Sprintf(SystemMessageLeaveTemplate, username, channel)
	return NewSystemMessage(content, channel)
}

// NewWelcomeMessage 建立頻道歡迎訊息
//
// Responsible for:
// - 建立頻道的歡迎訊息
// - 用於新用戶首次進入或歷史訊息為空時顯示
//
// Parameters:
// - channel: 頻道名稱
//
// Returns:
// - Message: 歡迎訊息
func NewWelcomeMessage(channel string) Message {
	content := fmt.Sprintf(WelcomeMessageTemplate, channel)
	return NewSystemMessage(content, channel)
}

// IsSystemMessage 檢查是否為系統訊息
//
// Returns:
// - bool: true 如果是系統訊息
func (m Message) IsSystemMessage() bool {
	return m.Type == MessageTypeSystem
}

// IsFromUser 檢查訊息是否來自指定用戶
//
// Parameters:
// - username: 要檢查的用戶名
//
// Returns:
// - bool: true 如果訊息來自指定用戶
func (m Message) IsFromUser(username string) bool {
	return m.User == username
}

// BelongsToChannel 檢查訊息是否屬於指定頻道
//
// Parameters:
// - channel: 要檢查的頻道名
//
// Returns:
// - bool: true 如果訊息屬於指定頻道
func (m Message) BelongsToChannel(channel string) bool {
	return m.Channel == channel
}

// MessageStore 提供訊息存儲的便利方法
type MessageStore map[string][]Message

// AddMessage 將訊息添加到指定頻道
//
// Responsible for:
// - 將訊息存儲到對應頻道
// - 自動初始化頻道存儲（如果不存在）
//
// Parameters:
// - message: 要存儲的訊息
func (ms MessageStore) AddMessage(message Message) {
	if ms[message.Channel] == nil {
		ms[message.Channel] = []Message{}
	}
	ms[message.Channel] = append(ms[message.Channel], message)
}

// GetRecentMessages 獲取頻道的最近訊息
//
// Responsible for:
// - 獲取指定頻道的最近 N 條訊息
// - 如果頻道無訊息則返回歡迎訊息
//
// Parameters:
// - channel: 頻道名稱
// - limit: 最多返回的訊息數量
//
// Returns:
// - []Message: 最近的訊息列表
func (ms MessageStore) GetRecentMessages(channel string, limit int) []Message {
	channelMessages := ms[channel]
	
	// 如果沒有訊息，返回歡迎訊息
	if len(channelMessages) == 0 {
		return []Message{NewWelcomeMessage(channel)}
	}
	
	// 計算起始位置
	start := 0
	if len(channelMessages) > limit {
		start = len(channelMessages) - limit
	}
	
	return channelMessages[start:]
}

// GetChannelMessageCount 獲取頻道的訊息總數
//
// Parameters:
// - channel: 頻道名稱
//
// Returns:
// - int: 訊息總數
func (ms MessageStore) GetChannelMessageCount(channel string) int {
	return len(ms[channel])
}

// Clear 清空所有訊息
func (ms MessageStore) Clear() {
	for channel := range ms {
		delete(ms, channel)
	}
}

// ClearChannel 清空指定頻道的訊息
//
// Parameters:
// - channel: 要清空的頻道名稱
func (ms MessageStore) ClearChannel(channel string) {
	delete(ms, channel)
}