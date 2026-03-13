package models

import (
	"fmt"
	"time"
)

// Message 消息模型
type Message struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `json:"user_id"`
	Username    string    `json:"username"`
	Content     string    `json:"content"`
	MessageType string    `json:"message_type" gorm:"default:'text'"` // 消息类型: text, image, file
	FileURL     string    `json:"file_url"`                           // 文件URL
	FileName    string    `json:"file_name"`                          // 文件名
	FileSize    int64     `json:"file_size"`                          // 文件大小
	GroupID     uint      `json:"group_id" gorm:"index"`              // 群组ID，0表示私聊或全局聊天
	Target      uint      `json:"target" gorm:"index"`                // 私聊目标用户ID，0表示群聊
	ReadCount   int       `json:"read_count" gorm:"default:0"`        // 已读人数（用于快速查询）
	CreatedAt   time.Time `json:"created_at"`
}

// MessageReadStatus 消息已读状态表
// 记录每条消息被哪些用户读取过
type MessageReadStatus struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	MessageID uint      `gorm:"not null;index:idx_message_reader,unique" json:"message_id"` // 消息ID
	ReaderID  uint      `gorm:"not null;index:idx_message_reader,unique" json:"reader_id"`  // 读取者ID
	ReadAt    time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"read_at"`                   // 读取时间
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定 Message 表名
func (Message) TableName() string {
	return "messages"
}

// TableName 指定 MessageReadStatus 表名
func (MessageReadStatus) TableName() string {
	return "message_read_status"
}

// CreateMessageIndexes 在数据库初始化后创建消息表索引
func CreateMessageIndexes() {
	// 为created_at字段创建索引，提高按时间排序的查询性能
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at)")
	// 为用户ID字段创建索引，提高按用户查询的性能
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_messages_user_id ON messages(user_id)")
	// 添加ID索引，用于游标分页
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_messages_id ON messages(id)")
	// 为群组ID字段创建索引，提高按群组查询的性能
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_messages_group_id ON messages(group_id)")
	// 创建复合索引，提高按群组和时间查询的性能
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_messages_group_created ON messages(group_id, created_at)")
	// 为target字段创建索引，提高私聊消息查询性能
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_messages_target ON messages(target)")
	
	fmt.Println("✅ 消息表索引创建完成")
}

// CreateMessageReadStatusIndexes 创建消息已读状态表索引
func CreateMessageReadStatusIndexes() {
	// 为message_id创建索引，提高查询消息已读记录的性能
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_read_status_message ON message_read_status(message_id)")
	// 为reader_id创建索引，提高查询用户已读消息的性能
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_read_status_reader ON message_read_status(reader_id)")
	// 为read_at创建索引，提高按时间查询的性能
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_read_status_time ON message_read_status(read_at)")
	
	fmt.Println("✅ 消息已读状态表索引创建完成")
}
