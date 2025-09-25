package routes

import (
	"go-chat/models"
	"go-chat/utils"
	"net/http"
	"strings"

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

// SearchUsers 按用户名模糊搜索用户
func SearchUsers(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "keyword 不能为空"})
		return
	}

	// 限制长度，避免滥用
	if len(keyword) > 50 {
		keyword = keyword[:50]
	}

	// 执行模糊查询
	var users []models.User
	if err := models.DB.
		Select("id, username, avatar, status").
		Where("username LIKE ?", "%"+keyword+"%").
		Limit(20).
		Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
		return
	}

	// 统一返回结构
	list := make([]gin.H, 0, len(users))
	for _, u := range users {
		list = append(list, gin.H{
			"id":       u.ID,
			"username": u.Username,
			"avatar":   u.Avatar,
			"status":   u.Status,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"users": list,
	})
}
