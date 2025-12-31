package LLM_Chat

import (
	"fmt"
	"github.com/sashabaranov/go-openai"
	"log"
	"time"
)

type SessionCreatorInterface interface {
	CreateSession(apiKey, systemPrompt, BaseUrl string, maxHistory int) LLMSessionInterface
	CreateSessionFromHistory(apiKey, systemPrompt, BaseUrl string, maxHistory int, existingMessages []openai.ChatCompletionMessage) LLMSessionInterface
}

type DefaultSessionCreator struct{}

var GlobalDefaultSessionCreator SessionCreatorInterface

func NewDefaultSessionCreator() SessionCreatorInterface {
	service := &DefaultSessionCreator{}
	GlobalDefaultSessionCreator = service
	return service
}

func (d *DefaultSessionCreator) CreateSession(apiKey, systemPrompt, BaseUrl string, maxHistory int) LLMSessionInterface {
	return NewAdvancedChatSession(apiKey, systemPrompt, BaseUrl, maxHistory)
}

func (d *DefaultSessionCreator) CreateSessionFromHistory(apiKey, systemPrompt, BaseUrl string, maxHistory int, existingMessages []openai.ChatCompletionMessage) LLMSessionInterface {
	return NewAdvancedChatSessionFromHistory(apiKey, systemPrompt, BaseUrl, maxHistory, existingMessages)
}

// InitSessionManager 初始化会话管理器
func InitSessionManager(
	chatService ChatServiceInterface,
	cacheService CacheServiceInterface,
	modelService UserAPIServiceInterface,
	personaManager PersonaManagerInterface,
) {
	GlobalSessionManager = &SessionManager{
		sessions:       make(map[string]LLMSessionInterface),
		chatService:    chatService,
		cacheService:   cacheService,
		modelService:   modelService,
		personaManager: personaManager,
		sessionCreator: &DefaultSessionCreator{},
	}
}

// GetSessionManager 获取会话管理器实例
func GetSessionManager() *SessionManager {
	if GlobalSessionManager == nil {
		log.Fatal("SessionManager 未初始化，请先调用 InitSessionManager")
	}
	return GlobalSessionManager
}

// GetOrCreateSession 获取或创建会话
func (sm *SessionManager) GetOrCreateSession(userID uint, sessionID, modelName, BaseUrl, persona string) (LLMSessionInterface, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 检查内存中是否已存在
	if session, exists := sm.sessions[sessionID]; exists {
		// 如果指定了人格且与当前不同，更新系统提示词
		if persona != "" {
			systemPrompt := sm.personaManager.GetPersonaContent(persona)
			if systemPrompt != "" {
				session.SetSystemPrompt(systemPrompt)
			}
		}
		return session, nil
	}

	// 获取人格对应的系统提示词
	systemPrompt := sm.personaManager.GetPersonaContent(persona)

	// 尝试从缓存加载完整会话
	if sm.cacheService != nil {
		cachedFullSession, err := sm.cacheService.GetCachedFullSession(sessionID)
		if err == nil && cachedFullSession != nil {
			session := sm.sessionCreator.CreateSessionFromHistory(
				cachedFullSession.ModelAPIKey,
				systemPrompt,
				BaseUrl,
				10,
				cachedFullSession.Messages,
			)
			session.SetSessionID(sessionID)
			sm.sessions[sessionID] = session
			log.Printf("从缓存恢复会话: %s", sessionID)
			return session, nil
		} else if err != nil && err.Error() != "redis不可用" {
			log.Printf("从缓存获取会话失败: %v", err)
		}
	}

	// 从数据库获取模型配置
	model, err := sm.modelService.GetAPIByModelName(userID, modelName)
	if err != nil {
		return nil, fmt.Errorf("获取模型配置失败: %v", err)
	}

	// 如果 BaseUrl 为空，使用模型配置中的 BaseURL
	if BaseUrl == "" && model.BaseURL != "" {
		BaseUrl = model.BaseURL
	}

	// 创建数据库会话记录
	dbSession, err := sm.chatService.CreateChatSession(sessionID, modelName, userID)
	if err != nil {
		return nil, fmt.Errorf("创建会话记录失败: %v", err)
	}

	// 从数据库加载历史消息
	existingMessages, err := sm.chatService.GetChatMessages(sessionID)
	if err != nil {
		return nil, fmt.Errorf("加载历史消息失败: %v", err)
	}

	var session LLMSessionInterface
	if len(existingMessages) > 0 {
		// 从历史消息创建会话
		session = sm.sessionCreator.CreateSessionFromHistory(
			model.APIKey,
			systemPrompt,
			BaseUrl,
			10,
			existingMessages,
		)
	} else {
		// 创建新会话
		session = sm.sessionCreator.CreateSession(
			model.APIKey,
			systemPrompt,
			BaseUrl,
			10,
		)
	}

	session.SetSessionID(sessionID)
	sm.sessions[sessionID] = session

	// 缓存完整会话状态
	if sm.cacheService != nil {
		cachedFullSession := &CachedSession{
			Session:     dbSession,
			Messages:    existingMessages,
			ModelAPIKey: model.APIKey,
			BaseUrl:     BaseUrl,
		}
		if err := sm.cacheService.CacheFullSession(sessionID, cachedFullSession, 1*time.Hour); err != nil && err.Error() != "redis不可用" {
			log.Printf("缓存会话失败: %v", err)
		}
	}

	return session, nil
}

// GetSession 获取会话（不创建）
func (sm *SessionManager) GetSession(sessionID string) (LLMSessionInterface, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	return session, exists
}

// SaveMessage 保存消息到数据库
func (sm *SessionManager) SaveMessage(sessionID, role, content string, userID uint) error {
	return sm.chatService.SaveChatMessage(sessionID, role, content, userID)
}

// DeleteSession 从内存中删除会话
func (sm *SessionManager) DeleteSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 从内存删除
	delete(sm.sessions, sessionID)

	// 从数据库删除
	if err := sm.chatService.DeleteChatSession(sessionID); err != nil {
		return err
	}

	log.Printf("删除会话成功: %s", sessionID)
	return nil
}

// GenerateSessionID 生成会话ID
func GenerateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

// GetChatService 获取聊天服务（用于路由中直接访问）
func (sm *SessionManager) GetChatService() ChatServiceInterface {
	return sm.chatService
}

// GetAvailablePersonas 新增：获取可用人格列表
func (sm *SessionManager) GetAvailablePersonas() []string {
	return sm.personaManager.GetAvailablePersonas()
}
