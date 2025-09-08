
# Flutter 集成範例

## REST API 使用

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

## 完整的模型定義範例

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
