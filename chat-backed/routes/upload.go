package routes

import (
	"fmt"
	"go-chat/utils"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

// 定义允许的文件类型和最大文件大小
const (
	maxUploadSize = 10 * 1024 * 1024 // 10MB
	uploadPath    = "./uploads"
)

// 初始化上传目录
// init 函数在程序启动时自动执行，用于进行初始化操作
func init() {
	// 检查上传路径是否存在，如果不存在则创建
	// os.Stat 函数用于获取文件或目录的信息
	// 如果返回的错误类型为 os.IsNotExist，则表示目录不存在
	if _, err := os.Stat(uploadPath); os.IsNotExist(err) {
		// 使用 MkdirAll 创建目录，包括所有必要的父目录
		// os.ModePerm 表示设置权限为 0777，即所有用户都有读、写、执行权限
		os.MkdirAll(uploadPath, os.ModePerm)
	}
}

// UploadFile 处理文件上传
func UploadFile(c *gin.Context) {
	// 验证用户身份
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
		return
	}

	// 去掉 "Bearer " 前缀
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	claims, err := utils.ParseJWT(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的令牌"})
		return
	}

	// 解析多部分表单
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法获取文件"})
		return
	}

	// 检查文件大小
	if file.Size > maxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件太大，最大支持10MB"})
		return
	}

	// 生成唯一文件名
	ext := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("%d_%d%s", claims.UserID, time.Now().UnixNano(), ext)
	filePath := filepath.Join(uploadPath, fileName)

	// 保存文件
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}

	// 返回文件信息
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"file_url":  "/uploads/" + fileName,
		"file_name": file.Filename,
		"file_size": file.Size,
	})
}

// 服务静态文件
func ServeFile(c *gin.Context) {
	fileName := c.Param("filename")
	filePath := filepath.Join(uploadPath, fileName)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	c.File(filePath)
}
