package models

import (
	"log"
	"time"

	"gorm.io/gorm"
)

// Group 群组模型
type Group struct {
	gorm.Model
	Name        string `json:"name" gorm:"not null"`         // 群名
	OwnerID     uint   `json:"owner_id" gorm:"not null"`     // 群主的用户ID
	Avatar      string `json:"avatar" gorm:"default:''"`     // 群头像链接
	Description string `json:"description" gorm:"type:text"` // 群描述

	// 关联关系
	Owner   User          `json:"owner" gorm:"foreignKey:OwnerID"`   // 群主信息
	Members []GroupMember `json:"members" gorm:"foreignKey:GroupID"` // 群成员列表
}

// GroupMember 群组成员关系模型
type GroupMember struct {
	UserID    uint      `json:"user_id" gorm:"primaryKey"`             // 用户ID
	GroupID   uint      `json:"group_id" gorm:"primaryKey"`            // 群组ID
	Role      string    `json:"role" gorm:"not null;default:'member'"` // 角色: owner, admin, member
	JoinedAt  time.Time `json:"joined_at" gorm:"autoCreateTime"`       // 加入时间
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`      // 更新时间

	// 关联关系
	User  User  `json:"user" gorm:"foreignKey:UserID"`   // 用户信息
	Group Group `json:"group" gorm:"foreignKey:GroupID"` // 群组信息
}

// TableName 指定群组表名
func (Group) TableName() string {
	return "groups"
}

// TableName 指定群成员表名
func (GroupMember) TableName() string {
	return "group_members"
}

// CreateGroupIndexes 创建群组相关索引
func CreateGroupIndexes() {
	// 为群组所有者创建索引
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_groups_owner_id ON groups(owner_id)")

	// 为群成员关系创建复合索引
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_group_members_user_id ON group_members(user_id)")
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_group_members_group_id ON group_members(group_id)")
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_group_members_role ON group_members(role)")

	// 为群成员关系创建复合索引，提高查询性能
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_group_members_user_group ON group_members(user_id, group_id)")

	DB.Exec("CREATE INDEX IF NOT EXISTS idx_groups_deleted_at ON groups(deleted_at)")

	// 为群组名称创建索引，支持群组搜索
	DB.Exec("CREATE INDEX IF NOT EXISTS idx_groups_name ON groups(name)")

	log.Println("✅ 群组相关索引创建完成")
}
