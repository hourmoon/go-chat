package utils

import (
	"go-chat/models"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// OnlineUserState 用户在线状态（支持多连接）
type OnlineUserState struct {
	UserID      uint                     `json:"user_id"`
	Username    string                   `json:"username"`
	Avatar      string                   `json:"avatar"`
	Bio         string                   `json:"bio"`
	Status      string                   `json:"status"`
	LastSeen    time.Time                `json:"last_seen"`
	Connections map[*websocket.Conn]bool `json:"-"` // 该用户的所有连接
}

// OnlineUser 保持向后兼容的结构（用于返回）
type OnlineUser struct {
	UserID   uint            `json:"user_id"`
	Username string          `json:"username"`
	Avatar   string          `json:"avatar"`
	Bio      string          `json:"bio"`
	Status   string          `json:"status"`
	Conn     *websocket.Conn `json:"-"`
	LastSeen time.Time       `json:"last_seen"`
}

var OnlineUsers = struct {
	sync.RWMutex
	Users map[uint]*OnlineUserState
}{Users: make(map[uint]*OnlineUserState)}

// 添加在线用户（支持多连接）
func AddOnlineUser(userID uint, username string, conn *websocket.Conn) (isFirstConnection bool) {
	OnlineUsers.Lock()
	defer OnlineUsers.Unlock()

	userState, exists := OnlineUsers.Users[userID]

	if !exists {
		// 首个连接：获取用户完整信息并创建状态
		var user models.User
		models.DB.First(&user, userID)

		// 更新数据库状态为在线
		models.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
			"status":    "online",
			"last_seen": time.Now(),
		})

		userState = &OnlineUserState{
			UserID:      userID,
			Username:    username,
			Avatar:      user.Avatar,
			Bio:         user.Bio,
			Status:      "online",
			LastSeen:    time.Now(),
			Connections: make(map[*websocket.Conn]bool),
		}
		OnlineUsers.Users[userID] = userState
		isFirstConnection = true
	}

	// 添加新连接到该用户的连接集合
	userState.Connections[conn] = true
	userState.LastSeen = time.Now()

	return isFirstConnection
}

// 移除在线用户（支持多连接）
func RemoveOnlineUser(userID uint, conn *websocket.Conn) (isLastConnection bool) {
	OnlineUsers.Lock()
	defer OnlineUsers.Unlock()

	userState, exists := OnlineUsers.Users[userID]
	if !exists {
		return false
	}

	// 从该用户的连接集合中移除指定连接
	delete(userState.Connections, conn)
	userState.LastSeen = time.Now()

	// 如果是最后一个连接，则将用户标记为离线
	if len(userState.Connections) == 0 {
		// 更新数据库状态为离线
		models.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
			"status":    "offline",
			"last_seen": time.Now(),
		})

		// 从在线用户列表中移除
		delete(OnlineUsers.Users, userID)
		isLastConnection = true
	}

	return isLastConnection
}

// 获取用户状态函数
func GetUserStatus(userID uint) string {
	OnlineUsers.RLock()
	defer OnlineUsers.RUnlock()

	if userState, exists := OnlineUsers.Users[userID]; exists && len(userState.Connections) > 0 {
		return userState.Status
	}
	return "offline"
}

// 更新用户状态
func UpdateUserStatus(userID uint, status string) {
	OnlineUsers.Lock()
	defer OnlineUsers.Unlock()

	// 更新数据库中的状态
	models.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	})

	// 如果用户在线，更新内存中的状态
	if userState, exists := OnlineUsers.Users[userID]; exists {
		userState.Status = status
	}
}

// 获取在线用户列表（向后兼容）
func GetOnlineUsers() []*OnlineUser {
	OnlineUsers.RLock()
	defer OnlineUsers.RUnlock()

	users := make([]*OnlineUser, 0, len(OnlineUsers.Users))
	for _, userState := range OnlineUsers.Users {
		if len(userState.Connections) > 0 {
			// 为向后兼容，选择第一个连接作为代表
			var firstConn *websocket.Conn
			for conn := range userState.Connections {
				firstConn = conn
				break
			}

			users = append(users, &OnlineUser{
				UserID:   userState.UserID,
				Username: userState.Username,
				Avatar:   userState.Avatar,
				Bio:      userState.Bio,
				Status:   userState.Status,
				Conn:     firstConn,
				LastSeen: userState.LastSeen,
			})
		}
	}
	return users
}

// 获取用户的所有连接（新增函数）
func GetUserConnections(userID uint) []*websocket.Conn {
	OnlineUsers.RLock()
	defer OnlineUsers.RUnlock()

	if userState, exists := OnlineUsers.Users[userID]; exists {
		connections := make([]*websocket.Conn, 0, len(userState.Connections))
		for conn := range userState.Connections {
			connections = append(connections, conn)
		}
		return connections
	}
	return nil
}

// 定期清理不活跃用户
func CleanInactiveUsers() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		OnlineUsers.Lock()
		for userID, userState := range OnlineUsers.Users {
			if time.Since(userState.LastSeen) > 10*time.Minute {
				// 更新数据库状态为离线
				models.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
					"status":    "offline",
					"last_seen": time.Now(),
				})
				delete(OnlineUsers.Users, userID)
			}
		}
		OnlineUsers.Unlock()
	}
}

// 初始化函数，在main.go中调用
func InitOnlineUsers() {
	go CleanInactiveUsers()
}
