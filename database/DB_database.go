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
	// 修复 chat_sessions 表的 model_name 列（如果存在迁移问题）
	err := fixChatSessionModelName(DB)
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
	)
	if err != nil {
		return fmt.Errorf("数据库迁移失败:%s", err)
	}

	// 修复 chat_sessions 表的时间戳列
	if err := fixChatSessionTimestamps(DB); err != nil {
		return fmt.Errorf("警告: 修复 chat_sessions 表时间戳失败: %v", err)
	}

	log.Println("数据库连接成功")
	return nil // ✅ 成功返回 nil
}

// fixChatSessionModelName 确保 chat_sessions 表有 model_name 列，且允许 NULL 或具有默认值
func fixChatSessionModelName(db *gorm.DB) error {
	if !db.Migrator().HasTable(&ChatSession{}) {
		return nil // 表不存在，将由 AutoMigrate 创建
	}
	// 检查 model_name 列是否存在
	if !db.Migrator().HasColumn(&ChatSession{}, "ModelName") {
		// 添加带有默认值的列
		err := db.Exec("ALTER TABLE chat_sessions ADD COLUMN model_name TEXT NOT NULL DEFAULT ''").Error
		if err != nil {
			// 如果失败，尝试添加可为 NULL 的列
			err2 := db.Exec("ALTER TABLE chat_sessions ADD COLUMN model_name TEXT").Error
			if err2 != nil {
				return err2
			}
			// 更新现有行为空字符串
			db.Exec("UPDATE chat_sessions SET model_name = '' WHERE model_name IS NULL")
			// 然后修改列约束为 NOT NULL（SQLite 不支持 ALTER COLUMN，需要重建表，所以跳过）
		}
	}
	return nil
}

// fixChatSessionTimestamps 确保 chat_sessions 表有 created_at 和 updated_at 列，并为现有记录设置默认值
func fixChatSessionTimestamps(db *gorm.DB) error {
	if !db.Migrator().HasTable(&ChatSession{}) {
		return nil // 表不存在，将由 AutoMigrate 创建
	}
	// 检查 created_at 列是否存在
	if !db.Migrator().HasColumn(&ChatSession{}, "CreatedAt") {
		// 添加列（AutoMigrate 应该已添加，但以防万一）
		err := db.Exec("ALTER TABLE chat_sessions ADD COLUMN created_at DATETIME DEFAULT CURRENT_TIMESTAMP").Error
		if err != nil {
			return err
		}
	}
	// 检查 updated_at 列是否存在
	if !db.Migrator().HasColumn(&ChatSession{}, "UpdatedAt") {
		err := db.Exec("ALTER TABLE chat_sessions ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP").Error
		if err != nil {
			return err
		}
	}
	// 更新现有记录的 created_at 和 updated_at（如果为 NULL）
	result := db.Exec("UPDATE chat_sessions SET created_at = CURRENT_TIMESTAMP WHERE created_at IS NULL")
	if result.Error != nil {
		return result.Error
	}
	result = db.Exec("UPDATE chat_sessions SET updated_at = CURRENT_TIMESTAMP WHERE updated_at IS NULL")
	if result.Error != nil {
		return result.Error
	}
	return nil
}
