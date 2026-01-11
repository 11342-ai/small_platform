package LLM_Chat

import (
	"github.com/gin-gonic/gin"
	"net/http"
	LLM_Chat_Service "platfrom/service/LLM_Chat"
	"time"
)

// SetupShareRoutes 设置分享相关路由
func SetupShareRoutes(router *gin.Engine) {
	shares := router.Group("/api/chat/shares")
	{
		// === 创建者操作（需要认证）===
		shares.POST("", CreateShare)             // 创建分享
		shares.GET("", GetMyShares)              // 我的分享列表
		shares.PUT("/:share_id", UpdateShare)    // 更新分享
		shares.DELETE("/:share_id", DeleteShare) // 删除分享

		// === 被分享者操作（公开访问，根据 is_public 控制权限）===
		shares.GET("/:share_id/access", AccessShare)     // 访问分享（计数+1）
		shares.GET("/:share_id/info", GetShareInfo)      // 获取分享信息
		shares.GET("/:share_id/validate", ValidateShare) // 验证分享有效性
	}
}

// ===== 请求/响应结构体 =====

type CreateShareRequest struct {
	SessionID string    `json:"session_id" binding:"required"`
	MaxViews  int       `json:"max_views" binding:"omitempty,gte=-1"` // -1表示无限制
	ExpiresAt time.Time `json:"expires_at" binding:"omitempty"`       // 空表示永不过期
	IsPublic  bool      `json:"is_public" binding:"omitempty"`        // 默认true
}

type UpdateShareRequest struct {
	MaxViews  *int       `json:"max_views" binding:"omitempty,gte=-1"`
	ExpiresAt *time.Time `json:"expires_at" binding:"omitempty"`
	IsPublic  *bool      `json:"is_public" binding:"omitempty"`
}

type ShareResponse struct {
	ShareID      string     `json:"share_id"`
	SessionID    string     `json:"session_id"`
	CreatedBy    uint       `json:"created_by"`
	IsPublic     bool       `json:"is_public"`
	ExpiresAt    *time.Time `json:"expires_at"`
	MaxViews     int        `json:"max_views"`
	ViewCount    int        `json:"view_count"`
	LastAccessAt *time.Time `json:"last_access_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ShareURL     string     `json:"share_url"` // 完整分享链接
}

type AccessShareResponse struct {
	Session  interface{}   `json:"session"`  // ChatSession
	Messages interface{}   `json:"messages"` // ChatMessage[]
	Share    ShareResponse `json:"share"`    // 分享元信息
}

// ===== 创建者操作（需要认证）=====

// CreateShare 创建分享链接
func CreateShare(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	var req CreateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 设置默认值
	if req.MaxViews == 0 {
		req.MaxViews = -1 // 默认无限制
	}

	// 调用服务创建分享
	var expiresAt *time.Time
	if !req.ExpiresAt.IsZero() {
		expiresAt = &req.ExpiresAt
	}

	shareID, err := LLM_Chat_Service.GlobalSharedSessionService.CreateSharedLink(
		req.SessionID,
		userID.(uint),
		req.MaxViews,
		expiresAt,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "创建分享失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "分享链接创建成功",
		"share_id":  shareID,
		"share_url": "/share/" + shareID, // 前端展示用
	})
}

// GetMyShares 获取我的分享列表
func GetMyShares(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	shares, err := LLM_Chat_Service.GlobalSharedSessionService.ListMySharedLinks(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取分享列表失败: " + err.Error()})
		return
	}

	// 转换为响应格式
	var shareResponses []ShareResponse
	for _, share := range shares {
		shareResponses = append(shareResponses, ShareResponse{
			ShareID:      share.ShareID,
			SessionID:    share.SessionID,
			CreatedBy:    share.CreatedBy,
			IsPublic:     share.IsPublic,
			ExpiresAt:    share.ExpiresAt,
			MaxViews:     share.MaxViews,
			ViewCount:    share.ViewCount,
			LastAccessAt: share.LastAccessAt,
			CreatedAt:    share.CreatedAt,
			UpdatedAt:    share.UpdatedAt,
			ShareURL:     "/share/" + share.ShareID,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"shares": shareResponses,
		"count":  len(shareResponses),
	})
}

// UpdateShare 更新分享配置
func UpdateShare(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	shareID := c.Param("share_id")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "分享ID不能为空"})
		return
	}

	var req UpdateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	if req.MaxViews != nil {
		updates["max_views"] = *req.MaxViews
	}
	if req.ExpiresAt != nil {
		updates["expires_at"] = *req.ExpiresAt
	}
	if req.IsPublic != nil {
		updates["is_public"] = *req.IsPublic
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有需要更新的字段"})
		return
	}

	err := LLM_Chat_Service.GlobalSharedSessionService.UpdateSharedLink(
		shareID,
		userID.(uint),
		updates,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "更新分享失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "分享更新成功"})
}

// DeleteShare 删除分享链接
func DeleteShare(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	shareID := c.Param("share_id")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "分享ID不能为空"})
		return
	}

	err := LLM_Chat_Service.GlobalSharedSessionService.DeleteSharedLink(shareID, userID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "删除分享失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "分享删除成功"})
}

// ===== 被分享者操作（公开访问）=====

// AccessShare 访问分享链接（增加计数）
func AccessShare(c *gin.Context) {
	shareID := c.Param("share_id")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "分享ID不能为空"})
		return
	}

	session, messages, share, err := LLM_Chat_Service.GlobalSharedSessionService.AccessSharedLink(shareID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "访问分享失败: " + err.Error()})
		return
	}

	// 转换为响应格式
	shareResponse := ShareResponse{
		ShareID:      share.ShareID,
		SessionID:    share.SessionID,
		CreatedBy:    share.CreatedBy,
		IsPublic:     share.IsPublic,
		ExpiresAt:    share.ExpiresAt,
		MaxViews:     share.MaxViews,
		ViewCount:    share.ViewCount,
		LastAccessAt: share.LastAccessAt,
		CreatedAt:    share.CreatedAt,
		UpdatedAt:    share.UpdatedAt,
		ShareURL:     "/share/" + share.ShareID,
	}

	c.JSON(http.StatusOK, AccessShareResponse{
		Session:  session,
		Messages: messages,
		Share:    shareResponse,
	})
}

// GetShareInfo 获取分享信息（不增加计数）
func GetShareInfo(c *gin.Context) {
	shareID := c.Param("share_id")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "分享ID不能为空"})
		return
	}

	share, err := LLM_Chat_Service.GlobalSharedSessionService.GetSharedLinkInfo(shareID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分享不存在"})
		return
	}

	shareResponse := ShareResponse{
		ShareID:      share.ShareID,
		SessionID:    share.SessionID,
		CreatedBy:    share.CreatedBy,
		IsPublic:     share.IsPublic,
		ExpiresAt:    share.ExpiresAt,
		MaxViews:     share.MaxViews,
		ViewCount:    share.ViewCount,
		LastAccessAt: share.LastAccessAt,
		CreatedAt:    share.CreatedAt,
		UpdatedAt:    share.UpdatedAt,
		ShareURL:     "/share/" + share.ShareID,
	}

	c.JSON(http.StatusOK, gin.H{"share": shareResponse})
}

// ValidateShare 验证分享有效性
func ValidateShare(c *gin.Context) {
	shareID := c.Param("share_id")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "分享ID不能为空"})
		return
	}

	valid, err := LLM_Chat_Service.GlobalSharedSessionService.ValidateSharedLink(shareID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "验证失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":    valid,
		"share_id": shareID,
	})
}
