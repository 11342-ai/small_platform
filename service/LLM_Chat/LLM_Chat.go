package LLM_Chat

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"platfrom/database"
	"sync"
	"time"
)

// ChatSessionService 聊天会话服务接口
type ChatSessionService interface {
	// CreateSession 会话管理
	CreateSession(userID uint, title string, personaName string) (*database.ChatSession, error)
	GetSessionByID(sessionID string) (*database.ChatSession, error)
	GetUserSessions(userID uint) ([]database.ChatSession, error)
	GetUserSessionsByPage(userID uint, page, pageSize int) ([]database.ChatSession, int64, error)
	UpdateSession(sessionID string, updates map[string]interface{}) error
	UpdateSessionTitle(sessionID, title string) error
	UpdateSessionPersona(sessionID, personaName string) error
	DeleteSession(sessionID string) error
	DeleteUserSessions(userID uint) error

	UpdateLastMessageTime(sessionID string) error
	IncrementMessageCount(sessionID string) error
}

// GlobalChatSessionService 全局ChatSessionService实例
var GlobalChatSessionService ChatSessionService

// chatSessionService 聊天会话服务实现
type chatSessionService struct {
	db *gorm.DB
	mu sync.RWMutex
}

// NewChatSessionService 创建新的聊天会话服务
func NewChatSessionService() ChatSessionService {
	service := &chatSessionService{
		db: database.DB,
	}
	GlobalChatSessionService = service
	return service
}

// generateSessionID 生成会话ID（使用UUID）
func generateSessionID() string {
	return uuid.New().String()
}

// CreateSession 创建新会话
func (s *chatSessionService) CreateSession(userID uint, title string, personaName string) (*database.ChatSession, error) {
	if userID == 0 {
		return nil, errors.New("用户ID不能为空")
	}
	if title == "" {
		// 可以设置默认标题
		title = "新对话"
	}
	// 检查用户是否存在（可选）
	var user database.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	// 验证人格是否存在（如果配置了人格管理器）
	if personaName != "" {
		if _, err := GlobalConfigManager.GetPersona(personaName); err != nil {
			// 如果人格不存在，使用默认人格
			defaultPersona, err := GlobalConfigManager.GetDefaultPersona()
			if err != nil {
				personaName = ""
			} else {
				personaName = defaultPersona.Name
			}
		}
	}
	session := &database.ChatSession{
		SessionID:     generateSessionID(),
		UserID:        userID,
		Title:         title,
		PersonaName:   personaName,
		LastMessageAt: time.Now(),
		MessageCount:  0,
	}
	if err := s.db.Create(session).Error; err != nil {
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}
	return session, nil
}

// GetSessionByID 根据ID获取会话
func (s *chatSessionService) GetSessionByID(sessionID string) (*database.ChatSession, error) {
	var session database.ChatSession
	if err := s.db.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("会话不存在")
		}
		return nil, fmt.Errorf("查询会话失败: %w", err)
	}

	return &session, nil
}

// GetUserSessions 获取用户的所有会话
func (s *chatSessionService) GetUserSessions(userID uint) ([]database.ChatSession, error) {
	var sessions []database.ChatSession
	if err := s.db.Where("user_id = ?", userID).
		Order("last_message_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("查询用户会话失败: %w", err)
	}

	return sessions, nil
}

// GetUserSessionsByPage 分页获取用户会话
func (s *chatSessionService) GetUserSessionsByPage(userID uint, page, pageSize int) ([]database.ChatSession, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 30 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var sessions []database.ChatSession
	var total int64
	// 获取总数
	if err := s.db.Model(&database.ChatSession{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计会话数量失败: %w", err)
	}
	// 获取分页数据
	if err := s.db.Where("user_id = ?", userID).
		Order("last_message_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&sessions).Error; err != nil {
		return nil, 0, fmt.Errorf("查询用户会话失败: %w", err)
	}
	return sessions, total, nil
}

// UpdateSession 更新会话
func (s *chatSessionService) UpdateSession(sessionID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.New("更新内容不能为空")
	}

	// 检查会话是否存在
	var session database.ChatSession
	if err := s.db.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("会话不存在")
		}
		return fmt.Errorf("查询会话失败: %w", err)
	}

	// 更新会话
	if err := s.db.Model(&database.ChatSession{}).Where("session_id = ?", sessionID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新会话失败: %w", err)
	}

	return nil
}

// UpdateSessionTitle 更新会话标题
func (s *chatSessionService) UpdateSessionTitle(sessionID, title string) error {
	if title == "" {
		return errors.New("标题不能为空")
	}

	return s.UpdateSession(sessionID, map[string]interface{}{
		"title": title,
	})
}

// UpdateSessionPersona 更新会话人格
func (s *chatSessionService) UpdateSessionPersona(sessionID, personaName string) error {
	// 验证人格是否存在（如果配置了人格管理器）
	if personaName != "" {
		if _, err := GlobalConfigManager.GetPersona(personaName); err != nil {
			return fmt.Errorf("人格不存在: %s", personaName)
		}
	}

	return s.UpdateSession(sessionID, map[string]interface{}{
		"persona_name": personaName,
	})
}

// DeleteSession 删除会话
func (s *chatSessionService) DeleteSession(sessionID string) error {
	// 检查会话是否存在
	var session database.ChatSession
	if err := s.db.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("会话不存在")
		}
		return fmt.Errorf("查询会话失败: %w", err)
	}
	// 删除会话（注意：这里可能需要级联删除消息，由数据库或应用层处理）
	if err := s.db.Where("session_id = ?", sessionID).Delete(&database.ChatSession{}).Error; err != nil {
		return fmt.Errorf("删除会话失败: %w", err)
	}

	return nil
}

// DeleteUserSessions 删除用户的所有会话
func (s *chatSessionService) DeleteUserSessions(userID uint) error {
	if userID == 0 {
		return errors.New("用户ID不能为空")
	}
	// 删除用户的所有会话
	if err := s.db.Where("user_id = ?", userID).Delete(&database.ChatSession{}).Error; err != nil {
		return fmt.Errorf("删除用户会话失败: %w", err)
	}

	return nil
}

// UpdateLastMessageTime 更新会话的最后消息时间
func (s *chatSessionService) UpdateLastMessageTime(sessionID string) error {
	return s.UpdateSession(sessionID, map[string]interface{}{
		"last_message_at": time.Now(),
	})
}

// IncrementMessageCount 增加会话的消息计数
func (s *chatSessionService) IncrementMessageCount(sessionID string) error {
	// 使用原子操作增加消息计数
	if err := s.db.Model(&database.ChatSession{}).
		Where("session_id = ?", sessionID).
		Update("message_count", gorm.Expr("message_count + ?", 1)).Error; err != nil {
		return fmt.Errorf("更新消息计数失败: %w", err)
	}

	return nil
}

// = = = = = = = = = = = = = = = =

// ChatMessageService 聊天消息服务接口
type ChatMessageService interface {
	CreateMessage(sessionID string, userID uint, role, content string) (*database.ChatMessage, error)
	GetSessionMessages(sessionID string) ([]database.ChatMessage, error)
	GetSessionMessagesByPage(sessionID string, page, pageSize int) ([]database.ChatMessage, int64, error)
	DeleteSessionMessages(sessionID string) error
	DeleteUserMessages(userID uint) error
	GetSessionContext(sessionID string, maxMessages int) ([]database.ChatMessage, error)
	GetNextMessageOrder(sessionID string) (int, error)
}

// GlobalChatMessageService 全局ChatMessageService实例
var GlobalChatMessageService ChatMessageService

// chatMessageService 聊天消息服务实现
type chatMessageService struct {
	db *gorm.DB
	mu sync.RWMutex
}

// MessageStat 消息统计结构体
type MessageStat struct {
	Date           string `json:"date"`
	MessageCount   int64  `json:"message_count"`
	UserCount      int64  `json:"user_count"`
	AssistantCount int64  `json:"assistant_count"`
}

// NewChatMessageService 创建新的聊天消息服务
func NewChatMessageService() ChatMessageService {
	service := &chatMessageService{
		db: database.DB,
	}
	GlobalChatMessageService = service
	return service
}

// CreateMessage 创建消息
func (s *chatMessageService) CreateMessage(sessionID string, userID uint, role, content string) (*database.ChatMessage, error) {
	if sessionID == "" {
		return nil, errors.New("会话ID不能为空")
	}

	if userID == 0 {
		return nil, errors.New("用户ID不能为空")
	}

	if role == "" {
		role = "user" // 默认角色
	}

	if content == "" {
		return nil, errors.New("消息内容不能为空")
	}

	// 验证角色
	if role != "user" && role != "assistant" && role != "system" {
		return nil, errors.New("无效的消息角色")
	}

	// 获取下一个消息顺序号
	messageOrder, err := s.GetNextMessageOrder(sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取消息顺序失败: %w", err)
	}

	// 创建消息
	message := &database.ChatMessage{
		SessionID:    sessionID,
		UserID:       userID,
		Role:         role,
		Content:      content,
		MessageOrder: messageOrder,
	}

	if err := s.db.Create(message).Error; err != nil {
		return nil, fmt.Errorf("创建消息失败: %w", err)
	}

	// 更新会话的最后消息时间
	if err := GlobalChatSessionService.UpdateLastMessageTime(sessionID); err != nil {
		// 如果更新会话时间失败，记录错误但不影响消息创建
		fmt.Printf("警告：更新会话最后消息时间失败: %v\n", err)
	}

	// 增加会话的消息计数
	if err := GlobalChatSessionService.IncrementMessageCount(sessionID); err != nil {
		fmt.Printf("警告：更新会话消息计数失败: %v\n", err)
	}

	return message, nil
}

// GetSessionMessages 获取会话的所有消息
func (s *chatMessageService) GetSessionMessages(sessionID string) ([]database.ChatMessage, error) {
	var messages []database.ChatMessage
	if err := s.db.Where("session_id = ?", sessionID).
		Order("message_order ASC").
		Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("查询会话消息失败: %w", err)
	}

	return messages, nil
}

// GetSessionMessagesByPage 分页获取会话消息
func (s *chatMessageService) GetSessionMessagesByPage(sessionID string, page, pageSize int) ([]database.ChatMessage, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	var messages []database.ChatMessage
	var total int64

	// 获取总数
	if err := s.db.Model(&database.ChatMessage{}).Where("session_id = ?", sessionID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计消息数量失败: %w", err)
	}

	// 获取分页数据
	if err := s.db.Where("session_id = ?", sessionID).
		Order("message_order DESC"). // 最新的消息在前
		Offset(offset).Limit(pageSize).
		Find(&messages).Error; err != nil {
		return nil, 0, fmt.Errorf("查询会话消息失败: %w", err)
	}

	return messages, total, nil
}

// DeleteSessionMessages 删除会话的所有消息
func (s *chatMessageService) DeleteSessionMessages(sessionID string) error {
	if sessionID == "" {
		return errors.New("会话ID不能为空")
	}

	// 删除会话的所有消息
	if err := s.db.Where("session_id = ?", sessionID).Delete(&database.ChatMessage{}).Error; err != nil {
		return fmt.Errorf("删除会话消息失败: %w", err)
	}

	return nil
}

// DeleteUserMessages 删除用户的所有消息
func (s *chatMessageService) DeleteUserMessages(userID uint) error {
	if userID == 0 {
		return errors.New("用户ID不能为空")
	}

	// 删除用户的所有消息
	if err := s.db.Where("user_id = ?", userID).Delete(&database.ChatMessage{}).Error; err != nil {
		return fmt.Errorf("删除用户消息失败: %w", err)
	}

	return nil
}

// GetSessionContext 获取会话上下文（最多返回指定数量的消息）
func (s *chatMessageService) GetSessionContext(sessionID string, maxMessages int) ([]database.ChatMessage, error) {
	if maxMessages < 1 || maxMessages > 100 {
		maxMessages = 20 // 默认最多10条消息
	}

	var messages []database.ChatMessage
	if err := s.db.Where("session_id = ? AND role IN ('user', 'assistant')", sessionID).
		Order("message_order DESC").
		Limit(maxMessages).
		Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("获取会话上下文失败: %w", err)
	}

	// 反转顺序，使最早的在前
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetNextMessageOrder 获取下一个消息顺序号
func (s *chatMessageService) GetNextMessageOrder(sessionID string) (int, error) {
	var maxOrder int

	// 查询当前会话的最大顺序号
	if err := s.db.Model(&database.ChatMessage{}).
		Where("session_id = ?", sessionID).
		Select("COALESCE(MAX(message_order), 0)").
		Scan(&maxOrder).Error; err != nil {
		return 0, fmt.Errorf("获取最大顺序号失败: %w", err)
	}

	return maxOrder + 1, nil
}

// MessageCreateRequest 辅助结构体
type MessageCreateRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Role      string `json:"role" binding:"required,oneof=user assistant system"`
	Content   string `json:"content" binding:"required"`
}

type MessageResponse struct {
	ID           uint      `json:"id"`
	SessionID    string    `json:"session_id"`
	UserID       uint      `json:"user_id"`
	Role         string    `json:"role"`
	Content      string    `json:"content"`
	MessageOrder int       `json:"message_order"`
	CreatedAt    time.Time `json:"created_at"`
}
