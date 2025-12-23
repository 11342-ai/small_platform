package main

import (
	"log"
	"platfrom/Config"
	"platfrom/Route"
	"platfrom/database"
	"platfrom/service"
)

// 添加全局 UserService
var userService service.UserService

func main() {

	// 初始化配置
	Config.InitConfig()

	// 初始化数据库
	database.InitDB()

	//启动验证码清理任务（只创建一次）
	// 初始化 UserService（数据库已初始化后）
	_ = service.NewUserService()
	if service.GlobalUserService == nil {
		log.Fatal("Failed to initialize UserService")
	}
	service.GlobalUserService.StartCleanupTask()

	// 启动路由
	log.Println("服务器启动中...")
	Route.AuthRoute()
}
