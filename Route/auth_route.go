package Route

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"platfrom/Config"
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

	// 公开路由
	api.POST("/register", Register)
	api.POST("/login", Login)
	api.POST("/logout", Logout)

	// 验证码相关路由
	api.POST("/auth/send-code", SendVerificationCode)
	api.POST("/auth/verify-code", VerifyCode)
	api.POST("/auth/reset-password", ResetPassword)

	// 需要认证的路由
	auth := api.Group("/")
	auth.Use(AuthMiddleware())

	// 用户相关
	{
		auth.GET("/profile", GetProfile)
		auth.POST("/update-password", UpdatePassword)
		auth.GET("/me", func(c *gin.Context) {
			// 为前端提供更友好的用户信息端点
			user, _ := c.Get("user_id")
			c.JSON(http.StatusOK, gin.H{"user_id": user})
		})
	}

	// 前端路由 - 支持SPA
	// 修改后
	r.NoRoute(func(c *gin.Context) {
		// 使用 strings.HasPrefix 检查前缀，更安全
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API not found"})
			return
		}

		// 个人中心页面
		//if strings.HasPrefix(c.Request.URL.Path, "/profile") {
		//	c.File("./web/profile.html")
		//	return
		//}

		// 返回前端应用
		c.File("./web/index.html")
	})

	// 启动服务器
	if err := r.Run(":" + Config.Cfg.ServerPort); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
