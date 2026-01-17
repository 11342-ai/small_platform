package LLM_Chat

import (
	"github.com/gin-gonic/gin"
	"math"
	"net/http"
	"platfrom/database"
	LLMService "platfrom/service/LLM_Chat"
	"strconv"
)

// RootGetAllSessions 管理员获取所有用户的会话列表
func RootGetAllSessions(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 移除 userID 相关逻辑，直接查询所有会话
	chatService := LLMService.GlobalChatService
	sessions, total, err := chatService.RootGetAllSessions(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取会话列表失败: " + err.Error()})
		return
	}

	// 关联查询用户名（批量查询优化）
	var userIDs []uint
	for _, session := range sessions {
		userIDs = append(userIDs, session.UserID)
	}

	// 查询所有相关用户
	var users []database.User
	database.DB.Where("id IN ?", userIDs).Find(&users)

	// 构建用户ID到用户名的映射
	userMap := make(map[uint]string)
	for _, user := range users {
		userMap[user.ID] = user.Username
	}

	// 转换为响应格式
	var sessionResponses []database.AdminSessionResponse
	for _, session := range sessions {
		sessionResponses = append(sessionResponses, database.AdminSessionResponse{
			SessionID:    session.SessionID,
			UserID:       session.UserID,
			Username:     userMap[session.UserID],
			Title:        session.Title,
			ModelName:    session.ModelName,
			MessageCount: session.MessageCount,
			CreatedAt:    session.CreatedAt,
			UpdatedAt:    session.UpdatedAt,
		})
	}

	// 计算总页数
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	c.JSON(http.StatusOK, database.AdminSessionListResponse{
		Sessions:   sessionResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// RootGetSessionMessages 管理员查看会话的所有消息
func RootGetSessionMessages(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话ID不能为空"})
		return
	}

	chatService := LLMService.GlobalChatService
	messages, err := chatService.RootGetSessionMessages(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取消息失败: " + err.Error()})
		return
	}

	// 转换为响应格式
	var messageResponses []database.AdminMessageResponse
	for _, msg := range messages {
		messageResponses = append(messageResponses, database.AdminMessageResponse{
			ID:        msg.ID,
			SessionID: msg.SessionID,
			Role:      msg.Role,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
		})
	}

	// 同时返回会话信息
	var session database.ChatSession
	if err := database.DB.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session": database.AdminSessionResponse{
			SessionID:    session.SessionID,
			UserID:       session.UserID,
			Title:        session.Title,
			ModelName:    session.ModelName,
			MessageCount: session.MessageCount,
			CreatedAt:    session.CreatedAt,
			UpdatedAt:    session.UpdatedAt,
		},
		"messages": messageResponses,
		"total":    len(messageResponses),
	})
}

// RootDeleteSession 管理员删除会话
func RootDeleteSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话ID不能为空"})
		return
	}

	// 先检查会话是否存在
	var session database.ChatSession
	if err := database.DB.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	chatService := LLMService.GlobalChatService
	if err := chatService.RootDeleteSession(sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除会话失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "会话删除成功",
		"session_id": sessionID,
		"user_id":    session.UserID,
	})
}
