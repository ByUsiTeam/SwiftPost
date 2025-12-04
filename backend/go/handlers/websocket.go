package handlers

import (
	"SwiftPost/models"
	"SwiftPost/utils"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket消息类型
const (
	MessageTypeNewEmail   = "new_email"
	MessageTypeReadEmail  = "read_email"
	MessageTypeDeleteEmail = "delete_email"
	MessageTypeTyping     = "typing"
	MessageTypePresence   = "presence"
	MessageTypeError      = "error"
	MessageTypeInfo       = "info"
)

// WebSocket消息结构
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

// 客户端结构
type WebSocketClient struct {
	ID     string
	UserID int
	Conn   *websocket.Conn
	Send   chan WebSocketMessage
	Mutex  sync.Mutex
}

// WebSocket管理器
type WebSocketManager struct {
	Clients    map[string]*WebSocketClient
	Register   chan *WebSocketClient
	Unregister chan *WebSocketClient
	Broadcast  chan WebSocketMessage
	Mutex      sync.RWMutex
}

// 全局WebSocket管理器
var manager = &WebSocketManager{
	Clients:    make(map[string]*WebSocketClient),
	Register:   make(chan *WebSocketClient),
	Unregister: make(chan *WebSocketClient),
	Broadcast:  make(chan WebSocketMessage),
}

// 启动WebSocket管理器
func StartWebSocketManager() {
	utils.PrintInfo("启动WebSocket管理器")
	
	for {
		select {
		case client := <-manager.Register:
			manager.Mutex.Lock()
			manager.Clients[client.ID] = client
			manager.Mutex.Unlock()
			
			utils.Debug("WebSocket客户端注册: %s (用户ID: %d)", client.ID, client.UserID)
			
			// 发送欢迎消息
			welcomeMsg := WebSocketMessage{
				Type:      MessageTypeInfo,
				Payload:   map[string]interface{}{"message": "连接成功"},
				Timestamp: time.Now(),
			}
			client.Send <- welcomeMsg
			
			// 广播用户上线通知
			presenceMsg := WebSocketMessage{
				Type: MessageTypePresence,
				Payload: map[string]interface{}{
					"user_id":  client.UserID,
					"status":   "online",
					"client_id": client.ID,
				},
				Timestamp: time.Now(),
			}
			manager.Broadcast <- presenceMsg
			
		case client := <-manager.Unregister:
			manager.Mutex.Lock()
			if _, ok := manager.Clients[client.ID]; ok {
				close(client.Send)
				delete(manager.Clients, client.ID)
			}
			manager.Mutex.Unlock()
			
			utils.Debug("WebSocket客户端注销: %s", client.ID)
			
			// 广播用户下线通知
			presenceMsg := WebSocketMessage{
				Type: MessageTypePresence,
				Payload: map[string]interface{}{
					"user_id":  client.UserID,
					"status":   "offline",
					"client_id": client.ID,
				},
				Timestamp: time.Now(),
			}
			manager.Broadcast <- presenceMsg
			
		case message := <-manager.Broadcast:
			manager.Mutex.RLock()
			for _, client := range manager.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(manager.Clients, client.ID)
				}
			}
			manager.Mutex.RUnlock()
		}
	}
}

// 发送消息给特定用户
func SendToUser(userID int, message WebSocketMessage) {
	manager.Mutex.RLock()
	defer manager.Mutex.RUnlock()
	
	for _, client := range manager.Clients {
		if client.UserID == userID {
			select {
			case client.Send <- message:
			default:
				// 如果发送通道已满，关闭连接
				close(client.Send)
				delete(manager.Clients, client.ID)
			}
		}
	}
}

// 发送消息给除发送者外的所有用户
func BroadcastExcluding(senderID string, message WebSocketMessage) {
	manager.Mutex.RLock()
	defer manager.Mutex.RUnlock()
	
	for id, client := range manager.Clients {
		if id != senderID {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(manager.Clients, id)
			}
		}
	}
}

// WebSocket处理器
func WebSocketHandler(w http.ResponseWriter, r *http.Request, db *models.Database, upgrader websocket.Upgrader) {
	// 从查询参数获取用户ID和Token
	userIDStr := r.URL.Query().Get("user_id")
	token := r.URL.Query().Get("token")
	
	if userIDStr == "" || token == "" {
		utils.Error("WebSocket连接缺少参数")
		http.Error(w, "缺少参数", http.StatusBadRequest)
		return
	}
	
	// 验证Token（简化版，实际应该使用JWT验证）
	// 这里为了简化，只做基本验证
	config, _ := utils.LoadConfig("config.json")
	if token != config.Security.JWTSecret {
		utils.Error("WebSocket连接Token无效")
		http.Error(w, "无效的Token", http.StatusUnauthorized)
		return
	}
	
	// 升级HTTP连接到WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		utils.Error("WebSocket升级失败: %v", err)
		return
	}
	
	// 生成客户端ID
	clientID := generateClientID()
	userID, _ := strconv.Atoi(userIDStr)
	
	// 创建客户端
	client := &WebSocketClient{
		ID:     clientID,
		UserID: userID,
		Conn:   conn,
		Send:   make(chan WebSocketMessage, 256),
	}
	
	// 注册客户端
	manager.Register <- client
	
	// 启动读写协程
	go client.writePump()
	go client.readPump(db)
	
	utils.Info("WebSocket连接建立: %s (用户ID: %d)", clientID, userID)
}

// 生成客户端ID
func generateClientID() string {
	return fmt.Sprintf("client_%d_%d", time.Now().UnixNano(), rand.Intn(1000))
}

// 写协程
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
		manager.Unregister <- c
	}()
	
	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				// 发送通道关闭
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			c.Mutex.Lock()
			err := c.Conn.WriteJSON(message)
			c.Mutex.Unlock()
			
			if err != nil {
				log.Printf("WebSocket写错误: %v", err)
				return
			}
			
		case <-ticker.C:
			// 发送ping消息保持连接
			c.Mutex.Lock()
			err := c.Conn.WriteMessage(websocket.PingMessage, nil)
			c.Mutex.Unlock()
			
			if err != nil {
				return
			}
		}
	}
}

// 读协程
func (c *WebSocketClient) readPump(db *models.Database) {
	defer func() {
		c.Conn.Close()
		manager.Unregister <- c
	}()
	
	for {
		var msg WebSocketMessage
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				utils.Error("WebSocket读错误: %v", err)
			}
			break
		}
		
		// 处理消息
		c.handleMessage(msg, db)
	}
}

// 处理WebSocket消息
func (c *WebSocketClient) handleMessage(msg WebSocketMessage, db *models.Database) {
	switch msg.Type {
	case MessageTypeTyping:
		// 处理输入状态
		if payload, ok := msg.Payload.(map[string]interface{}); ok {
			toUserID, ok := payload["to_user_id"].(float64)
			if ok {
				typingMsg := WebSocketMessage{
					Type: MessageTypeTyping,
					Payload: map[string]interface{}{
						"from_user_id": c.UserID,
						"is_typing":    payload["is_typing"],
					},
					Timestamp: time.Now(),
				}
				SendToUser(int(toUserID), typingMsg)
			}
		}
		
	case MessageTypePresence:
		// 处理在线状态更新
		if payload, ok := msg.Payload.(map[string]interface{}); ok {
			status, _ := payload["status"].(string)
			
			presenceMsg := WebSocketMessage{
				Type: MessageTypePresence,
				Payload: map[string]interface{}{
					"user_id": c.UserID,
					"status":  status,
					"client_id": c.ID,
				},
				Timestamp: time.Now(),
			}
			manager.Broadcast <- presenceMsg
		}
		
	case "ping":
		// 响应ping
		pongMsg := WebSocketMessage{
			Type:      "pong",
			Payload:   nil,
			Timestamp: time.Now(),
		}
		c.Send <- pongMsg
		
	default:
		// 未知消息类型
		errorMsg := WebSocketMessage{
			Type:      MessageTypeError,
			Payload:   map[string]interface{}{"error": "未知消息类型"},
			Timestamp: time.Now(),
		}
		c.Send <- errorMsg
	}
}

// 发送新邮件通知
func NotifyNewEmail(db *models.Database, emailID int) {
	email, err := models.GetEmailByID(db, emailID)
	if err != nil {
		utils.Error("获取邮件信息失败: %v", err)
		return
	}
	
	// 获取发件人信息
	sender, err := models.GetUserByID(db, email.SenderID)
	if err != nil {
		utils.Error("获取发件人信息失败: %v", err)
		sender = &models.User{Username: "未知用户"}
	}
	
	// 创建通知消息
	notification := WebSocketMessage{
		Type: MessageTypeNewEmail,
		Payload: map[string]interface{}{
			"email_id":      email.ID,
			"sender_id":     email.SenderID,
			"sender_name":   sender.Username,
			"sender_email":  email.SenderEmail,
			"subject":       email.Subject,
			"preview":       getBodyPreview(email.Body),
			"has_attachment": email.HasAttachment,
			"created_at":    email.CreatedAt,
		},
		Timestamp: time.Now(),
	}
	
	// 发送给收件人
	SendToUser(email.RecipientID, notification)
	
	utils.Debug("新邮件通知已发送: 邮件ID=%d, 收件人ID=%d", email.ID, email.RecipientID)
}

// 发送邮件已读通知
func NotifyEmailRead(db *models.Database, emailID int, readerID int) {
	email, err := models.GetEmailByID(db, emailID)
	if err != nil {
		return
	}
	
	// 只有发件人才需要知道邮件被阅读
	if email.SenderID != readerID {
		notification := WebSocketMessage{
			Type: MessageTypeReadEmail,
			Payload: map[string]interface{}{
				"email_id":    email.ID,
				"reader_id":   readerID,
				"read_at":     time.Now(),
			},
			Timestamp: time.Now(),
		}
		
		SendToUser(email.SenderID, notification)
	}
}

// 获取在线用户列表
func GetOnlineUsers() []int {
	manager.Mutex.RLock()
	defer manager.Mutex.RUnlock()
	
	onlineUsers := make(map[int]bool)
	var userList []int
	
	for _, client := range manager.Clients {
		if !onlineUsers[client.UserID] {
			onlineUsers[client.UserID] = true
			userList = append(userList, client.UserID)
		}
	}
	
	return userList
}

// 检查用户是否在线
func IsUserOnline(userID int) bool {
	manager.Mutex.RLock()
	defer manager.Mutex.RUnlock()
	
	for _, client := range manager.Clients {
		if client.UserID == userID {
			return true
		}
	}
	
	return false
}

// 初始化WebSocket
func init() {
	go StartWebSocketManager()
}

// 需要导入的包
import (
	"fmt"
	"math/rand"
	"strconv"
)