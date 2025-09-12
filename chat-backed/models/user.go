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
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"unique"`
	Password string
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
	DB.AutoMigrate(&User{}, &Message{})

	CreateMessageIndexes() // 创建消息表的索引
	fmt.Println("✅ 数据库初始化成功，已经创建用户表和消息表")
}
