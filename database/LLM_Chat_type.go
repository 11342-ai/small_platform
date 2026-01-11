package database

import (
	"gorm.io/gorm"
	"time"
)

// Persona 角色配置
type Persona struct {
	Name    string `yaml:"name" json:"name"`
	Content string `yaml:"content" json:"content"`
}

// StyleConfig 完整配置结构
type StyleConfig struct {
	Personas []Persona `yaml:"personas" json:"personas"`
}

// FileUploadConfig 文件上传配置结构
type FileUploadConfig struct {
	UploadDir         string   `yaml:"upload_dir"`
	MaxFileSize       int64    `yaml:"max_file_size"`
	AllowedExtensions []string `yaml:"allowed_extensions"`
}

type UploadedFile struct {
	gorm.Model
	SessionID   string `gorm:"index;not null"` // 关联的会话ID
	FileName    string `gorm:"not null"`       // 原文件名
	FilePath    string `gorm:"not null"`       // 存储路径
	FileSize    int64  `gorm:"not null"`       // 文件大小
	FileType    string `gorm:"not null"`       // 文件类型
	Content     string `gorm:"type:text"`      // 文件内容（文本文件）
	IsProcessed bool   `gorm:"default:false"`  // 是否已处理
}

// UserAPI 用户API配置
type UserAPI struct {
	gorm.Model
	UserID    uint   `gorm:"index;not null"`
	APIName   string `gorm:"size:100;not null"`
	APIKey    string `gorm:"size:500;not null"` // 加密存储
	ModelName string `gorm:"size:100"`
	BaseURL   string `gorm:"size:500"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ChatSession 聊天会话
type ChatSession struct {
	SessionID    string    `gorm:"primaryKey;size:50"`
	UserID       uint      `gorm:"index;not null"`
	Title        string    `gorm:"size:200"`
	ModelName    string    `gorm:"not null;default:''"`
	MessageCount int       `gorm:"default:0"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

// ChatMessage 聊天消息
type ChatMessage struct {
	gorm.Model
	SessionID string `gorm:"index;not null;size:50"`
	Role      string `gorm:"size:20;not null"` // user, assistant, system
	Content   string `gorm:"type:text"`
}

type SharedSession struct {
	ShareID      string     `gorm:"primaryKey;size:50;uniqueIndex"` // 分享ID
	SessionID    string     `gorm:"index;not null;size:50"`         // 关联会话
	CreatedBy    uint       `gorm:"index;not null"`                 // 创建者
	IsPublic     bool       `gorm:"default:true"`                   // 公开/私有
	ExpiresAt    *time.Time `gorm:"index"`                          // 过期时间
	MaxViews     int        `gorm:"default:-1"`                     // 最大访问次数（-1表示无限制）
	ViewCount    int        `gorm:"default:0"`                      // 当前访问次数
	LastAccessAt *time.Time // 最后访问时间
	CreatedAt    time.Time  `gorm:"autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime"`
}
