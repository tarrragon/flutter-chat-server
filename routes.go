package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

// setupRoutes 設置所有的 HTTP 路由
//
// Responsible for:
// - 配置所有 REST API 端點的路由映射
// - 設置 WebSocket 端點路由
// - 配置靜態檔案服務
//
// Design considerations:
// - 使用 Gorilla Mux 路由器支援更靈活的路由配置
// - REST API 端點支援 CORS 所需的 OPTIONS 方法
// - 靜態檔案服務使用 PathPrefix 處理所有未匹配的路徑
//
// Process flow:
// 1. 建立新的 mux 路由器實例
// 2. 註冊所有 REST API 端點及其處理函式
// 3. 註冊 WebSocket 端點
// 4. 設置靜態檔案服務作為後備處理
// 5. 返回配置完成的路由器
//
// Usage context:
// - 主程式啟動時調用以設置路由
// - HTTP 伺服器使用返回的路由器處理請求
//
// Returns:
//
//	*mux.Router: 配置完成的路由器實例
func setupRoutes() *mux.Router {
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

	return r
}
