package models

import (
	"fmt"

	"time"
)

type Message struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// 添加TableName方法指定表名（可选）
func (Message) TableName() string {
	return "messages"
}

// 在数据库初始化后创建索引
func CreateMessageIndexes() {
	// 为created_at字段创建索引，提高按时间排序的查询性能
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at)")
	// 为用户ID字段创建索引，提高按用户查询的性能
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_messages_user_id ON messages(user_id)")
	// 添加ID索引，用于游标分页
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_messages_id ON messages(id)")
	fmt.Println("✅ 消息表索引创建完成")
}
