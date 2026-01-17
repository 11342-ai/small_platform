package database

import (
	"gorm.io/gorm"
	"time"
)

type Note struct {
	gorm.Model
	UserID   uint     `gorm:"index;not null"`
	Title    string   `gorm:"size:255;not null" json:"title"`
	Content  string   `gorm:"type:text;not null" json:"content"`
	Tags     []string `gorm:"type:text" json:"tags"` // 使用text类型存储JSON
	Category string   `gorm:"size:100;default:'未分类'" json:"category"`
	IsPublic bool     `gorm:"default:false" json:"is_public"`
}

// ========== ROOT ==========

// AdminNoteResponse 管理员查看的笔记信息（包含用户信息）
type AdminNoteResponse struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username"` // 关联查询用户名
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	Category  string    `json:"category"`
	IsPublic  bool      `json:"is_public"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AdminNoteListResponse 管理员笔记列表响应
type AdminNoteListResponse struct {
	Notes      []AdminNoteResponse `json:"notes"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalPages int                 `json:"total_pages"`
}
