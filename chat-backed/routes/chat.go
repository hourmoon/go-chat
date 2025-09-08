package routes

import (
	"encoding/json"
	"fmt"
	"go-chat/models"
	"go-chat/utils"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// 升级 HTTP 连接为 WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 允许所有跨域，或根据需要限制
		return true
	},
	// 添加缓冲区大小配置
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// 定义广播消息的结构
type BroadcastMessage struct {
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// 全局存储连接
var (
	clients   = make(map[*websocket.Conn]bool) // 存储活跃的客户端
	broadcast = make(chan BroadcastMessage)    // 广播消息的通道，改为结构体类型
	mutex     sync.RWMutex                     // 读写锁
)

// WebSocket Handler
func WSHandler(c *gin.Context) {
	// 升级协议
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil) // 升级 HTTP 连接为 WebSocket
	if err != nil {
		fmt.Printf("升级 WebSocket 失败: %v\n", err)
		fmt.Printf("请求头: %+v\n", c.Request.Header)
		return
	}
	defer conn.Close() // 确保连接关闭时关闭 WebSocket 连接

	fmt.Println("WebSocket 连接已建立")

	// 等待客户端发送认证消息
	_, authMsg, err := conn.ReadMessage() // 读取认证消息
	if err != nil {
		fmt.Printf("读取认证消息失败: %v\n", err)
		return
	}

	fmt.Printf("收到认证消息: %s\n", string(authMsg))

	// 解析认证消息
	var authData struct { // 定义认证消息的结构
		Type  string `json:"type"`  // 消息类型
		Token string `json:"token"` // JWT token
	}
	if err := json.Unmarshal(authMsg, &authData); err != nil || authData.Type != "auth" { // 解析认证消息
		fmt.Printf("认证消息格式错误: %v\n", err)
		conn.WriteMessage(websocket.TextMessage, []byte("认证消息格式错误"))
		return
	}

	// 验证 JWT token
	claims, err := utils.ParseJWT(authData.Token)
	if err != nil {
		fmt.Printf("Token 无效: %v\n", err)
		conn.WriteMessage(websocket.TextMessage, []byte("Token 无效: "+err.Error()))
		return
	}

	// 保存用户信息
	userID := claims.UserID
	username := claims.Username
	fmt.Printf("✅ 新用户连接: %s (用户ID: %d)\n", username, userID)

	// 将连接添加到客户端映射
	mutex.Lock()
	clients[conn] = true
	mutex.Unlock()

	// 确保连接关闭时从客户端映射中移除
	defer func() {
		mutex.Lock()
		delete(clients, conn)
		mutex.Unlock()
		fmt.Printf("❌ 用户断开: %s\n", username)
	}()

	// 发送欢迎消息
	welcomeMsg := BroadcastMessage{
		UserID:    0, // 系统消息
		Username:  "系统",
		Content:   fmt.Sprintf("欢迎 %s 加入聊天室!", username),
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}
	broadcast <- welcomeMsg

	// 监听消息
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("读取消息错误: %v\n", err)
			break
		}

		// 创建消息实例并保存到数据库
		message := models.Message{
			UserID:    userID,
			Username:  username,
			Content:   string(msg),
			CreatedAt: time.Now(),
		}
		if err := models.DB.Create(&message).Error; err != nil {
			fmt.Printf("保存消息到数据库失败: %v\n", err)
		} else {
			fmt.Printf("消息已保存到数据库: %s: %s\n", username, string(msg))
		}

		// 构建广播消息
		broadcastMsg := BroadcastMessage{
			UserID:    userID,
			Username:  username,
			Content:   string(msg),
			CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
		}

		// 给广播通道发送消息
		broadcast <- broadcastMsg
	}
}

// 广播消息给所有连接
func HandleMessages() {
	for {
		msg := <-broadcast
		mutex.RLock()
		for client := range clients {
			// 发送结构化的消息
			messageData := map[string]interface{}{
				"type":       "message",
				"user_id":    msg.UserID,
				"username":   msg.Username,
				"content":    msg.Content,
				"created_at": msg.CreatedAt,
			}
			messageJSON, err := json.Marshal(messageData)
			if err != nil {
				fmt.Printf("JSON编码错误: %v\n", err)
				continue
			}

			err = client.WriteMessage(websocket.TextMessage, messageJSON)
			if err != nil {
				fmt.Printf("发送消息失败: %v\n", err)
				mutex.Lock()
				client.Close()
				delete(clients, client)
				mutex.Unlock()
			}
		}
		mutex.RUnlock()
	}
}
