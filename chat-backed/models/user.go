package models

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

type User struct {
	ID        uint   `gorm:"primaryKey"`
	Username  string `gorm:"unique"`
	Password  string
	Avatar    string    `gorm:"default:''"`        // 头像URL
	Bio       string    `gorm:"type:text"`         // 个性签名
	Status    string    `gorm:"default:'offline'"` // 状态: online, offline, busy, away
	LastSeen  time.Time // 最后在线时间
	CreatedAt time.Time
	UpdatedAt time.Time
}

// 好友关系模型
type Friendship struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"not null"`          // 用户ID
	FriendID  uint   `gorm:"not null"`          // 好友ID
	Status    string `gorm:"default:'pending'"` // 状态: pending, accepted, rejected
	CreatedAt time.Time
	UpdatedAt time.Time
}

// 为用户和好友关系创建索引
func (Friendship) TableName() string {
	return "friendships"
}

// InitDB 连接 MySQL
func InitDB() {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("数据库连接失败: %v", err)
		log.Printf("连接字符串: %s", strings.Replace(dsn, os.Getenv("DB_PASS"), "****", 1))
		panic("数据库连接失败")
	}

	// 测试连接
	sqlDB, err := DB.DB()
	if err != nil {
		panic("获取数据库实例失败: " + err.Error())
	}

	if err := sqlDB.Ping(); err != nil {
		panic("数据库 ping 失败: " + err.Error())
	}

	// 设置连接池
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动迁移模式
	DB.AutoMigrate(&User{}, &Message{}, &Friendship{})

	// 创建消息表索引
	CreateMessageIndexes()

	// 创建好友关系表索引
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_friendships_user_friend ON friendships(user_id, friend_id)")
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_friendships_status ON friendships(status)")

	fmt.Println("✅ 数据库初始化成功，已经创建用户表、消息表、好友关系表和索引")
}
