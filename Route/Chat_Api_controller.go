package Route

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"platfrom/database"
	"platfrom/service/LLM_Chat"
	"strconv"
)

// APICreateRequest API管理相关的请求和响应结构体
type APICreateRequest struct {
	APIName   string `json:"api_name" binding:"required"`
	APIKey    string `json:"api_key" binding:"required"`
	ModelName string `json:"model_name" binding:"required"`
	BaseURL   string `json:"base_url" binding:"omitempty,url"`
}

type APIUpdateRequest struct {
	APIName   string `json:"api_name" binding:"omitempty"`
	APIKey    string `json:"api_key" binding:"omitempty"`
	ModelName string `json:"model_name" binding:"omitempty"`
	BaseURL   string `json:"base_url" binding:"omitempty,url"`
}

type APIResponse struct {
	ID        uint   `json:"id"`
	APIName   string `json:"api_name"`
	ModelName string `json:"model_name"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CreateUserAPI 创建新的API配置
func CreateUserAPI(c *gin.Context) {
	// 从上下文中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	var req APICreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 创建API配置对象
	apiConfig := &database.UserAPI{
		UserID:    userID.(uint),
		APIName:   req.APIName,
		APIKey:    req.APIKey,
		ModelName: req.ModelName,
		BaseURL:   req.BaseURL,
	}

	// 调用服务创建API
	createdAPI, err := LLM_Chat.GlobalUserAPIService.CreateAPI(userID.(uint), apiConfig)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "创建API失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API创建成功",
		"api": APIResponse{
			ID:        createdAPI.ID,
			APIName:   createdAPI.APIName,
			ModelName: createdAPI.ModelName,
			BaseURL:   createdAPI.BaseURL,
			CreatedAt: createdAPI.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: createdAPI.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
	})
}

// GetUserAPIs 获取用户的所有API配置
func GetUserAPIs(c *gin.Context) {
	// 从上下文中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	// 调用服务获取用户所有API
	apis, err := LLM_Chat.GlobalUserAPIService.GetUserAPIs(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取API列表失败: " + err.Error(),
		})
		return
	}

	// 转换为响应格式
	var apiResponses []APIResponse
	for _, api := range apis {
		apiResponses = append(apiResponses, APIResponse{
			ID:        api.ID,
			APIName:   api.APIName,
			ModelName: api.ModelName,
			BaseURL:   api.BaseURL,
			APIKey:    api.APIKey,
			CreatedAt: api.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: api.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"apis":  apiResponses,
		"count": len(apiResponses),
	})
}

// GetUserAPIByName 根据名称获取用户的API配置
func GetUserAPIByName(c *gin.Context) {
	// 从上下文中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	apiName := c.Param("name")
	if apiName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "API名称不能为空",
		})
		return
	}

	// 调用服务获取API
	api, err := LLM_Chat.GlobalUserAPIService.GetAPIByName(userID.(uint), apiName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "API配置不存在: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api": APIResponse{
			ID:        api.ID,
			APIName:   api.APIName,
			ModelName: api.ModelName,
			BaseURL:   api.BaseURL,
			APIKey:    api.APIKey,
			CreatedAt: api.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: api.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
	})
}

// UpdateUserAPI 更新API配置
func UpdateUserAPI(c *gin.Context) {
	// 从上下文中获取用户ID
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	apiIDStr := c.Param("id")
	apiID, err := strconv.ParseUint(apiIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "API ID格式错误",
		})
		return
	}

	var req APIUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	if req.APIName != "" {
		updates["api_name"] = req.APIName
	}
	if req.APIKey != "" {
		updates["api_key"] = req.APIKey
	}
	if req.ModelName != "" {
		updates["model_name"] = req.ModelName
	}
	if req.BaseURL != "" {
		updates["base_url"] = req.BaseURL
	}

	// 检查是否有更新字段
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "没有提供更新内容",
		})
		return
	}

	// 调用服务更新API
	err = LLM_Chat.GlobalUserAPIService.UpdateAPI(uint(apiID), updates)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "更新API失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API更新成功",
	})
}

// DeleteUserAPI 删除API配置
func DeleteUserAPI(c *gin.Context) {
	// 从上下文中获取用户ID
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	apiIDStr := c.Param("id")
	apiID, err := strconv.ParseUint(apiIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "API ID格式错误",
		})
		return
	}

	// 调用服务删除API
	err = LLM_Chat.GlobalUserAPIService.DeleteAPI(uint(apiID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "删除API失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API删除成功",
	})
}

// GetFirstAvailableAPI 获取用户第一个可用的API配置（用于下拉列表初始化）
func GetFirstAvailableAPI(c *gin.Context) {
	// 从上下文中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	// 调用服务获取第一个可用API
	api, err := LLM_Chat.GlobalUserAPIService.GetFirstAvailableAPI(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "用户没有配置任何API: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api": APIResponse{
			ID:        api.ID,
			APIName:   api.APIName,
			ModelName: api.ModelName,
			BaseURL:   api.BaseURL,
			APIKey:    api.APIKey,
			CreatedAt: api.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: api.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
	})
}
