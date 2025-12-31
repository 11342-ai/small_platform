package main

import (
	"log"
	"platfrom/Config"
	"platfrom/Route"
	"platfrom/database"
	"platfrom/service/Auth"
	"platfrom/service/LLM_Chat"
	"platfrom/service/Note"
)

func main() {

	// 初始化配置
	Config.InitConfig()

	// 初始化数据库
	database.InitDB()

	err := database.InitRedis("localhost:6379", "", 0)
	if err != nil {
		log.Printf("Redis初始化失败，程序将继续在降级模式下运行: %v", err)
		_ = LLM_Chat.NewCacheService(nil, false)
	} else {
		log.Println("Redis初始化成功")
		_ = LLM_Chat.NewCacheService(database.GetRedis(), true)
	}

	//启动验证码清理任务（只创建一次）
	// 初始化 UserService（数据库已初始化后）
	_ = Auth.NewUserService()
	if Auth.GlobalUserService == nil {
		log.Fatal("Failed to initialize UserService")
	}
	Auth.GlobalUserService.StartCleanupTask()

	_ = LLM_Chat.NewUserAPIService()
	if LLM_Chat.GlobalUserAPIService == nil {
		log.Fatal("Failed to initialize GlobalUserAPIService")
	}

	_ = LLM_Chat.NewChatService()
	if LLM_Chat.GlobalChatService == nil {
		log.Fatal("Failed to initialize GlobalChatService")
	}

	_ = LLM_Chat.NewFileService()
	if LLM_Chat.GlobalFileService == nil {
		log.Fatal("Failed to initialize GlobalFileService")
	}

	_ = LLM_Chat.NewLLMSession()
	if LLM_Chat.GlobalLLMSession == nil {
		log.Fatal("Failed to initialize GlobalLLMSession")
	}

	// 初始化人格配置
	personaConfigs, err := LLM_Chat.LoadPersonaConfigs("style.yaml")
	if err != nil {
		log.Fatal("加载人格配置失败:", err)
	}
	_ = LLM_Chat.NewPersonaManager(personaConfigs)
	if LLM_Chat.GlobalPersonaManager == nil {
		log.Fatal("Failed to initialize GlobalPersonaManager")
	}

	_ = LLM_Chat.NewDefaultSessionCreator()
	if LLM_Chat.GlobalDefaultSessionCreator == nil {
		log.Fatal("Failed to initialize GlobalDefaultSessionCreator")
	}

	LLM_Chat.InitSessionManager(LLM_Chat.GlobalChatService, LLM_Chat.GlobalCacheService, LLM_Chat.GlobalUserAPIService, LLM_Chat.GlobalPersonaManager)

	_ = Note.NewNoteService()
	if Note.GlobalNoteService == nil {
		log.Fatal("Failed to initialize GlobalNoteService")
	}

	// 启动路由
	log.Println("服务器启动中...")
	Route.AuthRoute()
}
