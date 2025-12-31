package LLM_Chat

import (
	"errors"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"gorm.io/gorm"
	"log"
	"platfrom/database"
	"sync"
)

type ChatServiceInterface interface {
	CreateChatSession(sessionID, modelName string, UserId uint) (*database.ChatSession, error)
	SaveChatMessage(sessionID, role, content string, UserId uint) error
	GetChatMessages(sessionID string) ([]openai.ChatCompletionMessage, error)
	GetChatSessions(UserId uint) ([]database.ChatSession, error)
	GetChatSession(sessionID string, UserId uint) (*database.ChatSession, error)
	DeleteChatSession(sessionID string) error
	UpdateSessionTitle(sessionID, title string) error
}

var GlobalChatService ChatServiceInterface

type ChatSessionService struct {
	db *gorm.DB
	mu sync.RWMutex
}

func NewChatService() ChatServiceInterface {
	service := &ChatSessionService{
		db: database.DB,
	}
	GlobalChatService = service
	return service
}

// CreateChatSession 创建聊天会话
func (s *ChatSessionService) CreateChatSession(sessionID, modelName string, UserId uint) (*database.ChatSession, error) {
	if sessionID == "" || modelName == "" || UserId == 0 { // 修改这里：UserId == 0
		return nil, errors.New("sessionID、modelName 和 UserId 不能为空")
	}

	// 检查是否已存在
	var existingSession database.ChatSession
	result := s.db.Where("session_id = ? AND user_id = ?", sessionID, UserId).First(&existingSession)
	if result.RowsAffected > 0 {
		return &existingSession, nil
	}

	// 创建新会话时生成更有意义的标题
	title := fmt.Sprintf("与 %s 的对话", modelName)

	// 创建新会话
	session := &database.ChatSession{
		SessionID:    sessionID,
		ModelName:    modelName,
		Title:        title,
		UserID:       UserId,
		MessageCount: 0,
	}

	result = s.db.Create(session)
	if result.Error != nil {
		return nil, result.Error
	}

	log.Printf("创建聊天会话成功: %s, 标题: %s", sessionID, title)
	return session, nil
}

// SaveChatMessage 保存聊天消息
func (s *ChatSessionService) SaveChatMessage(sessionID, role, content string, UserId uint) error {
	if sessionID == "" || role == "" || content == "" {
		return errors.New("sessionID, role 和 content 不能为空")
	}

	message := &database.ChatMessage{
		SessionID: sessionID,
		Role:      role,
		Content:   content,
	}

	result := s.db.Create(message)
	if result.Error != nil {
		return result.Error
	}

	// 更新会话的消息计数
	updateResult := s.db.Model(&database.ChatSession{}).
		Where("session_id = ? AND user_id = ?", sessionID, UserId).
		Updates(map[string]interface{}{
			"message_count": gorm.Expr("message_count + ?", 1),
			"updated_at":    gorm.Expr("CURRENT_TIMESTAMP"),
		})
	if updateResult.Error != nil {
		log.Printf("更新消息计数失败: %v", updateResult.Error)
	}

	// 如果这是第一条用户消息，自动生成标题
	if role == "user" {
		var messageCount int64
		s.db.Model(&database.ChatMessage{}).
			Where("session_id = ? AND role = ?", sessionID, "user").
			Count(&messageCount)

		if messageCount == 1 {
			// 使用用户的第一条消息作为标题（截断）
			title := content
			if len(title) > 50 {
				title = title[:50] + "..."
			}
			s.db.Model(&database.ChatSession{}).
				Where("session_id = ?", sessionID).
				Update("title", title)
		}
	}

	return nil
}

// GetChatMessages 获取会话的所有消息
func (s *ChatSessionService) GetChatMessages(sessionID string) ([]openai.ChatCompletionMessage, error) {
	if sessionID == "" {
		return nil, errors.New("sessionID 不能为空")
	}

	var messages []database.ChatMessage
	result := s.db.Where("session_id = ?", sessionID).Order("created_at").Find(&messages)
	if result.Error != nil {
		return nil, result.Error
	}

	chatMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		chatMessages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return chatMessages, nil
}

// GetChatSessions 获取指定用户的所有聊天会话
func (s *ChatSessionService) GetChatSessions(UserId uint) ([]database.ChatSession, error) {
	if UserId == 0 {
		return nil, errors.New("UserId 不能为空")
	}

	var sessions []database.ChatSession
	result := s.db.Where("user_id = ?", UserId).Order("updated_at DESC").Find(&sessions)
	if result.Error != nil {
		return nil, result.Error
	}

	return sessions, nil
}

// GetChatSession 获取特定会话
func (s *ChatSessionService) GetChatSession(sessionID string, UserId uint) (*database.ChatSession, error) {
	if sessionID == "" {
		return nil, errors.New("sessionID 不能为空")
	}

	var session database.ChatSession
	result := s.db.Where("session_id = ? AND user_id = ?", sessionID, UserId).First(&session)
	if result.Error != nil {
		return nil, result.Error
	}

	return &session, nil
}

// DeleteChatSession 删除聊天会话及其所有消息
func (s *ChatSessionService) DeleteChatSession(sessionID string) error {
	if sessionID == "" {
		return errors.New("sessionID 不能为空")
	}
	// 开启事务
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除所有相关消息
		if err := tx.Where("session_id = ?", sessionID).Delete(&database.ChatMessage{}).Error; err != nil {
			return err
		}
		// 删除会话
		if err := tx.Where("session_id = ?", sessionID).Delete(&database.ChatSession{}).Error; err != nil {
			return err
		}
		log.Printf("删除聊天会话成功: %s", sessionID)
		return nil
	})
}

// UpdateSessionTitle 更新会话标题
func (s *ChatSessionService) UpdateSessionTitle(sessionID, title string) error {
	if sessionID == "" || title == "" {
		return errors.New("sessionID 和 title 不能为空")
	}

	result := s.db.Model(&database.ChatSession{}).
		Where("session_id = ?", sessionID).
		Update("title", title)

	return result.Error
}
