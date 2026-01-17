package Auth

import (
	"github.com/gin-gonic/gin"
	"math"
	"net/http"
	"platfrom/database"
	"platfrom/service/Auth"
	"strconv"
	"time"
)

// RootListAllUsers 获取所有用户列表
func RootListAllUsers(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	userService := getUserService()
	users, total, err := userService.RootListAllUsers(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败: " + err.Error()})
		return
	}

	// 转换为响应格式（隐藏密码哈希）
	var userResponses []database.AdminUserResponse
	for _, user := range users {
		userResponses = append(userResponses, database.AdminUserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      user.Role,
			LastLogin: user.LastLogin,
			CreatedAt: user.CreatedAt,
		})
	}

	// 计算总页数
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	c.JSON(http.StatusOK, database.UserListResponse{
		Users:      userResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// RootDeleteUser 删除用户
func RootDeleteUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 防止管理员删除自己
	currentUserID, _ := c.Get("user_id")
	if currentUserID.(uint) == uint(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除自己的账户"})
		return
	}

	userService := getUserService()
	if err := userService.RootDeleteUserByID(uint(userID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除用户失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "用户删除成功"})
}

// RootAddUser 管理员创建用户
func RootAddUser(c *gin.Context) {
	var req database.AdminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	userService := getUserService()
	user, err := userService.RootAddUser(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "用户创建成功",
		"user": database.AdminUserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
		},
	})
}


// RootLogin 管理员专用登录
func RootLogin(c *gin.Context) {
	var req database.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	userService := getUserService()

	// 1. 验证用户名密码
	user, err := userService.GetUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	if !Auth.VerifyPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 2. 关键：检查是否为管理员角色
	if user.Role != database.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "权限不足，此入口仅供管理员使用",
		})
		return
	}

	// 3. 生成 JWT Token（带上角色信息）
	token, err := Auth.GenerateToken(user.ID, user.Username, string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	// 4. 更新最后登录时间
	user.LastLogin = time.Now()
	if err := database.DB.Save(user).Error; err != nil {
		// 不影响登录流程，只记录日志
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新登录时间失败"})
		return
	}

	// 5. 返回响应（标记为管理员）
	c.JSON(http.StatusOK, gin.H{
		"message": "管理员登录成功",
		"token":   token,
		"user": database.AdminUserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      user.Role,
			LastLogin: user.LastLogin,
			CreatedAt: user.CreatedAt,
		},
	})
}
