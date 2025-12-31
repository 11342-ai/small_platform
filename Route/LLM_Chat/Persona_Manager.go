package LLM_Chat

import (
	"github.com/gin-gonic/gin"
	"net/http"
	llmchatservice "platfrom/service/LLM_Chat"
)

// 新增人格路由设置
func setupPersonaRoutes(router *gin.Engine) {
	personas := router.Group("/api/personas")
	{
		personas.GET("/", GetPersonas)
	}
}

// GetPersonas 新增：获取可用人格列表的接口
func GetPersonas(c *gin.Context) {
	personas := llmchatservice.GetSessionManager().GetAvailablePersonas()
	c.JSON(http.StatusOK, gin.H{
		"data": personas,
	})
}
