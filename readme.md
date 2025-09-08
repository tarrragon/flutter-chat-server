# Flutter 聊天室 WebSocket 服務器

一個支援即時聊天的 Go WebSocket 服務器，適用於 Flutter 應用程式開發測試。

## 功能

- ✅ 即時聊天 (WebSocket)
- ✅ 多用戶帳號系統 (3個預設測試帳號)
- ✅ 獨立頻道系統 (每個帳號有專屬頻道)
- ✅ 帳號驗證和登入
- ✅ 歷史訊息存儲 (按頻道分類)
- ✅ 用戶上線/離線通知
- ✅ REST API 支援
- ✅ 跨平台支援 (iOS/Android)
- ✅ 靜態檔案服務 (前端測試頁面)

## 快速開始

### 1.安裝 Go

官方下載：
<https://go.dev/dl/>

使用 Homebrew：

```bash
brew install go
```

### 2. 創建項目結構

```bash
mkdir flutter-chat-server
cd flutter-chat-server
go mod init flutter-chat-server
```

### 3. 安裝依賴

```bash
go get github.com/gorilla/mux
go get github.com/gorilla/websocket
```

### 4. 運行服務器

```bash
go run .
```

服務器啟動後會顯示：

```bash
🚀 服務器啟動在 http://localhost:8080
📱 手機端可連接: http://你的內網IP:8080
💻 WebSocket 端點: ws://localhost:8080/ws?username=帳號&password=密碼
📡 API 端點:
   GET  /api/messages?channel=頻道 - 獲取指定頻道的歷史消息
   POST /api/messages - 發送消息
   GET  /api/users - 獲取按頻道分組的在線用戶
   GET  /api/accounts - 獲取可用的測試帳號
   POST /api/login - 驗證帳號登入

🧪 測試帳號:
   用戶: alice, 密碼: password123, 頻道: general
   用戶: bob, 密碼: password123, 頻道: tech
   用戶: charlie, 密碼: password123, 頻道: random
```

### 5. 獲取內網 IP 地址

手機要連接到你的 Mac，需要使用內網 IP：

```bash
# 查看內網 IP
ifconfig | grep "inet " | grep -v 127.0.0.1
```

## API 文檔

### REST API 端點

#### GET /api/messages?channel=頻道名稱

獲取指定頻道的歷史訊息（最近 50 條）

**必要參數：**

- `channel`: 頻道名稱 (general, tech, random)

**回應格式：**

```json
[
  {
    "id": "1672502400",
    "user": "alice",
    "content": "你好，大家好！",
    "timestamp": "2023-01-01T12:00:00Z",
    "type": "text",
    "channel": "general"
  }
]
```

#### POST /api/messages

發送新訊息

**請求格式：**

```json
{
  "content": "訊息內容",
  "type": "text",
  "channel": "general"
}
```

**回應格式：**

```json
{
  "status": "sent"
}
```

#### GET /api/users

獲取按頻道分組的在線用戶

**回應格式：**

```json
{
  "channelUsers": {
    "general": ["alice"],
    "tech": ["bob"],
    "random": ["charlie"]
  },
  "totalCount": 3
}
```

#### GET /api/accounts

獲取可用的測試帳號列表

**回應格式：**

```json
{
  "accounts": [
    {
      "username": "alice",
      "channel": "general"
    },
    {
      "username": "bob", 
      "channel": "tech"
    },
    {
      "username": "charlie",
      "channel": "random"
    }
  ]
}
```

#### POST /api/login

驗證帳號登入

**請求格式：**

```json
{
  "username": "alice",
  "password": "password123"
}
```

**成功回應：**

```json
{
  "success": true,
  "account": {
    "username": "alice",
    "channel": "general"
  }
}
```

**錯誤回應：**

```json
{
  "error": "Invalid username or password"
}
```

### WebSocket 連接

**連接端點：** `ws://localhost:8080/ws?username=帳號名稱&password=密碼`

**必要參數：**

- `username`: 用戶名稱 (alice, bob, charlie)
- `password`: 密碼 (所有帳號都是 password123)

**連接驗證：**

- 如果帳號或密碼錯誤，連接會被拒絕
- 成功連接後會自動加入該帳號對應的頻道

#### 訊息結構

```json
{
  "id": "訊息ID",
  "user": "用戶名稱",
  "content": "訊息內容",
  "timestamp": "2023-01-01T12:00:00Z",
  "type": "訊息類型",
  "channel": "頻道名稱"
}
```

#### 支援的訊息類型

- `text` - 文字訊息
- `system` - 系統訊息（用戶加入/離開通知）
- `image` - 圖片訊息（預留）
- `file` - 檔案訊息（預留）

## Flutter 集成範例

### REST API 使用

```dart
import 'dart:convert';
import 'package:http/http.dart' as http;

class ChatService {
  static const String baseUrl = 'http://你的內網IP:8080';
  
  // 獲取可用帳號
  Future<List<Account>> getAccounts() async {
    final response = await http.get(
      Uri.parse('$baseUrl/api/accounts'),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      final List<dynamic> accounts = data['accounts'];
      return accounts.map((json) => Account.fromJson(json)).toList();
    }
    throw Exception('Failed to load accounts');
  }
  
  // 驗證登入
  Future<Account> login(String username, String password) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/login'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'username': username,
        'password': password,
      }),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return Account.fromJson(data['account']);
    }
    throw Exception('Login failed');
  }
  
  // 獲取指定頻道的歷史訊息
  Future<List<Message>> getMessages(String channel) async {
    final response = await http.get(
      Uri.parse('$baseUrl/api/messages?channel=$channel'),
    );
    
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((json) => Message.fromJson(json)).toList();
    }
    throw Exception('Failed to load messages');
  }
  
  // 發送訊息到指定頻道
  Future<void> sendMessage(String content, String channel) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/messages'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'content': content,
        'type': 'text',
        'channel': channel,
      }),
    );
    
    if (response.statusCode != 200) {
      throw Exception('Failed to send message');
    }
  }
  
  // 獲取在線用戶
  Future<Map<String, List<String>>> getOnlineUsers() async {
    final response = await http.get(
      Uri.parse('$baseUrl/api/users'),
    );
    
    if (response.statusCode == 200) {
      final data = json.decode(response.body);
      return Map<String, List<String>>.from(
        data['channelUsers'].map((k, v) => MapEntry(k, List<String>.from(v)))
      );
    }
    throw Exception('Failed to load users');
  }
}
```

### WebSocket 使用

```dart
import 'package:web_socket_channel/web_socket_channel.dart';

class WebSocketService {
  late WebSocketChannel channel;
  
  void connect(String username, String password) {
    channel = WebSocketChannel.connect(
      Uri.parse('ws://你的內網IP:8080/ws?username=$username&password=$password'),
    );
    
    // 監聽訊息
    channel.stream.listen((data) {
      final message = Message.fromJson(json.decode(data));
      // 處理收到的訊息（只會收到該帳號頻道的訊息）
    }, onError: (error) {
      // 處理連接錯誤（如帳號驗證失敗）
      print('WebSocket error: $error');
    });
  }
  
  void sendMessage(String content) {
    final message = {
      'content': content,
      'type': 'text',
      // channel 會由服務器自動設置為當前用戶的頻道
    };
    channel.sink.add(json.encode(message));
  }
  
  void disconnect() {
    channel.sink.close();
  }
}
```

### 完整的模型定義範例

```dart
// 帳號模型
class Account {
  final String username;
  final String channel;

  Account({
    required this.username,
    required this.channel,
  });

  factory Account.fromJson(Map<String, dynamic> json) {
    return Account(
      username: json['username'],
      channel: json['channel'],
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'username': username,
      'channel': channel,
    };
  }
}

// 訊息模型
class Message {
  final String id;
  final String user;
  final String content;
  final DateTime timestamp;
  final String type;
  final String channel;

  Message({
    required this.id,
    required this.user,
    required this.content,
    required this.timestamp,
    required this.type,
    required this.channel,
  });

  factory Message.fromJson(Map<String, dynamic> json) {
    return Message(
      id: json['id'],
      user: json['user'],
      content: json['content'],
      timestamp: DateTime.parse(json['timestamp']),
      type: json['type'],
      channel: json['channel'],
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'user': user,
      'content': content,
      'timestamp': timestamp.toIso8601String(),
      'type': type,
      'channel': channel,
    };
  }
}
```

## 前端測試頁面

服務器包含一個功能完整的多帳號測試頁面，支援所有新功能：

### 📱 測試頁面功能

1. **帳號選擇** - 選擇測試帳號
2. **即時聊天** - WebSocket 即時訊息更新
3. **頻道隔離** - 每個帳號只能看到自己頻道的訊息
4. **在線用戶** - 按頻道顯示在線用戶狀態
5. **除錯模式** - 詳細的連接和訊息除錯資訊

### 🚀 使用步驟

1. **啟動服務器** - `go run .`
2. **開啟瀏覽器** - 訪問 `http://localhost:8080`
3. **選擇帳號** - 點擊任一個帳號卡片 (alice/bob/charlie)
4. **連接聊天室** - 點擊「連接聊天室」按鈕
5. **開始聊天** - 自動進入對應頻道開始聊天

### 🧪 多帳號測試建議

- **開啟多個瀏覽器標籤** - 用不同帳號登入測試頻道隔離
- **開啟除錯模式** - 查看 WebSocket 連接狀態和訊息流
- **測試 API 功能** - 使用載入歷史訊息和查看在線用戶功能

## 疑難排解

### 使用設定

1. **訊息限制**：目前單次讀取限制 512 字元，大型訊息請分段發送
2. **訊息存儲**：目前使用記憶體存儲，服務器重啟後訊息會清空

## 測試帳號系統

### 預設帳號列表

| 用戶名 | 密碼 | 頻道 | 說明 |
|--------|------|------|------|
| alice | password123 | general | 一般討論頻道 |
| bob | password123 | tech | 技術討論頻道 |
| charlie | password123 | random | 隨機話題頻道 |

### 頻道隔離機制

- 每個帳號只能在自己的頻道內發送和接收訊息
- 不同頻道的用戶無法看到其他頻道的訊息
- 系統訊息（加入/離開通知）也按頻道分離

## 技術架構

- **後端框架**：Go + Gorilla WebSocket + Gorilla Mux
- **帳號系統**：預設三個測試帳號，支援密碼驗證
- **頻道系統**：獨立頻道隔離，訊息按頻道分類存儲和廣播
- **通訊協定**：WebSocket (即時) + HTTP REST API (歷史資料)
- **資料存儲**：記憶體存儲，按頻道分類（重啟後清空）
- **廣播機制**：256 緩衝區的 channel，確保訊息可靠傳遞
- **跨域支援**：已開啟 CORS，支援前端開發
- **並發處理**：每個客戶端連接使用獨立的 goroutine 處理

## 伺服器端點總覽

| 端點 | 方法 | 描述 | 用途 |
|------|------|------|------|
| `/api/messages?channel=頻道` | GET | 獲取指定頻道的歷史訊息 | 載入聊天記錄 |
| `/api/messages` | POST | 發送新訊息到指定頻道 | 透過 REST API 發送 |
| `/api/users` | GET | 獲取按頻道分組的在線用戶 | 顯示各頻道在線人數 |
| `/api/accounts` | GET | 獲取可用的測試帳號 | 登入頁面選擇帳號 |
| `/api/login` | POST | 驗證帳號登入 | 帳號驗證 |
| `/ws?username=&password=` | WebSocket | 需驗證的 WebSocket 連接 | 即時聊天通訊 |
| `/` | GET | 靜態檔案服務 | 前端測試頁面 |
