package Route

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"platfrom/Config"
	"platfrom/Route/Auth"
	"platfrom/Route/LLM_Chat"
	"platfrom/Route/Note"
	"strings"
	"time"
)

func AuthRoute() {
	r := gin.Default()

	// 配置CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:8080", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           120 * time.Hour,
		AllowOriginFunc: func(origin string) bool {
			// 在生产环境中可以动态验证来源
			return true
		},
	}))

	// 静态文件服务
	r.Static("/static", "./static")

	// API 路由
	api := r.Group("/api")
	{
		// 公开路由
		api.POST("/register", Auth.Register)
		api.POST("/login", Auth.Login)
		api.POST("/logout", Auth.Logout)
		// ← 管理员专用登录入口
		api.POST("/admin/login", Auth.RootLogin)
		// 验证码相关路由
		api.POST("/auth/send-code", Auth.SendVerificationCode)
		api.POST("/auth/verify-code", Auth.VerifyCode)
		api.POST("/auth/reset-password", Auth.ResetPassword)
	}

	// 管理员路由组
	adminGroup := r.Group("/api/admin")
	adminGroup.Use(Auth.AuthMiddleware(), Auth.AdminMiddleware())
	{
		adminGroup.GET("/users", Auth.RootListAllUsers)      // 获取用户列表
		adminGroup.POST("/users", Auth.RootAddUser)          // 创建用户
		adminGroup.DELETE("/users/:id", Auth.RootDeleteUser) // 删除用户

		// ← 新增：聊天管理
		adminGroup.GET("/sessions", LLM_Chat.RootGetAllSessions)                 // 获取所有会话列表
		adminGroup.GET("/sessions/:session_id", LLM_Chat.RootGetSessionMessages) // 查看会话消息
		adminGroup.DELETE("/sessions/:session_id", LLM_Chat.RootDeleteSession)   // 删除会话

		// ← 新增：笔记管理
		adminGroup.GET("/notes", Note.RootGetAllNotes)       // 获取所有笔记列表
		adminGroup.GET("/notes/:id", Note.RootGetNoteByID)   // 查看笔记详情
		adminGroup.DELETE("/notes/:id", Note.RootDeleteNote) // 删除笔记
	}

	// 需要认证的路由
	auth := api.Group("/")
	auth.Use(Auth.AuthMiddleware())

	// 用户相关
	{
		auth.GET("/profile", Auth.GetProfile)
		auth.POST("/update-password", Auth.UpdatePassword)
		auth.GET("/me", func(c *gin.Context) {
			// 为前端提供更友好的用户信息端点
			user, _ := c.Get("user_id")
			c.JSON(http.StatusOK, gin.H{"user_id": user})
		})

		// = = = = = = = 路由模型的配置 = = = = = = = =

		{
			auth.POST("/user/apis", LLM_Chat.CreateUserAPI)
			auth.GET("/user/apis", LLM_Chat.GetUserAPIs)
			auth.GET("/user/apis/first", LLM_Chat.GetFirstAvailableAPI)
			auth.GET("/user/apis/:name", LLM_Chat.GetUserAPIByName)
			auth.PUT("/user/apis/:id", LLM_Chat.UpdateUserAPI)
			auth.DELETE("/user/apis/:id", LLM_Chat.DeleteUserAPI)
		}

		// = = = = = 聊天相关路由 = = = = =

		chat := auth.Group("/chat")
		{
			chat.POST("/message", LLM_Chat.SendMessage)
			chat.POST("/message/stream", LLM_Chat.SendMessageStream)
			chat.POST("/session", LLM_Chat.CreateSession)
			chat.GET("/sessions", LLM_Chat.GetSessions)
			chat.GET("/sessions/:session_id/messages", LLM_Chat.GetSessionMessages)
			chat.DELETE("/sessions/:session_id", LLM_Chat.DeleteSession)
			chat.GET("/recover", LLM_Chat.RecoverStreamResponse)
		}

		// 人格管理路由
		personas := auth.Group("/personas")
		{
			personas.GET("/", LLM_Chat.GetPersonas)
		}

		// 文件管理路由
		files := auth.Group("/files")
		{
			files.POST("/upload", LLM_Chat.UploadFile())
			files.GET("/session/:session_id", LLM_Chat.GetSessionFiles())
			files.DELETE("/:file_id", LLM_Chat.DeleteFile())
		}

		// 笔记管理路由
		notes := auth.Group("/notes")
		{
			notes.GET("/", Note.GetNotes)
			notes.GET("/:id", Note.GetNoteByID)
			notes.POST("/", Note.CreateNote)
			notes.PUT("/:id", Note.UpdateNote)
			notes.DELETE("/:id", Note.DeleteNote)
			notes.GET("/category/:category", Note.GetNotesByCategory)
			notes.GET("/tag/:tag", Note.GetNotesByTag)
			notes.GET("/search/:keyword", Note.SearchNotes)
		}

		// 分享相关路由（全部要认证访问）
		shares := auth.Group("/chat/shares")
		{
			shares.POST("", LLM_Chat.CreateShare)                     // 创建分享（需要认证，在函数内部检查）
			shares.GET("", LLM_Chat.GetMyShares)                      // 我的分享列表（需要认证，在函数内部检查）
			shares.PUT("/:share_id", LLM_Chat.UpdateShare)            // 更新分享（需要认证，在函数内部检查）
			shares.DELETE("/:share_id", LLM_Chat.DeleteShare)         // 删除分享（需要认证，在函数内部检查）
			shares.GET("/:share_id/access", LLM_Chat.AccessShare)     // 访问分享（需要认证）
			shares.GET("/:share_id/info", LLM_Chat.GetShareInfo)      // 获取分享信息（需要认证）
			shares.GET("/:share_id/validate", LLM_Chat.ValidateShare) // 验证分享有效性（需要认证）
		}

	}

	r.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})

	r.GET("/profile", func(c *gin.Context) {
		c.File("./web/profile.html")
	})

	r.GET("/api_keys", func(c *gin.Context) {
		c.File("./web/api_keys.html")
	})

	r.GET("/chat", func(c *gin.Context) {
		c.File("./web/chat.html")
	})

	r.GET("/note", func(c *gin.Context) {
		c.File("./web/note.html")
	})

	r.GET("/share/:share_id", func(c *gin.Context) {
		c.File("./web/share.html")
	})

	// =====  ROOT  ======

	// 添加管理员前端页面路由
	r.GET("/admin/", func(c *gin.Context) {
		c.File("./web/root/index.html")
	})

	r.GET("/admin/users", func(c *gin.Context) {
		c.File("./web/root/admin_users.html")
	})

	r.GET("/admin/chats", func(c *gin.Context) {
		c.File("./web/root/admin_chats.html")
	})

	r.GET("/admin/notes", func(c *gin.Context) {
		c.File("./web/root/admin_notes.html")
	})

	// 前端路由 - 支持SPA
	r.NoRoute(func(c *gin.Context) {
		// 如果是API请求，返回404
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API not found"})
			return
		}

		// 否则返回前端应用
		c.File("./web/index.html")
	})

	// 启动服务器
	if err := r.Run(":" + Config.Cfg.ServerPort); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
