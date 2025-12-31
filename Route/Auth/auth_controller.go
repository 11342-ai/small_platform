package Auth

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"platfrom/database"
	"platfrom/service/Auth"
	"time"
)

// 修改：删除全局变量，在函数中获取服务实例
func getUserService() Auth.UserService {
	return Auth.GlobalUserService
}

// Register 用户注册
func Register(c *gin.Context) {
	var req database.RegisterRequest

	// 绑定请求数据
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 创建用户
	userService := getUserService()
	user, err := userService.CreateUser(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "创建用户失败: " + err.Error(),
		})
		return
	}

	// 生成JWT令牌
	token, err := Auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "生成令牌失败",
		})
		return
	}

	c.SetCookie("access_token", token, 3600*24*7, "/", "", false, true)

	c.JSON(http.StatusOK, database.LoginResponse{
		Message: "注册成功",
		Token:   token,
		User: database.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		},
	})
}

// Login 用户登录
func Login(c *gin.Context) {
	var req database.LoginRequest

	// 绑定请求数据
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取用户
	userService := getUserService()
	user, err := userService.GetUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户名或密码错误",
		})
		return
	}

	// 验证密码
	if !Auth.VerifyPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "用户名或密码错误",
		})
		return
	}

	// 更新最后登录时间
	now := time.Now()
	user.LastLogin = now
	// 这里可以保存到数据库

	// 生成JWT令牌
	token, err := Auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "生成令牌失败",
		})
		return
	}

	// 设置Cookie
	c.SetCookie("access_token", token, 3600*24*7, "/", "", false, true)

	// 返回响应
	c.JSON(http.StatusOK, database.LoginResponse{
		Message: "登录成功",
		Token:   token,
		User: database.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		},
	})
}

// Logout 用户注销
func Logout(c *gin.Context) {
	// 清除Cookie
	c.SetCookie("access_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "已退出登录",
	})
}

// GetProfile 获取用户资料
func GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	userService := getUserService()
	user, err := userService.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "用户不存在",
		})
		return
	}

	c.JSON(http.StatusOK, database.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	})
}

// SendVerificationCode 发送验证码
func SendVerificationCode(c *gin.Context) {
	var req database.SendCodeRequest

	// 绑定请求数据
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 发送验证码
	userService := getUserService()
	_, err := userService.SendVerificationCode(req.Username, req.CodeType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "发送验证码失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, database.CodeResponse{
		Message: "验证码已发送",
		Expires: 5, // 5分钟有效期
	})
}

// VerifyCode 验证验证码
func VerifyCode(c *gin.Context) {
	var req database.VerifyCodeRequest

	// 绑定请求数据
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 验证验证码
	userService := getUserService()
	isValid, err := userService.VerifyCode(req.Username, req.Code, req.CodeType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "验证码验证失败: " + err.Error(),
		})
		return
	}

	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "验证码无效",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "验证码验证成功",
		"valid":   true,
	})
}

// ResetPassword 忘记密码重置（通过验证码）
func ResetPassword(c *gin.Context) {
	var req database.ResetPasswordRequest

	// 绑定请求数据
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 重置密码
	userService := getUserService()
	err := userService.ResetPassword(req.Username, req.Code, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "重置密码失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "密码重置成功",
	})
}

// UpdatePassword 修改密码（需要旧密码验证）
func UpdatePassword(c *gin.Context) {
	// 从上下文中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	var req database.UpdatePasswordRequest

	// 绑定请求数据
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 修改密码
	userService := getUserService()
	err := userService.UpdatePassword(userID.(uint), req.OldPassword, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "修改密码失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "密码修改成功",
	})
}
