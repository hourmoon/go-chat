package routes

import (
	"go-chat/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 添加好友
func AddFriend(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var req struct {
		Username string `json:"username"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 查找目标用户
	var targetUser models.User
	if err := models.DB.Where("username = ?", req.Username).First(&targetUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 不能添加自己为好友
	if targetUser.ID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能添加自己为好友"})
		return
	}

	// 检查是否已经是好友
	var existingFriend models.Friendship
	models.DB.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
		userID, targetUser.ID, targetUser.ID, userID).First(&existingFriend)

	if existingFriend.ID != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "已经是好友关系"})
		return
	}

	// 创建好友关系
	friendship := models.Friendship{
		UserID:    userID,
		FriendID:  targetUser.ID,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := models.DB.Create(&friendship).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加好友失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "好友请求已发送"})
}

// 获取好友列表
func GetFriends(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var friendships []models.Friendship
	models.DB.Where("(user_id = ? OR friend_id = ?) AND status = ?",
		userID, userID, "accepted").Find(&friendships)

	friendIDs := make([]uint, 0)
	for _, f := range friendships {
		if f.UserID == userID {
			friendIDs = append(friendIDs, f.FriendID)
		} else {
			friendIDs = append(friendIDs, f.UserID)
		}
	}

	var friends []models.User
	if len(friendIDs) > 0 {
		models.DB.Where("id IN ?", friendIDs).Find(&friends)
	}

	// 格式化好友信息
	friendList := make([]gin.H, 0)
	for _, friend := range friends {
		friendList = append(friendList, gin.H{
			"id":        friend.ID,
			"username":  friend.Username,
			"avatar":    friend.Avatar,
			"bio":       friend.Bio,
			"status":    friend.Status,
			"last_seen": friend.LastSeen,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": friendList})
}

// 处理好友请求
func HandleFriendRequest(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var req struct {
		FriendID uint   `json:"friend_id"`
		Action   string `json:"action"` // accept or reject
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 查找好友关系
	var friendship models.Friendship
	if err := models.DB.Where("user_id = ? AND friend_id = ?", req.FriendID, userID).
		First(&friendship).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "好友请求不存在"})
		return
	}

	// 更新好友关系状态
	status := "accepted"
	if req.Action == "reject" {
		status = "rejected"
	}

	if err := models.DB.Model(&friendship).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "处理好友请求失败"})
		return
	}

	// 如果接受好友请求，创建反向的好友关系
	if status == "accepted" {
		reverseFriendship := models.Friendship{
			UserID:    userID,
			FriendID:  req.FriendID,
			Status:    "accepted",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		models.DB.Create(&reverseFriendship)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "好友请求已处理"})
}

// 删除好友
func RemoveFriend(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	friendID := c.Param("id")

	// 删除好友关系
	if err := models.DB.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
		userID, friendID, friendID, userID).Delete(&models.Friendship{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除好友失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "好友删除成功"})
}

// 获取待处理的好友请求
func GetPendingFriendRequests(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var friendships []models.Friendship
	models.DB.Where("friend_id = ? AND status = ?", userID, "pending").Find(&friendships)

	// 获取发送请求的用户信息
	requestUserIDs := make([]uint, 0)
	for _, f := range friendships {
		requestUserIDs = append(requestUserIDs, f.UserID)
	}

	var requestUsers []models.User
	if len(requestUserIDs) > 0 {
		models.DB.Where("id IN ?", requestUserIDs).Find(&requestUsers)
	}

	// 格式化请求信息
	requestList := make([]gin.H, 0)
	for _, user := range requestUsers {
		requestList = append(requestList, gin.H{
			"id":       user.ID,
			"username": user.Username,
			"avatar":   user.Avatar,
			"bio":      user.Bio,
			"status":   user.Status,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": requestList})
}
