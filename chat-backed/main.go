package main

import (
	"go-chat/middleware"
	"go-chat/models"
	"go-chat/routes"
	"go-chat/utils"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("警告: 无法加载 .env 文件:", err)
	}

	checkEnvVariables()
	models.InitDB()

	go routes.HandleMessages() // 启动广播协程
	utils.InitOnlineUsers()    // 启动在线用户清理协程

	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	r.Use(func(c *gin.Context) {
		log.Printf("收到请求: %s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
	})

	// 路由绑定
	r.POST("/register", routes.Register)
	r.POST("/login", routes.Login)
	r.GET("/ws", routes.WSHandler)
	r.GET("/messages", routes.GetMessages)
	r.GET("/online-users", routes.GetOnlineUsers)
	// 添加文件上传路由
	r.POST("/upload", middleware.JWTAuthMiddleware(), routes.UploadFile)
	r.GET("/uploads/:filename", routes.ServeFile)

	// 添加新的路由
	// 用户资料路由
	r.GET("/profile", middleware.JWTAuthMiddleware(), routes.GetProfile)
	r.PUT("/profile", middleware.JWTAuthMiddleware(), routes.UpdateProfile)
	r.POST("/profile/avatar", middleware.JWTAuthMiddleware(), routes.UploadAvatar)
	r.PUT("/profile/status", middleware.JWTAuthMiddleware(), routes.UpdateUserStatus)

	// 好友系统路由
	r.GET("/friends", middleware.JWTAuthMiddleware(), routes.GetFriends)
	r.GET("/friends/pending", middleware.JWTAuthMiddleware(), routes.GetPendingFriendRequests)
	r.POST("/friends", middleware.JWTAuthMiddleware(), routes.AddFriend)
	r.POST("/friends/:id/action", middleware.JWTAuthMiddleware(), routes.HandleFriendRequest)
	r.DELETE("/friends/:id", middleware.JWTAuthMiddleware(), routes.RemoveFriend)

	// 静态文件服务（添加头像目录）
	r.Static("/uploads/avatars", "./uploads/avatars")

	// 注册群组路由
	routes.GroupsRoutes(r)

	log.Println("服务器启动在 :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}

func checkEnvVariables() {
	requiredEnv := []string{"JWT_SECRET", "DB_USER", "DB_PASS", "DB_HOST", "DB_PORT", "DB_NAME"}
	for _, env := range requiredEnv {
		if os.Getenv(env) == "" {
			log.Fatalf("环境变量 %s 未设置", env)
		}
	}

	if os.Getenv("JWT_SECRET") == "your_jwt_secret_key" {
		log.Fatal("请修改默认的JWT_SECRET，不要使用示例值")
	}
}
