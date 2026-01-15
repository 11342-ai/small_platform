package database

import (
	"fmt"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"log"
)

var (
	DB  *gorm.DB
	err error
)

func InitDB() error {
	DB, err = gorm.Open(sqlite.Open("E:/procedure/Go/tmp/device.db"), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	if err != nil {
		return fmt.Errorf("警告: 修复 chat_sessions 表失败: %v", err)
	}
	// 自动迁移表结构
	err = DB.AutoMigrate(
		&User{},
		&VerificationCode{},
		&UserAPI{},     // 新增
		&ChatSession{}, // 新增
		&ChatMessage{}, // 新增
		&UploadedFile{},
		&Note{},
		&SharedSession{},
		&FollowModel{},
		&ArticleModel{},
		&ArticleUserModel{},
		&FavoriteModel{},
		&TagModel{},
		&CommentModel{},
	)
	if err != nil {
		return fmt.Errorf("数据库迁移失败:%s", err)
	}

	log.Println("数据库连接成功")
	return nil // ✅ 成功返回 nil
}
