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
