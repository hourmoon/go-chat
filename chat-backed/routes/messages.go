package routes

import (
	"go-chat/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetMessages 获取历史消息
func GetMessages(c *gin.Context) {
	// 从查询参数中获取limit，默认50条
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	var messages []models.Message
	// 按创建时间降序获取消息，然后反转顺序，使得最新的消息在最后
	if err := models.DB.Order("created_at desc").Limit(limit).Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取消息失败"})
		return
	}

	// 反转消息顺序，使得最早的消息在前，最新的在后
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	c.JSON(http.StatusOK, messages)
}
