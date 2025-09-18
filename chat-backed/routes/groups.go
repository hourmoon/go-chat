package routes

import (
	"go-chat/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GroupsRoutes 注册群组相关路由
func GroupsRoutes(r *gin.Engine) {
	// 创建群组路由分组，应用JWT认证中间件
	groupRoutes := r.Group("/api/groups")
	// groupRoutes.Use(middleware.JWTAuthMiddleware()) // 暂时注释，后续启用

	// 群组管理路由
	groupRoutes.POST("/", createGroup)       // 创建群组
	groupRoutes.GET("/", getUserGroups)      // 获取用户所在的所有群组
	groupRoutes.GET("/:id", getGroupDetails) // 获取特定群组的详细信息
	groupRoutes.PUT("/:id", updateGroup)     // 更新特定群组的信息
	groupRoutes.DELETE("/:id", deleteGroup)  // 解散特定群组

	// 群成员管理路由
	groupRoutes.GET("/:id/members", getGroupMembers)              // 获取群成员列表
	groupRoutes.POST("/:id/members", addGroupMember)              // 添加（邀请）新成员
	groupRoutes.DELETE("/:id/members/:userId", removeGroupMember) // 移除群成员
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

	// 从上下文获取用户ID (暂时使用固定值，实际使用时从JWT获取)
	// userID := c.GetUint("userID")
	userID := uint(1) // 临时固定值，用于测试

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
	// 从上下文获取用户ID (暂时使用固定值)
	// userID := c.GetUint("userID")
	userID := uint(1) // 临时固定值

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

	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 查找并删除群成员记录
	var member models.GroupMember
	if err := models.DB.Where("user_id = ? AND group_id = ?", uint(userID), uint(groupID)).First(&member).Error; err != nil {
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

	if err := models.DB.Delete(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "移除成员失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "成员移除成功",
	})
}
