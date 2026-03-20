package models

import (
	"time"
)

// OfflineMessage 离线消息模型
type OfflineMessage struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`                                    // 主键ID
	MessageID uint64    `gorm:"not null;index" json:"message_id"`                        // 消息ID，外键关联messages表
	ReceiverID uint     `gorm:"not null;index:idx_offline_receiver_created" json:"receiver_id"` // 接收者ID，外键关联users表
	IsRead    int       `gorm:"default:0" json:"is_read"`                               // 是否已推送，0=未推送，1=已推送
	CreatedAt time.Time `gorm:"index:idx_offline_receiver_created" json:"created_at"`   // 创建时间
}

// TableName 指定表名
func (OfflineMessage) TableName() string {
	return "offline_messages"
}
