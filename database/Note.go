package database

import "gorm.io/gorm"

type Note struct {
	gorm.Model
	UserID   uint     `gorm:"index;not null"`
	Title    string   `gorm:"size:255;not null" json:"title"`
	Content  string   `gorm:"type:text;not null" json:"content"`
	Tags     []string `gorm:"type:text" json:"tags"` // 使用text类型存储JSON
	Category string   `gorm:"size:100;default:'未分类'" json:"category"`
	IsPublic bool     `gorm:"default:false" json:"is_public"`
}
