package LLM_Chat

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"net/http"
	LLM_Chat_Service "platfrom/service/LLM_Chat"
	"strconv"
	"strings"
	"time"
)

// SetupChatRoutes 设置聊天相关路由
func SetupChatRoutes(router *gin.Engine) {
	chat := router.Group("/api/chat")
	{
		chat.POST("/message", SendMessage)
		chat.POST("/message/stream", SendMessageStream) // 新增流式接口
		chat.POST("/session", CreateSession)
		chat.GET("/sessions", GetSessions)
		chat.GET("/sessions/:session_id/messages", GetSessionMessages)
		chat.DELETE("/sessions/:session_id", DeleteSession)
	}
}

// fileService LLM_Chat_Service.FileServiceInterface,
// 在处理消息前添加文件内容
func processFilesWithMessage(sessionID string, message string, fileIDs []uint) (string, error) {
	if len(fileIDs) == 0 {
		return message, nil
	}

	var fileContents []string
	for _, fileID := range fileIDs {
		file, err := LLM_Chat_Service.GlobalFileService.GetFileByID(fileID)
		if err != nil {
			return "", fmt.Errorf("获取文件失败: %v", err)
		}

		content, err := LLM_Chat_Service.GlobalFileService.ProcessFileContent(file)
		if err == nil && content != "" {
			fileContents = append(fileContents, fmt.Sprintf("【文件：%s】\n%s\n", file.FileName, content))
		}
	}

	if len(fileContents) > 0 {
		fileSection := strings.Join(fileContents, "\n")
		if message != "" {
			return message + "\n\n" + fileSection, nil
		}
		return fileSection, nil
	}

	return message, nil
}

// SendMessage 原有的同步消息发送（保持不变）
func SendMessage(c *gin.Context) {
	var request struct {
		BaseUrl   string `json:"BaseUrl"`
		SessionID string `json:"session_id" binding:"required"`
		ModelName string `json:"model_name" binding:"required"`
		Message   string `json:"message" binding:"required"`
		Persona   string `json:"persona"`
		FileIDs   []uint `json:"file_ids"`
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "参数错误: " + err.Error(),
		})
		return
	}

	// 获取或创建会话
	session, err := LLM_Chat_Service.GetSessionManager().GetOrCreateSession(userID.(uint), request.SessionID, request.ModelName, request.BaseUrl, request.Persona)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取会话失败: " + err.Error(),
		})
		return
	}

	// 处理文件内容
	fullMessage, err := processFilesWithMessage(request.SessionID, request.Message, request.FileIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "处理文件内容失败: " + err.Error(),
		})
		return
	}

	// 保存用户消息到数据库
	if err := LLM_Chat_Service.GetSessionManager().SaveMessage(request.SessionID, "user", request.Message, userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "保存用户消息失败: " + err.Error(),
		})
		return
	}

	// 发送消息（使用包含文件内容的完整消息）
	response, err := session.SendMessage(fullMessage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "发送消息失败: " + err.Error(),
		})
		return
	}

	// 保存AI回复到数据库
	if err := LLM_Chat_Service.GetSessionManager().SaveMessage(request.SessionID, "assistant", response, userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "保存AI回复失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"response": response,
	})
}

// SendMessageStream 新增：流式发送消息
func SendMessageStream(c *gin.Context) {
	var request struct {
		BaseUrl   string `json:"BaseUrl"`
		SessionID string `json:"session_id" binding:"required"`
		ModelName string `json:"model_name" binding:"required"`
		Message   string `json:"message" binding:"required"`
		Persona   string `json:"persona"`
		FileIDs   []uint `json:"file_ids"`
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "参数错误: " + err.Error(),
		})
		return
	}

	// 获取或创建会话
	session, err := LLM_Chat_Service.GetSessionManager().GetOrCreateSession(userID.(uint), request.SessionID, request.ModelName, request.BaseUrl, request.Persona)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取会话失败: " + err.Error(),
		})
		return
	}

	// 处理文件内容
	fullMessage, err := processFilesWithMessage(request.SessionID, request.Message, request.FileIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "处理文件内容失败: " + err.Error(),
		})
		return
	}

	// 保存用户消息到数据库
	if err := LLM_Chat_Service.GetSessionManager().SaveMessage(request.SessionID, "user", request.Message, userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "保存用户消息失败: " + err.Error(),
		})
		return
	}

	// 设置响应头为流式传输
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Flush()

	// 用于保存完整的AI回复
	var fullResponse string

	// 使用流式发送消息（使用包含文件内容的完整消息）
	fullResponse, err = session.SendMessageStream(fullMessage, func(chunk string) error {
		// 构建SSE格式的数据
		data := map[string]interface{}{
			"content": chunk,
			"done":    false,
		}

		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}

		// 发送SSE格式数据
		fmt.Fprintf(c.Writer, "data: %s\n\n", jsonData)
		c.Writer.Flush()

		return nil
	})

	if err != nil {
		// 发送错误信息
		errorData := map[string]interface{}{
			"error": err.Error(),
			"done":  true,
		}
		jsonData, _ := json.Marshal(errorData)
		fmt.Fprintf(c.Writer, "data: %s\n\n", jsonData)
		c.Writer.Flush()
		return
	}

	// 保存AI回复到数据库
	if err := LLM_Chat_Service.GetSessionManager().SaveMessage(request.SessionID, "assistant", fullResponse, userID.(uint)); err != nil {
		errorData := map[string]interface{}{
			"error": "保存AI回复失败: " + err.Error(),
			"done":  true,
		}
		jsonData, _ := json.Marshal(errorData)
		fmt.Fprintf(c.Writer, "data: %s\n\n", jsonData)
		c.Writer.Flush()
		return
	}

	// 发送结束信号
	endData := map[string]interface{}{
		"content": "",
		"done":    true,
	}
	jsonData, _ := json.Marshal(endData)
	fmt.Fprintf(c.Writer, "data: %s\n\n", jsonData)
	c.Writer.Flush()
}

// createSession 创建新会话
func CreateSession(c *gin.Context) {
	var request struct {
		BaseUrl   string `json:"BaseUrl"`
		ModelName string `json:"model_name" binding:"required"`
		Persona   string `json:"persona"` // 新增：人格选择
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "参数错误: " + err.Error(),
		})
		return
	}

	// 生成会话ID
	sessionID := GenerateSessionID()

	// 预创建会话
	_, err := LLM_Chat_Service.GetSessionManager().GetOrCreateSession(userID.(uint), sessionID, request.ModelName, request.BaseUrl, request.Persona)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建会话失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
	})
}

// GetSessions  获取会话列表
func GetSessions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 获取分页参数
	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}

	sessions, total, err := LLM_Chat_Service.GetSessionManager().GetChatService().GetChatSessions(userID.(uint), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取会话列表失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": sessions,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

type PaginatedMessagesResponse struct {
	Data       []openai.ChatCompletionMessage `json:"data"`
	NextCursor uint                           `json:"next_cursor"`
	HasMore    bool                           `json:"has_more"`
	Total      int64                          `json:"total,omitempty"`
}

type MessageWithID struct {
	ID      uint   `json:"id"`
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GetSessionMessages 获取特定会话的消息
func GetSessionMessages(c *gin.Context) {
	sessionID := c.Param("session_id")

	// 从查询参数获取分页信息
	cursorStr := c.DefaultQuery("cursor", "0")
	limitStr := c.DefaultQuery("limit", "50")

	cursor, _ := strconv.ParseUint(cursorStr, 10, 32)
	limit, _ := strconv.Atoi(limitStr)

	// 调用 ChatService 获取数据库消息
	dbMessages, nextCursor, hasMore, err := LLM_Chat_Service.GetSessionManager().GetChatService().GetChatMessages(sessionID, uint(cursor), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取消息失败: " + err.Error(),
		})
		return
	}

	// 转换为带 ID 的消息结构
	messages := make([]MessageWithID, len(dbMessages))
	for i, msg := range dbMessages {
		messages[i] = MessageWithID{
			ID:      msg.ID,
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     messages,
		"cursor":   nextCursor,
		"has_more": hasMore,
	})
}

// DeleteSession 删除会话
func DeleteSession(c *gin.Context) {
	sessionID := c.Param("session_id")

	// 从内存中移除并删除数据库记录
	err := LLM_Chat_Service.GetSessionManager().DeleteSession(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "删除会话失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})
}

// GenerateSessionID 生成会话ID
func GenerateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}
