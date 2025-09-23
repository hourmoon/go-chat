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
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// 定义广播消息的结构
type BroadcastMessage struct {
	Type        string `json:"type"`
	UserID      uint   `json:"user_id"`
	Username    string `json:"username"`
	Content     string `json:"content"`
	MessageType string `json:"message_type"` // text, image, file
	FileURL     string `json:"file_url"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
	Target      uint   `json:"target"`   // 0表示全局/群聊，>0表示私聊目标用户ID
	GroupID     uint   `json:"group_id"` // 群组ID，0表示全局聊天，>0表示群聊
	CreatedAt   string `json:"created_at"`
}

// 全局存储连接
var (
	clients   = make(map[*websocket.Conn]bool)
	broadcast = make(chan BroadcastMessage)
	mutex     sync.RWMutex
)

// WebSocket Handler
func WSHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Printf("升级 WebSocket 失败: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("WebSocket 连接已建立")

	// 等待客户端发送认证消息
	_, authMsg, err := conn.ReadMessage()
	if err != nil {
		fmt.Printf("读取认证消息失败: %v\n", err)
		return
	}

	// 解析认证消息
	var authData struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(authMsg, &authData); err != nil || authData.Type != "auth" {
		conn.WriteMessage(websocket.TextMessage, []byte("认证消息格式错误"))
		return
	}

	// 验证 JWT token
	claims, err := utils.ParseJWT(authData.Token)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("Token 无效"))
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

	// 添加用户到在线列表
	utils.AddOnlineUser(userID, username, conn)

	// 广播用户上线消息
	joinMsg := BroadcastMessage{
		Type:      "user_joined",
		UserID:    userID,
		Username:  username,
		Content:   fmt.Sprintf("%s 加入了聊天室", username),
		Target:    0,
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}
	broadcast <- joinMsg

	// 确保连接关闭时从客户端映射中移除
	defer func() {
		utils.RemoveOnlineUser(userID)

		// 广播用户下线消息
		leaveMsg := BroadcastMessage{
			Type:      "user_left",
			UserID:    userID,
			Username:  username,
			Content:   fmt.Sprintf("%s 离开了聊天室", username),
			Target:    0,
			CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
		}
		broadcast <- leaveMsg

		mutex.Lock()
		delete(clients, conn)
		mutex.Unlock()
		fmt.Printf("❌ 用户断开: %s\n", username)
	}()

	// 监听消息
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("读取消息错误: %v\n", err)
			break
		}

		// 解析消息内容（可能是普通文本或JSON）
		var messageData map[string]interface{}
		if err := json.Unmarshal(msg, &messageData); err != nil {
			// 如果不是JSON，当作普通文本处理
			messageData = map[string]interface{}{
				"content": string(msg),
				"target":  0, // 默认群聊
			}
		}

		// 获取消息内容和目标
		content, ok := messageData["content"].(string)
		if !ok {
			content = string(msg)
		}

		messageType := "text"
		if typeVal, ok := messageData["messageType"].(string); ok {
			messageType = typeVal
		}

		fileURL := ""
		if urlVal, ok := messageData["fileUrl"].(string); ok {
			fileURL = urlVal
		}

		fileName := ""
		if nameVal, ok := messageData["fileName"].(string); ok {
			fileName = nameVal
		}

		fileSize := int64(0)
		if sizeVal, ok := messageData["fileSize"].(float64); ok {
			fileSize = int64(sizeVal)
		}

		target := uint(0)
		if targetVal, ok := messageData["target"].(float64); ok {
			target = uint(targetVal)
		}

		groupID := uint(0)
		if groupVal, ok := messageData["group_id"].(float64); ok {
			groupID = uint(groupVal)
		}

		// 创建消息实例并保存到数据库
		message := models.Message{
			UserID:      userID,
			Username:    username,
			Content:     content,
			MessageType: messageType,
			FileURL:     fileURL,
			FileName:    fileName,
			FileSize:    fileSize,
			GroupID:     groupID,
			CreatedAt:   time.Now(),
		}

		if err := models.DB.Create(&message).Error; err != nil {
			fmt.Printf("保存消息到数据库失败: %v\n", err)
		} else {
			fmt.Printf("消息已保存到数据库: %s: %s\n", username, content)
		}

		// 构建广播消息
		broadcastMsg := BroadcastMessage{
			Type:        "message",
			UserID:      userID,
			Username:    username,
			Content:     content,
			MessageType: messageType,
			FileURL:     fileURL,
			FileName:    fileName,
			FileSize:    fileSize,
			Target:      target,
			GroupID:     groupID,
			CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
		}
		// 给广播通道发送消息
		broadcast <- broadcastMsg
	}
}

// 广播消息给所有连接
func HandleMessages() {
	for {
		msg := <-broadcast

		// 根据消息类型处理
		if msg.Type == "user_joined" || msg.Type == "user_left" {
			// 用户上下线消息，广播给所有人
			mutex.RLock()
			for client := range clients {
				sendMessageToClient(client, msg)
			}
			mutex.RUnlock()
		} else if msg.Type == "group_member_joined" || msg.Type == "group_member_left" {
			// 群成员变动消息，仅广播给该群在线成员
			if msg.GroupID > 0 {
				broadcastToGroupMembers(msg, msg.GroupID)
			}
		} else if msg.GroupID > 0 {
			// 群聊消息，仅广播给该群在线成员
			broadcastToGroupMembers(msg, msg.GroupID)
		} else if msg.Target > 0 {
			// 私聊消息，只发送给目标用户和发送者
			utils.OnlineUsers.RLock()
			// 发送给目标用户
			if targetUser, exists := utils.OnlineUsers.Users[msg.Target]; exists {
				sendMessageToClient(targetUser.Conn, msg)
			}
			// 发送给发送者
			if sender, exists := utils.OnlineUsers.Users[msg.UserID]; exists {
				sendMessageToClient(sender.Conn, msg)
			}
			utils.OnlineUsers.RUnlock()
		} else {
			// 全局群聊消息，发送给所有用户
			mutex.RLock()
			for client := range clients {
				sendMessageToClient(client, msg)
			}
			mutex.RUnlock()
		}
	}
}

// 辅助函数：发送消息到客户端
func sendMessageToClient(client *websocket.Conn, msg BroadcastMessage) {
	messageJSON, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("JSON编码错误: %v\n", err)
		return
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

// 辅助函数：向群成员广播消息
func broadcastToGroupMembers(msg BroadcastMessage, groupID uint) {
	// 查询群成员ID列表
	var memberIDs []uint
	if err := models.DB.Model(&models.GroupMember{}).Where("group_id = ?", groupID).Pluck("user_id", &memberIDs).Error; err != nil {
		fmt.Printf("查询群成员失败: %v\n", err)
		return
	}

	// 发送给在线的群成员
	utils.OnlineUsers.RLock()
	for _, memberID := range memberIDs {
		if user, exists := utils.OnlineUsers.Users[memberID]; exists {
			sendMessageToClient(user.Conn, msg)
		}
	}
	utils.OnlineUsers.RUnlock()
}

// SendBroadcastMessage 向广播通道发送消息
func SendBroadcastMessage(msg BroadcastMessage) {
	broadcast <- msg
}
