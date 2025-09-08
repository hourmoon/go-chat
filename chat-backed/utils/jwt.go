package utils

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

var jwtKey []byte

func init() {
	// 尝试从环境变量获取 JWT_SECRET
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// 尝试从 .env 文件加载
		err := godotenv.Load()
		if err != nil {
			log.Println("警告: 无法加载 .env 文件:", err)
			panic("JWT_SECRET 环境变量未设置。请在 .env 文件中设置 JWT_SECRET 或设置环境变量")
		}

		secret = os.Getenv("JWT_SECRET")
		if secret == "" {
			panic("JWT_SECRET 环境变量未设置。请在 .env 文件中设置 JWT_SECRET")
		}
	}

	// 检查是否是默认值
	if secret == "your_jwt_secret_key" {
		log.Println("警告: 你正在使用默认的 JWT_SECRET，这在生产环境中是不安全的")
	}

	jwtKey = []byte(secret)
	log.Println("JWT 密钥已初始化")
}

type Claims struct {
	UserID   uint   `json:"userID"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func GenerateJWT(userID uint, username string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(jwtKey)
	return signed, err
}

func ParseJWT(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("无效的 token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("token claims 解析失败")
	}

	return claims, nil
}
