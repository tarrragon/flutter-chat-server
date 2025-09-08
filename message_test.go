package main

import (
	"testing"
	"time"
)

// TestMessageConstructors 測試訊息建構函數
func TestMessageConstructors(t *testing.T) {
	t.Run("NewMessage", func(t *testing.T) {
		msg := NewMessage("alice", "測試內容", "general")
		
		if msg.User != "alice" {
			t.Errorf("Expected user 'alice', got '%s'", msg.User)
		}
		
		if msg.Content != "測試內容" {
			t.Errorf("Expected content '測試內容', got '%s'", msg.Content)
		}
		
		if msg.Channel != "general" {
			t.Errorf("Expected channel 'general', got '%s'", msg.Channel)
		}
		
		if msg.Type != MessageTypeText {
			t.Errorf("Expected type '%s', got '%s'", MessageTypeText, msg.Type)
		}
		
		if msg.ID == "" {
			t.Error("Expected ID to be generated")
		}
		
		if msg.Timestamp.IsZero() {
			t.Error("Expected timestamp to be set")
		}
	})

	t.Run("NewSystemMessage", func(t *testing.T) {
		msg := NewSystemMessage("系統通知", "general")
		
		if msg.User != "System" {
			t.Errorf("Expected user 'System', got '%s'", msg.User)
		}
		
		if msg.Type != MessageTypeSystem {
			t.Errorf("Expected type '%s', got '%s'", MessageTypeSystem, msg.Type)
		}
	})

	t.Run("NewJoinMessage", func(t *testing.T) {
		msg := NewJoinMessage("alice", "general")
		
		expectedContent := "alice 加入了 general 頻道"
		if msg.Content != expectedContent {
			t.Errorf("Expected content '%s', got '%s'", expectedContent, msg.Content)
		}
		
		if msg.Type != MessageTypeSystem {
			t.Errorf("Expected type '%s', got '%s'", MessageTypeSystem, msg.Type)
		}
	})

	t.Run("NewLeaveMessage", func(t *testing.T) {
		msg := NewLeaveMessage("alice", "general")
		
		expectedContent := "alice 離開了 general 頻道"
		if msg.Content != expectedContent {
			t.Errorf("Expected content '%s', got '%s'", expectedContent, msg.Content)
		}
	})

	t.Run("NewWelcomeMessage", func(t *testing.T) {
		msg := NewWelcomeMessage("general")
		
		expectedContent := "歡迎來到 general 頻道！開始你的第一條消息吧 👋"
		if msg.Content != expectedContent {
			t.Errorf("Expected content '%s', got '%s'", expectedContent, msg.Content)
		}
	})
}

// TestMessageMethods 測試訊息方法
func TestMessageMethods(t *testing.T) {
	systemMsg := NewSystemMessage("系統訊息", "general")
	userMsg := NewMessage("alice", "用戶訊息", "general")

	t.Run("IsSystemMessage", func(t *testing.T) {
		if !systemMsg.IsSystemMessage() {
			t.Error("Expected system message to return true for IsSystemMessage()")
		}
		
		if userMsg.IsSystemMessage() {
			t.Error("Expected user message to return false for IsSystemMessage()")
		}
	})

	t.Run("IsFromUser", func(t *testing.T) {
		if !userMsg.IsFromUser("alice") {
			t.Error("Expected message from alice to return true")
		}
		
		if userMsg.IsFromUser("bob") {
			t.Error("Expected message not from bob to return false")
		}
	})

	t.Run("BelongsToChannel", func(t *testing.T) {
		if !userMsg.BelongsToChannel("general") {
			t.Error("Expected message to belong to general channel")
		}
		
		if userMsg.BelongsToChannel("tech") {
			t.Error("Expected message not to belong to tech channel")
		}
	})
}

// TestMessageStore 測試訊息存儲
func TestMessageStore(t *testing.T) {
	store := make(MessageStore)

	t.Run("AddMessage", func(t *testing.T) {
		msg := NewMessage("alice", "測試訊息", "general")
		store.AddMessage(msg)
		
		if len(store["general"]) != 1 {
			t.Errorf("Expected 1 message in general channel, got %d", len(store["general"]))
		}
		
		if store["general"][0].Content != "測試訊息" {
			t.Errorf("Expected message content '測試訊息', got '%s'", store["general"][0].Content)
		}
	})

	t.Run("GetRecentMessages_WithMessages", func(t *testing.T) {
		// 添加多條訊息
		for i := 0; i < 5; i++ {
			msg := NewMessage("alice", "訊息"+string(rune('1'+i)), "general")
			store.AddMessage(msg)
		}
		
		recent := store.GetRecentMessages("general", 3)
		if len(recent) != 3 {
			t.Errorf("Expected 3 recent messages, got %d", len(recent))
		}
		
		// 檢查是否返回最新的訊息
		if recent[len(recent)-1].Content != "訊息5" {
			t.Errorf("Expected last message to be '訊息5', got '%s'", recent[len(recent)-1].Content)
		}
	})

	t.Run("GetRecentMessages_EmptyChannel", func(t *testing.T) {
		recent := store.GetRecentMessages("empty", 10)
		
		if len(recent) != 1 {
			t.Errorf("Expected 1 welcome message for empty channel, got %d", len(recent))
		}
		
		if recent[0].Type != MessageTypeSystem {
			t.Errorf("Expected system message type, got '%s'", recent[0].Type)
		}
		
		expectedContent := "歡迎來到 empty 頻道！開始你的第一條消息吧 👋"
		if recent[0].Content != expectedContent {
			t.Errorf("Expected welcome message, got '%s'", recent[0].Content)
		}
	})

	t.Run("GetChannelMessageCount", func(t *testing.T) {
		count := store.GetChannelMessageCount("general")
		if count != 6 { // 1 + 5 from previous tests
			t.Errorf("Expected 6 messages in general channel, got %d", count)
		}
		
		emptyCount := store.GetChannelMessageCount("nonexistent")
		if emptyCount != 0 {
			t.Errorf("Expected 0 messages in nonexistent channel, got %d", emptyCount)
		}
	})

	t.Run("ClearChannel", func(t *testing.T) {
		store.ClearChannel("general")
		count := store.GetChannelMessageCount("general")
		if count != 0 {
			t.Errorf("Expected 0 messages after clearing channel, got %d", count)
		}
	})

	t.Run("Clear", func(t *testing.T) {
		// 添加測試訊息到多個頻道
		store.AddMessage(NewMessage("alice", "test1", "general"))
		store.AddMessage(NewMessage("bob", "test2", "tech"))
		
		store.Clear()
		
		if len(store) != 0 {
			t.Errorf("Expected empty store after clear, got %d channels", len(store))
		}
	})
}

// TestMessageIDUniqueness 測試訊息 ID 唯一性
func TestMessageIDUniqueness(t *testing.T) {
	ids := make(map[string]bool)
	
	// 建立多條訊息測試 ID 唯一性
	for i := 0; i < 1000; i++ {
		msg := NewMessage("test", "content", "general")
		
		if ids[msg.ID] {
			t.Errorf("Duplicate message ID found: %s", msg.ID)
		}
		
		ids[msg.ID] = true
		
		// 微小延遲確保時間戳不同
		time.Sleep(time.Nanosecond)
	}
}

// TestMessageTimestamp 測試時間戳
func TestMessageTimestamp(t *testing.T) {
	before := time.Now()
	msg := NewMessage("alice", "test", "general")
	after := time.Now()
	
	if msg.Timestamp.Before(before) || msg.Timestamp.After(after) {
		t.Errorf("Message timestamp %v should be between %v and %v", 
			msg.Timestamp, before, after)
	}
}

// BenchmarkNewMessage 基準測試：訊息建立效能
func BenchmarkNewMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewMessage("alice", "benchmark message", "general")
	}
}

// BenchmarkMessageStoreAddMessage 基準測試：訊息存儲效能
func BenchmarkMessageStoreAddMessage(b *testing.B) {
	store := make(MessageStore)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := NewMessage("alice", "benchmark message", "general")
		store.AddMessage(msg)
	}
}

// BenchmarkGetRecentMessages 基準測試：獲取最近訊息效能
func BenchmarkGetRecentMessages(b *testing.B) {
	store := make(MessageStore)
	
	// 預先添加大量訊息
	for i := 0; i < 10000; i++ {
		msg := NewMessage("alice", "benchmark message", "general")
		store.AddMessage(msg)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.GetRecentMessages("general", 50)
	}
}