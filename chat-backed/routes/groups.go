package routes

import (
	"go-chat/middleware"
	"go-chat/models"
	"go-chat/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GroupsRoutes 注册群组相关路由
func GroupsRoutes(r *gin.Engine) {
	// 创建一个新的群组路由，并应用中间件
	groups := r.Group("/groups").Use(middleware.JWTAuthMiddleware())
	{
		// 根路由兼容，无斜杠
		groups.GET("", getUserGroups)
		groups.POST("", createGroup)

		// 群组管理路由
		groups.POST("/", createGroup)       // 创建群组
		groups.GET("/", getUserGroups)      // 获取用户所在的所有群组
		groups.GET("/:id", getGroupDetails) // 获取特定群组的详细信息
		groups.PUT("/:id", updateGroup)     // 更新特定群组的信息
		groups.DELETE("/:id", deleteGroup)  // 解散特定群组

		// 群成员管理路由
		groups.GET("/:id/members", getGroupMembers)               // 获取群成员列表
		groups.POST("/:id/members", addGroupMember)               // 添加（邀请）新成员
		groups.DELETE("/:id/members/:userId", removeGroupMember)  // 移除群成员
		groups.PUT("/:id/members/:userId/role", updateMemberRole) // 修改成员角色
		groups.POST("/:id/transfer-owner", transferOwner)         // 转让群主

		// 群消息管理路由
		groups.GET("/:id/messages", getGroupMessages)       // 获取群聊消息历史
		groups.GET("/:id/online-members", getOnlineMembers) // 获取群在线成员
	}
}

// createGroup 创建群组
func createGroup(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Avatar      string `json:"avatar"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 从上下文获取用户ID
	userID := c.MustGet("userID").(uint)

	// 开始数据库事务
	tx := models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建群组记录
	group := models.Group{
		Name:        req.Name,
		OwnerID:     userID,
		Avatar:      req.Avatar,
		Description: req.Description,
	}

	if err := tx.Create(&group).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建群组失败"})
		return
	}

	// 将创建者加入群组，设置为群主
	groupMember := models.GroupMember{
		UserID:  userID,
		GroupID: group.ID,
		Role:    "owner",
	}

	if err := tx.Create(&groupMember).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加群主失败"})
		return
	}

	// 提交事务
	tx.Commit()

	c.JSON(http.StatusCreated, gin.H{
		"message": "群组创建成功",
		"group":   group,
	})
}

// getUserGroups 获取用户所在的所有群组
func getUserGroups(c *gin.Context) {
	// 从上下文获取用户ID
	userID := c.MustGet("userID").(uint)

	var groups []models.Group

	// 查询用户所在的群组
	err := models.DB.Joins("JOIN group_members ON groups.id = group_members.group_id").
		Where("group_members.user_id = ?", userID).
		Find(&groups).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取群组列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"groups": groups,
	})
}

// getGroupDetails 获取特定群组的详细信息
func getGroupDetails(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组ID"})
		return
	}

	var group models.Group
	err = models.DB.Preload("Owner").First(&group, uint(groupID)).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "群组不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取群组详情失败"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"group": group,
	})
}

// updateGroup 更新特定群组的信息
func updateGroup(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组ID"})
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Avatar      string `json:"avatar"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 从上下文获取用户ID
	userID := c.MustGet("userID").(uint)

	// 查找群组
	var group models.Group
	if err := models.DB.First(&group, uint(groupID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "群组不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询群组失败"})
		}
		return
	}

	// 检查操作者权限（仅群主和管理员可以更新群组信息）
	var member models.GroupMember
	if err := models.DB.Where("user_id = ? AND group_id = ?", userID, uint(groupID)).First(&member).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusForbidden, gin.H{"error": "您不是该群组成员"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询权限失败"})
		}
		return
	}

	if member.Role != "owner" && member.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅群主和管理员可以更新群组信息"})
		return
	}

	// 更新群组信息
	updateData := models.Group{}
	if req.Name != "" {
		updateData.Name = req.Name
	}
	if req.Description != "" {
		updateData.Description = req.Description
	}
	if req.Avatar != "" {
		updateData.Avatar = req.Avatar
	}

	if err := models.DB.Model(&group).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新群组失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "群组更新成功",
		"group":   group,
	})
}

// deleteGroup 解散特定群组
func deleteGroup(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组ID"})
		return
	}

	// 从上下文获取用户ID
	userID := c.MustGet("userID").(uint)

	// 开始数据库事务
	tx := models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查群组是否存在
	var group models.Group
	if err := tx.First(&group, uint(groupID)).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "群组不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询群组失败"})
		}
		return
	}

	// 检查操作者权限（仅群主可以解散群组）
	var member models.GroupMember
	if err := tx.Where("user_id = ? AND group_id = ?", userID, uint(groupID)).First(&member).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusForbidden, gin.H{"error": "您不是该群组成员"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询权限失败"})
		}
		return
	}

	if member.Role != "owner" {
		tx.Rollback()
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅群主可以解散群组"})
		return
	}

	// 删除所有群成员记录
	if err := tx.Where("group_id = ?", uint(groupID)).Delete(&models.GroupMember{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除群成员失败"})
		return
	}

	// 删除群组记录
	if err := tx.Delete(&group).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除群组失败"})
		return
	}

	// 提交事务
	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "群组解散成功",
	})
}

// getGroupMembers 获取群成员列表
func getGroupMembers(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组ID"})
		return
	}

	var members []models.GroupMember
	err = models.DB.Preload("User").Where("group_id = ?", uint(groupID)).Find(&members).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取群成员失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"members": members,
	})
}

// addGroupMember 添加（邀请）新成员
func addGroupMember(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组ID"})
		return
	}

	// 从上下文获取邀请者用户ID
	inviterUserID := c.MustGet("userID").(uint)

	// 校验邀请者必须是该群成员
	var inviterMember models.GroupMember
	if err := models.DB.Where("user_id = ? AND group_id = ?", inviterUserID, uint(groupID)).First(&inviterMember).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusForbidden, gin.H{"error": "您不是该群组成员，无权邀请新成员"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "验证邀请者权限失败"})
		}
		return
	}

	var req struct {
		UserID uint   `json:"user_id" binding:"required"`
		Role   string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 设置默认角色
	if req.Role == "" {
		req.Role = "member"
	}

	// 邀请成员权限强化：当req.Role == "admin"时，仅允许当前操作者角色为owner
	if req.Role == "admin" && inviterMember.Role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅群主可授予管理员"})
		return
	}

	// 检查被邀请用户是否存在
	var targetUser models.User
	if err := models.DB.First(&targetUser, req.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "被邀请的用户不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
		}
		return
	}

	// 检查用户是否已经在群组中
	var existingMember models.GroupMember
	if err := models.DB.Where("user_id = ? AND group_id = ?", req.UserID, uint(groupID)).First(&existingMember).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户已经在群组中"})
		return
	}

	// 创建群成员记录
	member := models.GroupMember{
		UserID:  req.UserID,
		GroupID: uint(groupID),
		Role:    req.Role,
	}

	if err := models.DB.Create(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加群成员失败"})
		return
	}

	// 发送群成员加入通知
	memberJoinedMsg := BroadcastMessage{
		Type:      "group_member_joined",
		UserID:    req.UserID,
		Username:  targetUser.Username,
		Content:   targetUser.Username + " 加入了群组",
		GroupID:   uint(groupID),
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}
	SendBroadcastMessage(memberJoinedMsg)

	c.JSON(http.StatusCreated, gin.H{
		"message": "成员添加成功",
		"member":  member,
	})
}

// removeGroupMember 移除群成员
func removeGroupMember(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组ID"})
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 从上下文获取操作者用户ID
	operatorUserID := c.MustGet("userID").(uint)

	// 查找操作者在群组中的权限
	var operatorMember models.GroupMember
	if err := models.DB.Where("user_id = ? AND group_id = ?", operatorUserID, uint(groupID)).First(&operatorMember).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusForbidden, gin.H{"error": "您不是该群组成员"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询权限失败"})
		}
		return
	}

	// 检查操作者权限（仅群主和管理员可以移除成员）
	if operatorMember.Role != "owner" && operatorMember.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅群主和管理员可以移除成员"})
		return
	}

	// 查找要移除的群成员记录并预加载用户信息用于通知
	var member models.GroupMember
	if err := models.DB.Preload("User").Where("user_id = ? AND group_id = ?", uint(targetUserID), uint(groupID)).First(&member).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "成员不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询成员失败"})
		}
		return
	}

	// 检查是否是群主，群主不能被移除（需要先转让群组）
	if member.Role == "owner" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "群主不能被移除，请先转让群组"})
		return
	}

	// 管理员不能移除其他管理员，只有群主可以
	if operatorMember.Role == "admin" && member.Role == "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "管理员无权移除其他管理员"})
		return
	}

	if err := models.DB.Delete(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "移除成员失败"})
		return
	}

	// 发送群成员离开通知
	memberLeftMsg := BroadcastMessage{
		Type:      "group_member_left",
		UserID:    uint(targetUserID),
		Username:  member.User.Username,
		Content:   member.User.Username + " 离开了群组",
		GroupID:   uint(groupID),
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}
	SendBroadcastMessage(memberLeftMsg)

	c.JSON(http.StatusOK, gin.H{
		"message": "成员移除成功",
	})
}

// getGroupMessages 获取群聊消息历史
func getGroupMessages(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组ID"})
		return
	}

	// 从上下文获取用户ID
	userID := c.MustGet("userID").(uint)

	// 校验当前用户是否为该群成员
	var member models.GroupMember
	if err := models.DB.Where("user_id = ? AND group_id = ?", userID, uint(groupID)).First(&member).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusForbidden, gin.H{"error": "您不是该群组成员"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "验证群组成员身份失败"})
		}
		return
	}

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

	// 获取群组消息总数
	if err := models.DB.Model(&models.Message{}).Where("group_id = ?", uint(groupID)).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取消息总数失败"})
		return
	}

	// 按创建时间降序获取群组消息
	if err := models.DB.Where("group_id = ?", uint(groupID)).Order("created_at desc").Offset(offset).Limit(pageSize).Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取群组消息失败"})
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

// getOnlineMembers 获取群在线成员
func getOnlineMembers(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组ID"})
		return
	}

	// 从上下文获取用户ID
	userID := c.MustGet("userID").(uint)

	// 校验当前用户是否为该群成员
	var member models.GroupMember
	if err := models.DB.Where("user_id = ? AND group_id = ?", userID, uint(groupID)).First(&member).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusForbidden, gin.H{"error": "您不是该群组成员"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "验证群组成员身份失败"})
		}
		return
	}

	// 查询群成员ID列表
	var memberIDs []uint
	if err := models.DB.Model(&models.GroupMember{}).Where("group_id = ?", uint(groupID)).Pluck("user_id", &memberIDs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询群成员失败"})
		return
	}

	// 构建在线成员列表
	var onlineMembers []gin.H
	utils.OnlineUsers.RLock()
	for _, memberID := range memberIDs {
		if userState, exists := utils.OnlineUsers.Users[memberID]; exists && len(userState.Connections) > 0 {
			// 获取用户详细信息
			var userInfo models.User
			if err := models.DB.First(&userInfo, memberID).Error; err == nil {
				onlineMembers = append(onlineMembers, gin.H{
					"id":       userInfo.ID,
					"username": userInfo.Username,
					"avatar":   userInfo.Avatar,
					"status":   userState.Status,
				})
			}
		}
	}
	utils.OnlineUsers.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"onlineMembers": onlineMembers,
		"count":         len(onlineMembers),
	})
}

// updateMemberRole 修改成员角色
func updateMemberRole(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组ID"})
		return
	}

	targetUserID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 从上下文获取操作者用户ID
	operatorUserID := c.MustGet("userID").(uint)

	var req struct {
		Role string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 校验role参数
	if req.Role != "admin" && req.Role != "member" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色只允许admin或member"})
		return
	}

	// 查找操作者在群组中的权限
	var operatorMember models.GroupMember
	if err := models.DB.Where("user_id = ? AND group_id = ?", operatorUserID, uint(groupID)).First(&operatorMember).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusForbidden, gin.H{"error": "您不是该群组成员"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询权限失败"})
		}
		return
	}

	// 仅群主可操作
	if operatorMember.Role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅群主可操作"})
		return
	}

	// 查找要修改的群成员记录
	var targetMember models.GroupMember
	if err := models.DB.Preload("User").Where("user_id = ? AND group_id = ?", uint(targetUserID), uint(groupID)).First(&targetMember).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "成员不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询成员失败"})
		}
		return
	}

	// 不可修改群主
	if targetMember.Role == "owner" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不可修改群主角色"})
		return
	}

	// 更新成员角色
	if err := models.DB.Model(&targetMember).Update("role", req.Role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新成员角色失败"})
		return
	}

	// 重新加载更新后的数据
	if err := models.DB.Preload("User").Where("user_id = ? AND group_id = ?", uint(targetUserID), uint(groupID)).First(&targetMember).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取更新后成员信息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "成员角色更新成功",
		"member":  targetMember,
	})
}

// transferOwner 转让群主
func transferOwner(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的群组ID"})
		return
	}

	// 从上下文获取操作者用户ID
	operatorUserID := c.MustGet("userID").(uint)

	var req struct {
		TargetUserID uint `json:"target_user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 开始数据库事务
	tx := models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 查找操作者在群组中的权限
	var operatorMember models.GroupMember
	if err := tx.Where("user_id = ? AND group_id = ?", operatorUserID, uint(groupID)).First(&operatorMember).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusForbidden, gin.H{"error": "您不是该群组成员"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询权限失败"})
		}
		return
	}

	// 仅群主可操作
	if operatorMember.Role != "owner" {
		tx.Rollback()
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅群主可操作"})
		return
	}

	// 查找目标用户在群组中的记录
	var targetMember models.GroupMember
	if err := tx.Where("user_id = ? AND group_id = ?", req.TargetUserID, uint(groupID)).First(&targetMember).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "目标用户不在群中"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询目标用户失败"})
		}
		return
	}

	// 更新群组的owner_id
	if err := tx.Model(&models.Group{}).Where("id = ?", uint(groupID)).Update("owner_id", req.TargetUserID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新群主失败"})
		return
	}

	// 将目标成员设置为owner
	if err := tx.Model(&targetMember).Update("role", "owner").Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新目标成员角色失败"})
		return
	}

	// 将原群主设置为admin
	if err := tx.Model(&operatorMember).Update("role", "admin").Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新原群主角色失败"})
		return
	}

	// 提交事务
	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "群主转让成功",
	})
}
