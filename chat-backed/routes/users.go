package routes

import (
	"go-chat/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 获取在线用户列表
func GetOnlineUsers(c *gin.Context) {
	users := utils.GetOnlineUsers()

	// 转换为前端需要的格式
	userList := make([]map[string]interface{}, len(users))
	for i, user := range users {
		userList[i] = map[string]interface{}{
			"id":       user.UserID,
			"username": user.Username,
			"avatar":   user.Avatar,
			"bio":      user.Bio,
			"status":   user.Status,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    userList,
	})
}
