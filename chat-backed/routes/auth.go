package routes

import (
    "go-chat/models"
    "go-chat/utils"
    "net/http"

    "github.com/gin-gonic/gin"
    "golang.org/x/crypto/bcrypt"
)

// Login 登录接口
func Login(c *gin.Context) {
    var req struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
        return
    }

    // 检查用户名和密码
    var user models.User
    if err := models.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "用户不存在"})
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "密码错误"})
        return
    }

    // ✅ 使用统一的 GenerateToken
    tokenString, err := utils.GenerateJWT(user.ID, user.Username)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "生成JWT失败"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "token":    tokenString,
        "username": user.Username,
    })
}

// Register 注册接口
func Register(c *gin.Context) {
    var req struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
        return
    }

    // 检查用户名是否存在
    var existing models.User
    if err := models.DB.Where("username = ?", req.Username).First(&existing).Error; err == nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "用户已存在"})
        return
    }

    // 密码加密
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
        return
    }
    // 存数据库
    user := models.User{Username: req.Username, Password: string(hashedPassword)}
    if err := models.DB.Create(&user).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "注册成功"})
}