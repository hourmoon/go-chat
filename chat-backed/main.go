package main

import (
	"go-chat/middleware"
	"go-chat/models"
	"go-chat/routes"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("警告: 无法加载 .env 文件:", err)
	}

	checkEnvVariables()
	models.InitDB()

	go routes.HandleMessages() // 启动广播协程

	r := gin.Default()

	// 应用 CORS 中间件
	r.Use(middleware.CORSMiddleware())

	// 添加请求日志中间件
	r.Use(func(c *gin.Context) {
		log.Printf("收到请求: %s %s", c.Request.Method, c.Request.URL.Path)
		log.Printf("请求头: %+v", c.Request.Header)
		c.Next()
	})

	// 路由绑定
	r.POST("/register", routes.Register)
	r.POST("/login", routes.Login)
	r.GET("/ws", routes.WSHandler)
	r.GET("/messages", routes.GetMessages)

	log.Println("服务器启动在 :8080")
	log.Println("允许的前端地址: http://localhost:5173")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}

// 检查环境变量
func checkEnvVariables() {
	requiredEnv := []string{"JWT_SECRET", "DB_USER", "DB_PASS", "DB_HOST", "DB_PORT", "DB_NAME"}
	for _, env := range requiredEnv {
		if os.Getenv(env) == "" {
			log.Fatalf("环境变量 %s 未设置", env)
		}
	}

	// 检查是否使用了默认的JWT密钥
	if os.Getenv("JWT_SECRET") == "your_jwt_secret_key" {
		log.Fatal("请修改默认的JWT_SECRET，不要使用示例值")
	}
}
