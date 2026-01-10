package main

import (
	"fmt"
	"log"
	"os"
	"platfrom/Config"
	"platfrom/Route"
	"platfrom/database"
	"platfrom/service/Auth"
	"platfrom/service/LLM_Chat"
	"platfrom/service/Note"
)

func main() {

	// 初始化配置
	if err := Config.InitConfig(); err != nil {
		log.Printf("配置初始化失败: %v", err)
		os.Exit(1) // 只在 main 函数里决定是否退出
	}

	// 初始化数据库
	if err := database.InitDB(); err != nil {
		log.Printf("数据库初始化失败: %v", err)
		os.Exit(1)
	}

	redisAddr := fmt.Sprintf("%s:%s", Config.Cfg.RedisHost, Config.Cfg.RedisPort)
	fmt.Println(redisAddr)
	err := database.InitRedis(redisAddr, Config.Cfg.RedisPassword, Config.Cfg.RedisDB)
	if err != nil {
		log.Printf("Redis初始化失败，程序将继续在降级模式下运行: %v", err)
		_ = LLM_Chat.NewCacheService(nil, false)
	} else {
		log.Println("Redis初始化成功")
		_ = LLM_Chat.NewCacheService(database.GetRedis(), true)
	}

	//启动验证码清理任务（只创建一次）
	// 初始化 UserService（数据库已初始化后）
	_, _ = Auth.NewUserService(database.DB)
	if Auth.GlobalUserService == nil {
		log.Printf("Failed to initialize UserService")
		os.Exit(1)
	}
	Auth.GlobalUserService.StartCleanupTask()

	_, _ = LLM_Chat.NewUserAPIService(database.DB)
	if LLM_Chat.GlobalUserAPIService == nil {
		log.Printf("Failed to initialize GlobalUserAPIService")
		os.Exit(1)
	}

	_, _ = LLM_Chat.NewChatService(database.DB)
	if LLM_Chat.GlobalChatService == nil {
		log.Printf("Failed to initialize GlobalChatService")
		os.Exit(1)
	}

	_, _ = LLM_Chat.NewFileService(database.DB)
	if LLM_Chat.GlobalFileService == nil {
		log.Printf("Failed to initialize GlobalFileService")
		os.Exit(1)
	}

	// 初始化人格配置
	personaConfigs, err := LLM_Chat.LoadPersonaConfigs("style.yaml")
	if err != nil {
		log.Printf("加载人格配置失败:%s", err)
		os.Exit(1)
	}
	_, _ = LLM_Chat.NewPersonaManager(personaConfigs)
	if LLM_Chat.GlobalPersonaManager == nil {
		log.Printf("Failed to initialize GlobalPersonaManager")
		os.Exit(1)
	}

	if LLM_Chat.GlobalDefaultSessionCreator == nil {
		log.Printf("Failed to initialize GlobalDefaultSessionCreator")
		os.Exit(1)
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
