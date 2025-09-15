package utils

import (
	"go-chat/models"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type OnlineUser struct {
	UserID   uint
	Username string
	Avatar   string
	Bio      string
	Status   string
	Conn     *websocket.Conn
	LastSeen time.Time
}

var OnlineUsers = struct {
	sync.RWMutex
	Users map[uint]*OnlineUser
}{Users: make(map[uint]*OnlineUser)}

// 添加在线用户
func AddOnlineUser(userID uint, username string, conn *websocket.Conn) {
	OnlineUsers.Lock()
	defer OnlineUsers.Unlock()

	// 获取用户完整信息
	var user models.User
	models.DB.First(&user, userID)

	// 更新用户状态为在线
	models.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"status":    "online",
		"last_seen": time.Now(),
	})

	OnlineUsers.Users[userID] = &OnlineUser{
		UserID:   userID,
		Username: username,
		Avatar:   user.Avatar,
		Bio:      user.Bio,
		Status:   "online",
		Conn:     conn,
		LastSeen: time.Now(),
	}
}

// 移除在线用户
func RemoveOnlineUser(userID uint) {
	OnlineUsers.Lock()
	defer OnlineUsers.Unlock()

	// 更新用户状态为离线
	models.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"status":    "offline",
		"last_seen": time.Now(),
	})

	delete(OnlineUsers.Users, userID)
}

// 添加获取用户状态函数
func GetUserStatus(userID uint) string {
	OnlineUsers.RLock()
	defer OnlineUsers.RUnlock()

	if _, exists := OnlineUsers.Users[userID]; exists {
		return "online"
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
	if user, exists := OnlineUsers.Users[userID]; exists {
		user.Status = status
	}
}

// 获取在线用户列表
func GetOnlineUsers() []*OnlineUser {
	OnlineUsers.RLock()
	defer OnlineUsers.RUnlock()

	users := make([]*OnlineUser, 0, len(OnlineUsers.Users))
	for _, user := range OnlineUsers.Users {
		users = append(users, user)
	}
	return users
}

// 定期清理不活跃用户
func CleanInactiveUsers() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		OnlineUsers.Lock()
		for userID, user := range OnlineUsers.Users {
			if time.Since(user.LastSeen) > 10*time.Minute {
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
