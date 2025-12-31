package Auth

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"platfrom/service/Auth"
	"strings"
)

// AuthMiddleware 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从Header获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// 从Cookie获取token
			token, err := c.Cookie("access_token")
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "未提供认证令牌",
				})
				c.Abort()
				return
			}
			authHeader = "Bearer " + token
		}

		// 检查Bearer前缀
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "认证令牌格式错误",
			})
			c.Abort()
			return
		}

		// 验证token
		claims, err := Auth.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "认证令牌无效或已过期",
			})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}
