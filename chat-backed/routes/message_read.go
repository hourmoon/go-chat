package routes

import (
	"encoding/json"
	"go-chat/models"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// MarkMessageAsRead ?????????
func MarkMessageAsRead(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	messageIDStr := c.Param("id")

	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "?????ID"})
		return
	}

	// ????????
	var message models.Message
	if err := models.DB.First(&message, messageID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "??????"})
		}
		return
	}

	// ??????????????
	if message.UserID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "????????????"})
		return
	}

	// ??????????????????????
	readStatus := models.MessageReadStatus{
		MessageID: uint(messageID),
		ReaderID:  userID,
		ReadAt:    time.Now(),
	}

	result := models.DB.Create(&readStatus)
	if result.Error != nil {
		// ?????????????????
		if strings.Contains(result.Error.Error(), "Duplicate") || strings.Contains(result.Error.Error(), "unique") {
			log.Printf("??  ?????????: messageID=%d, userID=%d", messageID, userID)
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "????????",
			})
			return
		}
		log.Printf("? ????????: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "??????"})
		return
	}

	// ?????????
	if err := models.DB.Model(&models.Message{}).
		Where("id = ?", messageID).
		Update("read_count", gorm.Expr("read_count + 1")).Error; err != nil {
		log.Printf("? ????????: %v", err)
	}

	log.Printf("? ????????: messageID=%d, userID=%d", messageID, userID)

	// ?? WebSocket ????????????
	SendReadReceipt(uint(messageID), userID, message.UserID, message.GroupID, message.ReadCount+1)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "????????",
		"message_id": messageID,
		"read_at":    readStatus.ReadAt,
	})
}

// MarkMessagesAsReadBatch ?????????
func MarkMessagesAsReadBatch(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var req struct {
		MessageIDs []uint `json:"message_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "????"})
		return
	}

	if len(req.MessageIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??ID??????"})
		return
	}

	if len(req.MessageIDs) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??????100???"})
		return
	}

	var messages []models.Message
	if err := models.DB.Where("id IN ? AND user_id != ?", req.MessageIDs, userID).Find(&messages).Error; err != nil {
		log.Printf("? ??????: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "??????"})
		return
	}

	if len(messages) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "?????????", "count": 0})
		return
	}

	successCount := 0
	now := time.Now()
	for _, msg := range messages {
		readStatus := models.MessageReadStatus{
			MessageID: msg.ID,
			ReaderID:  userID,
			ReadAt:    now,
		}
		
		result := models.DB.Create(&readStatus)
		if result.Error == nil {
			successCount++
			models.DB.Model(&models.Message{}).Where("id = ?", msg.ID).Update("read_count", gorm.Expr("read_count + 1"))
			SendReadReceipt(msg.ID, userID, msg.UserID, msg.GroupID, msg.ReadCount+1)
		} else if !strings.Contains(result.Error.Error(), "Duplicate") && !strings.Contains(result.Error.Error(), "unique") {
			log.Printf("? ????????: messageID=%d, error=%v", msg.ID, result.Error)
		}
	}

	log.Printf("? ?????????: userID=%d, ??=%d/%d", userID, successCount, len(messages))

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "??????", "count": successCount, "total": len(messages)})
}

// GetUnreadCount ????????
func GetUnreadCount(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var count int64

	subQuery := models.DB.Model(&models.MessageReadStatus{}).Select("message_id").Where("reader_id = ?", userID)

	if err := models.DB.Model(&models.Message{}).
		Where("user_id != ?", userID).
		Where("(target = ? OR group_id IN (SELECT group_id FROM group_members WHERE user_id = ?))", userID, userID).
		Where("id NOT IN (?)", subQuery).
		Count(&count).Error; err != nil {
		log.Printf("? ??????????: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "unread_count": count})
}

// GetMessageReadStatus ?????????
func GetMessageReadStatus(c *gin.Context) {
	messageIDStr := c.Param("id")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "?????ID"})
		return
	}

	var message models.Message
	if err := models.DB.First(&message, messageID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "??????"})
		}
		return
	}

	var readStatuses []models.MessageReadStatus
	if err := models.DB.Where("message_id = ?", messageID).Find(&readStatuses).Error; err != nil {
		log.Printf("? ????????: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????????"})
		return
	}

	readerIDs := make([]uint, len(readStatuses))
	for i, status := range readStatuses {
		readerIDs[i] = status.ReaderID
	}

	var readers []models.User
	if len(readerIDs) > 0 {
		models.DB.Select("id, username, avatar").Where("id IN ?", readerIDs).Find(&readers)
	}

	readList := make([]gin.H, 0)
	for _, status := range readStatuses {
		for _, reader := range readers {
			if reader.ID == status.ReaderID {
				readList = append(readList, gin.H{
					"user_id":  reader.ID,
					"username": reader.Username,
					"avatar":   reader.Avatar,
					"read_at":  status.ReadAt,
				})
				break
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message_id": messageID,
		"read_count": message.ReadCount,
		"read_list":  readList,
	})
}

// SendReadReceipt ???????? WebSocket
func SendReadReceipt(messageID, readerID, senderID, groupID uint, readCount int) {
	readReceiptData := map[string]interface{}{
		"type":       "read_receipt",
		"message_id": messageID,
		"reader_id":  readerID,
		"read_count": readCount,
		"read_at":    time.Now().Format("2006-01-02 15:04:05"),
		"group_id":   groupID,
	}

	jsonData, _ := json.Marshal(readReceiptData)
	
	SendBroadcastMessage(BroadcastMessage{
		Type:      "read_receipt",
		UserID:    readerID,
		Target:    senderID,
		GroupID:   groupID,
		Content:   string(jsonData),
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
	})
}
