package routes

import (
	"fmt"
	"go-chat/models"
	"go-chat/utils"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

// 获取用户资料
func GetProfile(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var user models.User
	if err := models.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"avatar":    user.Avatar,
			"bio":       user.Bio,
			"status":    user.Status,
			"last_seen": user.LastSeen,
		},
	})
}

// 更新用户资料
func UpdateProfile(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var req struct {
		Bio string `json:"bio"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 更新用户资料
	if err := models.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"bio":        req.Bio,
		"updated_at": time.Now(),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "资料更新成功"})
}

// 上传用户头像
func UploadAvatar(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法获取头像文件"})
		return
	}

	// 检查文件类型
	ext := filepath.Ext(file.Filename)
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true}
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件格式"})
		return
	}

	// 生成文件名
	fileName := fmt.Sprintf("avatar_%d_%d%s", userID, time.Now().UnixNano(), ext)
	filePath := filepath.Join("./uploads/avatars", fileName)

	// 确保目录存在
	if err := os.MkdirAll("./uploads/avatars", os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建目录失败"})
		return
	}

	// 保存文件
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存头像失败"})
		return
	}

	// 更新用户头像
	avatarURL := "/uploads/avatars/" + fileName
	if err := models.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"avatar":     avatarURL,
		"updated_at": time.Now(),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新头像失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"avatar":  avatarURL,
		"message": "头像上传成功",
	})
}

// 更新用户状态
func UpdateUserStatus(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var req struct {
		Status string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 验证状态值
	validStatuses := map[string]bool{
		"online":  true,
		"offline": true,
		"busy":    true,
		"away":    true,
	}

	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的状态值"})
		return
	}

	// 更新用户状态
	if err := models.DB.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"status":     req.Status,
		"updated_at": time.Now(),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新状态失败"})
		return
	}

	// 如果用户在线，更新内存中的状态
	utils.UpdateUserStatus(userID, req.Status)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "状态更新成功"})
}
