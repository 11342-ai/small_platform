package database

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"log"
)

var (
	DB  *gorm.DB
	err error
)

func InitDB() {
	DB, err = gorm.Open(sqlite.Open("device.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	// 自动迁移表结构
	err = DB.AutoMigrate(
		&User{},
		&VerificationCode{},
	)
	if err != nil {
		log.Fatal("数据库迁移失败:", err)
	}

	log.Println("数据库连接成功")
}
