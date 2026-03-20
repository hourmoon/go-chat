package routes

import (
	"encoding/json"
	"fmt"
	"go-chat/models"
	"go-chat/utils"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// SaveOfflineMessage 将消息保存为离线消息
// 参数：messageID - 消息ID，receiverID - 接收者ID
// 返回：错误信息
func SaveOfflineMessage(messageID uint64, receiverID uint) error {
	// 检查消息是否存在
	var message models.Message
	if err := models.DB.First(&message, messageID).Error; err != nil {
		log.Printf("❌ 消息不存在: messageID=%d, 错误: %v", messageID, err)
		return fmt.Errorf("消息不存在: %v", err)
	}

	// 检查接收者是否存在
	var receiver models.User
	if err := models.DB.First(&receiver, receiverID).Error; err != nil {
		log.Printf("❌ 接收者不存在: receiverID=%d, 错误: %v", receiverID, err)
		return fmt.Errorf("接收者不存在: %v", err)
	}

	// 创建离线消息记录
	offlineMessage := models.OfflineMessage{
		MessageID:  messageID,
		ReceiverID: receiverID,
		IsRead:     0, // 0表示未推送
	}

	// 插入离线消息
	if err := models.DB.Create(&offlineMessage).Error; err != nil {
		log.Printf("❌ 保存离线消息失败: messageID=%d, receiverID=%d, 错误: %v", messageID, receiverID, err)
		return fmt.Errorf("保存离线消息失败: %v", err)
	}

	log.Printf("✅ 离线消息保存成功: messageID=%d, receiverID=%d, offlineMessageID=%d", messageID, receiverID, offlineMessage.ID)
	return nil
}

// SaveOfflineMessagesForGroup 为群组中的所有离线成员保存离线消息
// 参数：messageID - 消息ID，groupID - 群组ID，senderID - 发送者ID
// 返回：错误信息
func SaveOfflineMessagesForGroup(messageID uint64, groupID uint, senderID uint) error {
	// 获取群组的所有成员
	var groupMembers []models.GroupMember
	if err := models.DB.Where("group_id = ?", groupID).Find(&groupMembers).Error; err != nil {
		log.Printf("❌ 获取群组成员失败: groupID=%d, 错误: %v", groupID, err)
		return fmt.Errorf("获取群组成员失败: %v", err)
	}

	if len(groupMembers) == 0 {
		log.Printf("ℹ️  群组没有成员: groupID=%d", groupID)
		return nil
	}

	// 过滤出离线的成员（除了发送者）
	var offlineMessages []models.OfflineMessage
	for _, member := range groupMembers {
		// 跳过发送者
		if member.UserID == senderID {
			continue
		}

		// 检查成员是否离线
		if !IsUserOnline(member.UserID) {
			offlineMessages = append(offlineMessages, models.OfflineMessage{
				MessageID:  messageID,
				ReceiverID: member.UserID,
				IsRead:     0, // 0表示未推送
			})
		}
	}

	if len(offlineMessages) == 0 {
		log.Printf("ℹ️  群组中没有离线成员: groupID=%d, senderID=%d", groupID, senderID)
		return nil
	}

	// 使用事务确保数据一致性
	tx := models.DB.Begin()
	if tx.Error != nil {
		log.Printf("❌ 开启事务失败: %v", tx.Error)
		return fmt.Errorf("开启事务失败: %v", tx.Error)
	}

	// 批量插入离线消息
	if err := tx.CreateInBatches(offlineMessages, 100).Error; err != nil {
		tx.Rollback()
		log.Printf("❌ 批量保存离线消息失败: groupID=%d, messageID=%d, 错误: %v", groupID, messageID, err)
		return fmt.Errorf("批量保存离线消息失败: %v", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		log.Printf("❌ 事务提交失败: %v", err)
		return fmt.Errorf("事务提交失败: %v", err)
	}

	log.Printf("✅ 群组离线消息保存成功: groupID=%d, messageID=%d, 离线成员数=%d", groupID, messageID, len(offlineMessages))
	return nil
}

// IsUserOnline 检查用户是否在线
// 参数：userID - 用户ID
// 返回：是否在线
func IsUserOnline(userID uint) bool {
	status := utils.GetUserStatus(userID)
	isOnline := status == "online"
	log.Printf("ℹ️  检查用户在线状态: userID=%d, status=%s, isOnline=%v", userID, status, isOnline)
	return isOnline
}

// OfflineMessageResponse 离线消息推送格式
type OfflineMessageResponse struct {
	Type        string `json:"type"`
	ID          uint64 `json:"id"`
	UserID      uint   `json:"user_id"`
	Username    string `json:"username"`
	Content     string `json:"content"`
	MessageType string `json:"message_type"`
	FileURL     string `json:"file_url"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
	Target      uint   `json:"target"`
	GroupID     uint   `json:"group_id"`
	CreatedAt   string `json:"created_at"`
}

// PushOfflineMessages 推送用户的所有离线消息
// 参数：userID - 用户ID，conn - WebSocket连接
// 返回：错误信息
func PushOfflineMessages(userID uint, conn *websocket.Conn) error {
	// 查询该用户的所有离线消息，按created_at排序
	var offlineMessages []struct {
		OfflineMessageID uint64
		Message          models.Message
	}

	if err := models.DB.
		Table("offline_messages").
		Select("offline_messages.id as offline_message_id, messages.*").
		Joins("LEFT JOIN messages ON offline_messages.message_id = messages.id").
		Where("offline_messages.receiver_id = ?", userID).
		Order("offline_messages.created_at ASC").
		Scan(&offlineMessages).Error; err != nil {
		log.Printf("❌ 查询离线消息失败: userID=%d, 错误: %v", userID, err)
		return fmt.Errorf("查询离线消息失败: %v", err)
	}

	if len(offlineMessages) == 0 {
		log.Printf("ℹ️  用户没有离线消息: userID=%d", userID)
		return nil
	}

	log.Printf("📥 开始推送离线消息: userID=%d, 消息数=%d", userID, len(offlineMessages))

	// 收集要删除的离线消息ID
	var offlineMessageIDs []uint64
	successCount := 0

	// 逐条推送离线消息
	for _, item := range offlineMessages {
		msg := item.Message
		offlineMessageIDs = append(offlineMessageIDs, item.OfflineMessageID)

		// 构建离线消息响应
		response := OfflineMessageResponse{
			Type:        "offline_message",
			ID:          uint64(msg.ID),
			UserID:      msg.UserID,
			Username:    msg.Username,
			Content:     msg.Content,
			MessageType: msg.MessageType,
			FileURL:     msg.FileURL,
			FileName:    msg.FileName,
			FileSize:    msg.FileSize,
			Target:      0, // 离线消息不需要target
			GroupID:     msg.GroupID,
			CreatedAt:   msg.CreatedAt.Format("2006-01-02 15:04:05"),
		}

		// 序列化为JSON
		responseJSON, err := json.Marshal(response)
		if err != nil {
			log.Printf("❌ JSON编码离线消息失败: messageID=%d, 错误: %v", msg.ID, err)
			continue
		}

		// 推送给客户端
		if err := conn.WriteMessage(websocket.TextMessage, responseJSON); err != nil {
			log.Printf("❌ 推送离线消息失败: userID=%d, messageID=%d, 错误: %v", userID, msg.ID, err)
			return fmt.Errorf("推送离线消息失败: %v", err)
		}

		log.Printf("✅ 离线消息已推送: userID=%d, messageID=%d, 发送者=%s", userID, msg.ID, msg.Username)
		successCount++
	}

	// 删除已推送的离线消息记录
	if len(offlineMessageIDs) > 0 {
		if err := models.DB.Where("id IN ?", offlineMessageIDs).Delete(&models.OfflineMessage{}).Error; err != nil {
			log.Printf("❌ 删除离线消息记录失败: userID=%d, 错误: %v", userID, err)
			return fmt.Errorf("删除离线消息记录失败: %v", err)
		}
		log.Printf("✅ 离线消息记录已删除: userID=%d, 删除数=%d", userID, len(offlineMessageIDs))
	}

	log.Printf("✅ 离线消息推送完成: userID=%d, 成功推送=%d条", userID, successCount)
	return nil
}

// CleanOldOfflineMessages 清理7天前的离线消息
// 返回：删除的记录数和错误信息
func CleanOldOfflineMessages() (int64, error) {
	// 计算7天前的时间
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	log.Printf("📝 开始清理7天前的离线消息: 时间阈值=%s", sevenDaysAgo.Format("2006-01-02 15:04:05"))

	// 删除7天前的离线消息
	result := models.DB.Where("created_at < ?", sevenDaysAgo).Delete(&models.OfflineMessage{})
	if result.Error != nil {
		log.Printf("❌ 清理离线消息失败: 错误: %v", result.Error)
		return 0, fmt.Errorf("清理离线消息失败: %v", result.Error)
	}

	deletedCount := result.RowsAffected
	log.Printf("✅ 离线消息清理完成: 删除数=%d条", deletedCount)
	return deletedCount, nil
}

// StartOfflineMessageCleanupScheduler 启动定期清理协程
// 每天凌晨2点执行一次清理
func StartOfflineMessageCleanupScheduler() {
	go func() {
		for {
			// 计算下次执行时间（明天凌晨2点）
			now := time.Now()
			nextRun := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())

			// 如果当前时间已经过了凌晨2点，则设置为明天凌晨2点
			if now.After(nextRun) {
				nextRun = nextRun.AddDate(0, 0, 1)
			}

			// 计算等待时间
			waitDuration := nextRun.Sub(now)
			log.Printf("📅 离线消息清理计划: 下次执行时间=%s, 等待时间=%v", nextRun.Format("2006-01-02 15:04:05"), waitDuration)

			// 等待到执行时间
			time.Sleep(waitDuration)

			// 执行清理
			deletedCount, err := CleanOldOfflineMessages()
			if err != nil {
				log.Printf("❌ 定期清理离线消息失败: %v", err)
			} else {
				log.Printf("✅ 定期清理离线消息成功: 删除数=%d条", deletedCount)
			}
		}
	}()
	log.Printf("✅ 离线消息清理计划已启动")
}

// OfflineMessageQueryResponse 离线消息查询响应格式
type OfflineMessageQueryResponse struct {
	ID          uint64 `json:"id"`
	UserID      uint   `json:"user_id"`
	Username    string `json:"username"`
	Content     string `json:"content"`
	MessageType string `json:"message_type"`
	FileURL     string `json:"file_url"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
	Target      uint   `json:"target"`
	GroupID     uint   `json:"group_id"`
	CreatedAt   string `json:"created_at"`
}

// GetOfflineMessages 获取用户的离线消息（支持分页）
// 路由：GET /offline-messages
// 认证：需要JWT认证
func GetOfflineMessages(c *gin.Context) {
	// 从JWT Token中获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "未授权，请先登录",
		})
		return
	}

	currentUserID := userID.(uint)
	log.Printf("📥 查询离线消息: userID=%d", currentUserID)

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

	// 查询该用户的离线消息总数
	var total int64
	if err := models.DB.Model(&models.OfflineMessage{}).
		Where("receiver_id = ?", currentUserID).
		Count(&total).Error; err != nil {
		log.Printf("❌ 查询离线消息总数失败: userID=%d, 错误: %v", currentUserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "查询离线消息总数失败",
		})
		return
	}

	// 查询该用户的离线消息，按created_at降序排列
	var offlineMessages []struct {
		OfflineMessageID uint64
		Message          models.Message
	}

	if err := models.DB.
		Table("offline_messages").
		Select("offline_messages.id as offline_message_id, messages.*").
		Joins("LEFT JOIN messages ON offline_messages.message_id = messages.id").
		Where("offline_messages.receiver_id = ?", currentUserID).
		Order("offline_messages.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Scan(&offlineMessages).Error; err != nil {
		log.Printf("❌ 查询离线消息失败: userID=%d, 错误: %v", currentUserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "查询离线消息失败",
		})
		return
	}

	// 构建响应数据
	var data []OfflineMessageQueryResponse
	for _, item := range offlineMessages {
		msg := item.Message
		data = append(data, OfflineMessageQueryResponse{
			ID:          uint64(msg.ID),
			UserID:      msg.UserID,
			Username:    msg.Username,
			Content:     msg.Content,
			MessageType: msg.MessageType,
			FileURL:     msg.FileURL,
			FileName:    msg.FileName,
			FileSize:    msg.FileSize,
			Target:      0,
			GroupID:     msg.GroupID,
			CreatedAt:   msg.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	// 计算总页数
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	log.Printf("✅ 离线消息查询成功: userID=%d, 返回数=%d, 总数=%d", currentUserID, len(data), total)

	// 返回分页响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
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

// GetOfflineMessageCount 获取用户的离线消息数量
// 路由：GET /offline-messages/count
// 认证：需要JWT认证
func GetOfflineMessageCount(c *gin.Context) {
	// 从JWT Token中获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "未授权，请先登录",
		})
		return
	}

	currentUserID := userID.(uint)
	log.Printf("📊 查询离线消息数量: userID=%d", currentUserID)

	// 统计该用户的离线消息数量
	var count int64
	if err := models.DB.Model(&models.OfflineMessage{}).
		Where("receiver_id = ?", currentUserID).
		Count(&count).Error; err != nil {
		log.Printf("❌ 查询离线消息数量失败: userID=%d, 错误: %v", currentUserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "查询离线消息数量失败",
		})
		return
	}

	log.Printf("✅ 离线消息数量查询成功: userID=%d, count=%d", currentUserID, count)

	// 构建消息提示
	message := ""
	if count == 0 {
		message = "您没有未读离线消息"
	} else if count == 1 {
		message = "您有1条未读离线消息"
	} else {
		message = fmt.Sprintf("您有%d条未读离线消息", count)
	}

	// 返回统计结果
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   count,
		"message": message,
	})
}
