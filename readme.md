# Flutter 聊天室 WebSocket 服務器

一個支援即時聊天的 Go WebSocket 服務器，適用於 Flutter 應用程式開發測試。

## 功能

- ✅ 即時聊天 (WebSocket)
- ✅ 歷史訊息存儲
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
go run main.go
```

服務器啟動後會顯示：

```bash
🚀 服務器啟動在 http://localhost:8080
📱 手機端可連接: http://你的內網IP:8080
💻 WebSocket 端點: ws://localhost:8080/ws
📡 API 端點:
   GET  /api/messages - 獲取歷史消息
   POST /api/messages - 發送消息
   GET  /api/users    - 獲取在線用戶
```

### 5. 獲取內網 IP 地址

手機要連接到你的 Mac，需要使用內網 IP：

```bash
# 查看內網 IP
ifconfig | grep "inet " | grep -v 127.0.0.1
```

## API 文檔

### REST API 端點

#### GET /api/messages

獲取歷史訊息（最近 50 條）

**回應格式：**

```json
[
  {
    "id": "1672502400",
    "user": "張三",
    "content": "你好，大家好！",
    "timestamp": "2023-01-01T12:00:00Z",
    "type": "text"
  }
]
```

#### POST /api/messages

發送新訊息

**請求格式：**

```json
{
  "content": "訊息內容",
  "type": "text"
}
```

**回應格式：**

```json
{
  "status": "sent"
}
```

#### GET /api/users

獲取目前在線用戶

**回應格式：**

```json
{
  "users": ["張三", "李四", "王五"],
  "count": 3
}
```

### WebSocket 連接

**連接端點：** `ws://localhost:8080/ws?username=你的用戶名`

#### 訊息結構

```json
{
  "id": "訊息ID",
  "user": "用戶名稱",
  "content": "訊息內容",
  "timestamp": "2023-01-01T12:00:00Z",
  "type": "訊息類型"
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
  
  // 獲取歷史訊息
  Future<List<Message>> getMessages() async {
    final response = await http.get(
      Uri.parse('$baseUrl/api/messages'),
    );
    
    if (response.statusCode == 200) {
      final List<dynamic> data = json.decode(response.body);
      return data.map((json) => Message.fromJson(json)).toList();
    }
    throw Exception('Failed to load messages');
  }
  
  // 發送訊息
  Future<void> sendMessage(String content) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/messages'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'content': content,
        'type': 'text',
      }),
    );
    
    if (response.statusCode != 200) {
      throw Exception('Failed to send message');
    }
  }
}
```

### WebSocket 使用

```dart
import 'package:web_socket_channel/web_socket_channel.dart';

class WebSocketService {
  late WebSocketChannel channel;
  
  void connect(String username) {
    channel = WebSocketChannel.connect(
      Uri.parse('ws://你的內網IP:8080/ws?username=$username'),
    );
    
    // 監聽訊息
    channel.stream.listen((data) {
      final message = Message.fromJson(json.decode(data));
      // 處理收到的訊息
    });
  }
  
  void sendMessage(String content) {
    final message = {
      'content': content,
      'type': 'text',
    };
    channel.sink.add(json.encode(message));
  }
  
  void disconnect() {
    channel.sink.close();
  }
}
```

### 完整的 Message 模型範例

```dart
class Message {
  final String id;
  final String user;
  final String content;
  final DateTime timestamp;
  final String type;

  Message({
    required this.id,
    required this.user,
    required this.content,
    required this.timestamp,
    required this.type,
  });

  factory Message.fromJson(Map<String, dynamic> json) {
    return Message(
      id: json['id'],
      user: json['user'],
      content: json['content'],
      timestamp: DateTime.parse(json['timestamp']),
      type: json['type'],
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'user': user,
      'content': content,
      'timestamp': timestamp.toIso8601String(),
      'type': type,
    };
  }
}
```

## 前端測試頁面

服務器包含一個簡單的測試頁面，可在瀏覽器中測試聊天功能：

1. 啟動服務器後，開啟瀏覽器
2. 訪問 `http://localhost:8080` 或 `http://你的內網IP:8080`
3. 在測試頁面中進行聊天測試

## 疑難排解

### 使用設定

1. **測試環境**：建議先在本機測試，再移至手機測試
2. **錯誤處理**：實作適當的錯誤處理和重連機制
3. **訊息限制**：目前單次讀取限制 512 字元，大型訊息請分段發送
4. **訊息存儲**：目前使用記憶體存儲，服務器重啟後訊息會清空
5. **生產環境**：部署至生產環境時需要額外的安全性考量

## 技術架構

- **後端框架**：Go + Gorilla WebSocket + Gorilla Mux
- **通訊協定**：WebSocket (即時) + HTTP REST API (歷史資料)
- **資料存儲**：記憶體存儲（重啟後清空）
- **跨域支援**：已開啟 CORS，支援前端開發
- **並發處理**：每個客戶端連接使用獨立的 goroutine 處理

## 伺服器端點總覽

| 端點 | 方法 | 描述 | 用途 |
|------|------|------|------|
| `/api/messages` | GET | 獲取歷史訊息 | 載入聊天記錄 |
| `/api/messages` | POST | 發送新訊息 | 透過 REST API 發送 |
| `/api/users` | GET | 獲取在線用戶列表 | 顯示目前在線人數 |
| `/ws` | WebSocket | WebSocket 連接 | 即時聊天通訊 |
| `/` | GET | 靜態檔案服務 | 前端測試頁面 |