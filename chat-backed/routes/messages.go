package routes

import (
	"go-chat/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetMessages 获取历史消息（支持分页）
func GetMessages(c *gin.Context) {
	// 从查询参数中获取分页参数
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "50")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 50
	}

	// 限制每页最大消息数
	if pageSize > 100 {
		pageSize = 100
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	var messages []models.Message
	var total int64

	// 获取总消息数（仅全局聊天消息，group_id = 0）
	if err := models.DB.Model(&models.Message{}).Where("group_id = 0").Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取消息总数失败"})
		return
	}

	// 按创建时间降序获取消息（仅全局聊天消息，group_id = 0）
	if err := models.DB.Where("group_id = 0").Order("created_at desc").Offset(offset).Limit(pageSize).Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取消息失败"})
		return
	}

	// 反转消息顺序，使最新的消息在最后
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	// 计算总页数
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	// 返回分页响应
	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"pagination": gin.H{
			"page":       page,
			"pageSize":   pageSize,
			"total":      total,
			"totalPages": totalPages,
			"hasNext":    page < totalPages,
			"hasPrev":    page > 1,
		},
	})
}
