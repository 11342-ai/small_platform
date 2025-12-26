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

// UserAPI 用户API配置
type UserAPI struct {
	gorm.Model
	UserID    uint   `gorm:"index;not null"`
	APIName   string `gorm:"size:100;not null"`
	APIKey    string `gorm:"size:500;not null"` // 加密存储
	ModelName string `gorm:"size:100"`
	BaseURL   string `gorm:"size:500"`
}

// ChatSession 聊天会话
type ChatSession struct {
	SessionID     string `gorm:"primaryKey;size:50"`
	UserID        uint   `gorm:"index;not null"`
	Title         string `gorm:"size:200"`
	PersonaName   string `gorm:"size:100"`
	LastMessageAt time.Time
	MessageCount  int `gorm:"default:0"`
}

// ChatMessage 聊天消息
type ChatMessage struct {
	gorm.Model
	SessionID    string `gorm:"index;not null;size:50"`
	UserID       uint   `gorm:"index;not null"`
	Role         string `gorm:"size:20;not null"` // user, assistant, system
	Content      string `gorm:"type:text"`
	MessageOrder int    `gorm:"not null"`
}
