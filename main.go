package main

import (
	"log"
	"platfrom/Config"
	"platfrom/Route"
	"platfrom/database"
	"platfrom/service"
	"platfrom/service/LLM_Chat"
)

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

	_ = LLM_Chat.NewChatMessageService()
	if LLM_Chat.GlobalChatMessageService == nil {
		log.Fatal("Failed to initialize ChatMessageService")
	}

	_ = LLM_Chat.NewChatSessionService()
	if LLM_Chat.GlobalChatSessionService == nil {
		log.Fatal("Failed to initialize GlobalChatSessionService")
	}

	_ = LLM_Chat.NewUserAPIService()
	if LLM_Chat.GlobalUserAPIService == nil {
		log.Fatal("Failed to initialize GlobalUserAPIService")
	}

	// 初始化人格配置管理器
	_ = LLM_Chat.NewPersonaConfigManager()
	if LLM_Chat.GlobalConfigManager == nil {
		log.Fatal("Failed to initialize GlobalConfigManager")
	}

	if err := LLM_Chat.GlobalConfigManager.LoadConfig("style.yaml"); err != nil {
		// 如果配置文件不存在，记录警告但不终止程序，可以使用默认配置
		log.Printf("Warning: Failed to load persona config: %v", err)
		// 可以在这里创建默认配置或使用内置默认值
	}

	// 初始化文件上传配置管理器
	_ = LLM_Chat.NewFileUploadConfigManager()
	if LLM_Chat.GlobalFileUploadConfigManager == nil {
		log.Fatal("Failed to initialize GlobalFileUploadConfigManager")
	}

	// 加载文件上传配置文件
	if err := LLM_Chat.GlobalFileUploadConfigManager.LoadConfig("style.yaml"); err != nil {
		// 如果配置文件不存在，记录警告但不终止程序
		log.Printf("Warning: Failed to load file upload config: %v", err)
		// 可以在这里设置默认配置
	}

	// 初始化LLM服务
	_ = LLM_Chat.NewLLMService()

	// 启动路由
	log.Println("服务器启动中...")
	Route.AuthRoute()
}
