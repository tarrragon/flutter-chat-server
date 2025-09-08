package main

// getTestAccounts 獲取測試帳號列表
//
// Responsible for:
// - 從設定檔獲取測試帳號資料
// - 每個帳號關聯到特定的頻道
//
// Design considerations:
// - 從外部設定檔載入，提高靈活性
// - 支援動態設定不同的測試帳號
// - 保持與原有程式碼的相容性
//
// Usage context:
// - 帳號驗證時進行比對
// - API 回應時提供可用帳號列表
//
// Returns:
//
//	[]Account: 從設定檔載入的測試帳號列表
func getTestAccounts() []Account {
	return DefaultTestAccounts
}

// validateAccount 驗證用戶帳號和密碼
//
// Responsible for:
// - 檢查用戶提供的帳號密碼是否有效
// - 返回對應的帳號資訊以便後續使用
//
// Design considerations:
// - 使用線性查找，適合小量帳號的情況
// - 返回帳號指標以避免不必要的複製
// - 布林返回值明確表示驗證結果
//
// Process flow:
// 1. 遍歷所有預設測試帳號
// 2. 比對用戶名和密碼是否完全匹配
// 3. 找到匹配項目時返回帳號資訊和 true
// 4. 遍歷完成仍未找到時返回 nil 和 false
//
// Usage context:
// - WebSocket 連接建立時驗證客戶端身份
// - 登入 API 端點驗證用戶憑證
//
// Parameters:
//
//	username: 用戶提供的用戶名
//	password: 用戶提供的密碼
//
// Returns:
//
//	*Account: 匹配的帳號資訊，驗證失敗時為 nil
//	bool: 驗證是否成功
func validateAccount(username, password string) (*Account, bool) {
	testAccounts := getTestAccounts()
	for _, account := range testAccounts {
		if account.Username == username && account.Password == password {
			return &account, true
		}
	}
	return nil, false
}
