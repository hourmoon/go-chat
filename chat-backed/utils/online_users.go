package utils

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type OnlineUser struct {
	UserID   uint
	Username string
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

	OnlineUsers.Users[userID] = &OnlineUser{
		UserID:   userID,
		Username: username,
		Conn:     conn,
		LastSeen: time.Now(),
	}
}

// 移除在线用户
func RemoveOnlineUser(userID uint) {
	OnlineUsers.Lock()
	defer OnlineUsers.Unlock()

	delete(OnlineUsers.Users, userID)
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
			if time.Since(user.LastSeen) > 5*time.Minute {
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
