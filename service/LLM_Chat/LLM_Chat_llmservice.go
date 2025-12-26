package LLM_Chat

import (
	"context"
	"errors"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"gorm.io/gorm"
	"io"
	"platfrom/database"
	"strings"
	"sync"
)

// LLMService LLM服务接口（更完整的设计）
type LLMService interface {
	// ChatStream 聊天相关
	ChatStream(sessionID string, userMessage string, onChunk func(chunk string) error) error
	Chat(sessionID string, userMessage string) (string, error)

	// BuildContext 上下文管理
	BuildContext(sessionID string, maxMessages int) ([]openai.ChatCompletionMessage, error)

	// SetSessionPersona 会话配置
	SetSessionPersona(sessionID, personaName string) error
	SetSessionAPI(sessionID string, apiConfig *database.UserAPI) error

	// TestConnection 工具方法
	TestConnection(apiConfig *database.UserAPI) (bool, error)
}

// llmServiceImpl LLM服务实现
type llmServiceImpl struct {
	db             *gorm.DB
	messageService ChatMessageService
	sessionService ChatSessionService
	configManager  ConfigManager
	userAPIService UserAPIService
	mu             sync.RWMutex
	sessionConfigs map[string]*sessionConfig // 会话配置缓存
}

type sessionConfig struct {
	apiConfig  *database.UserAPI
	persona    *database.Persona
	maxHistory int
}

// NewLLMService 创建LLM服务
func NewLLMService() LLMService {
	return &llmServiceImpl{
		db:             database.DB,
		messageService: GlobalChatMessageService,
		sessionService: GlobalChatSessionService,
		configManager:  GlobalConfigManager,
		userAPIService: GlobalUserAPIService,
		sessionConfigs: make(map[string]*sessionConfig),
	}
}

// ChatStream 流式聊天（符合实际使用场景）
func (s *llmServiceImpl) ChatStream(sessionID string, userMessage string, onChunk func(chunk string) error) error {
	// 1. 验证会话
	session, err := s.sessionService.GetSessionByID(sessionID)
	if err != nil {
		return fmt.Errorf("会话不存在: %w", err)
	}

	// 2. 获取会话配置
	config, err := s.getSessionConfig(sessionID)
	if err != nil {
		return fmt.Errorf("获取会话配置失败: %w", err)
	}

	// 3. 创建用户消息
	_, err = s.messageService.CreateMessage(sessionID, session.UserID, "user", userMessage)
	if err != nil {
		return fmt.Errorf("创建用户消息失败: %w", err)
	}

	// 4. 构建上下文
	messages, err := s.BuildContext(sessionID, config.maxHistory)
	if err != nil {
		return fmt.Errorf("构建上下文失败: %w", err)
	}

	// 5. 如果配置了人格，添加系统消息
	if config.persona != nil {
		systemMsg := openai.ChatCompletionMessage{
			Role:    "system",
			Content: config.persona.Content,
		}
		messages = append([]openai.ChatCompletionMessage{systemMsg}, messages...)
	}

	// 6. 调用LLM API
	client := s.createClient(config.apiConfig)
	streamReq := openai.ChatCompletionRequest{
		Model:    config.apiConfig.ModelName,
		Messages: messages,
		Stream:   true,
	}

	stream, err := client.CreateChatCompletionStream(context.Background(), streamReq)
	if err != nil {
		return fmt.Errorf("创建流式请求失败: %w", err)
	}
	defer stream.Close()

	// 7. 处理流式响应
	var fullResponse strings.Builder
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("接收流数据失败: %w", err)
		}

		if len(response.Choices) > 0 && response.Choices[0].Delta.Content != "" {
			chunk := response.Choices[0].Delta.Content
			fullResponse.WriteString(chunk)

			if onChunk != nil {
				if err := onChunk(chunk); err != nil {
					return fmt.Errorf("处理数据块失败: %w", err)
				}
			}
		}
	}

	// 8. 保存助手响应
	_, err = s.messageService.CreateMessage(sessionID, session.UserID, "assistant", fullResponse.String())
	if err != nil {
		return fmt.Errorf("保存助手消息失败: %w", err)
	}

	return nil
}

// Chat 非流式聊天
func (s *llmServiceImpl) Chat(sessionID string, userMessage string) (string, error) {
	var response strings.Builder
	err := s.ChatStream(sessionID, userMessage, func(chunk string) error {
		response.WriteString(chunk)
		return nil
	})

	return response.String(), err
}

// BuildContext 构建聊天上下文
func (s *llmServiceImpl) BuildContext(sessionID string, maxMessages int) ([]openai.ChatCompletionMessage, error) {
	dbMessages, err := s.messageService.GetSessionContext(sessionID, maxMessages)
	if err != nil {
		return nil, err
	}

	var messages []openai.ChatCompletionMessage
	for _, msg := range dbMessages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return messages, nil
}

// SetSessionPersona 设置会话人格
func (s *llmServiceImpl) SetSessionPersona(sessionID, personaName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	persona, err := s.configManager.GetPersona(personaName)
	if err != nil {
		return err
	}

	// 更新会话配置
	if config, exists := s.sessionConfigs[sessionID]; exists {
		config.persona = persona
	} else {
		s.sessionConfigs[sessionID] = &sessionConfig{
			persona: persona,
		}
	}

	// 更新数据库中的会话人格
	return s.sessionService.UpdateSessionPersona(sessionID, personaName)
}

// SetSessionAPI 设置会话使用的API
func (s *llmServiceImpl) SetSessionAPI(sessionID string, apiConfig *database.UserAPI) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 更新会话配置
	if config, exists := s.sessionConfigs[sessionID]; exists {
		config.apiConfig = apiConfig
	} else {
		s.sessionConfigs[sessionID] = &sessionConfig{
			apiConfig: apiConfig,
		}
	}

	return nil
}

// getSessionConfig 获取或初始化会话配置
func (s *llmServiceImpl) getSessionConfig(sessionID string) (*sessionConfig, error) {
	s.mu.RLock()
	config, exists := s.sessionConfigs[sessionID]
	s.mu.RUnlock()

	if exists {
		return config, nil
	}

	// 初始化配置
	s.mu.Lock()
	defer s.mu.Unlock()

	// 再次检查，防止并发重复初始化
	if config, exists := s.sessionConfigs[sessionID]; exists {
		return config, nil
	}

	// 获取会话信息
	session, err := s.sessionService.GetSessionByID(sessionID)
	if err != nil {
		return nil, err
	}

	// 获取默认API
	api, err := s.userAPIService.GetFirstAvailableAPI(session.UserID)
	if err != nil {
		return nil, fmt.Errorf("用户未配置API: %w", err)
	}

	// 获取人格配置
	var persona *database.Persona
	if session.PersonaName != "" {
		persona, _ = s.configManager.GetPersona(session.PersonaName)
	}

	// 创建配置
	config = &sessionConfig{
		apiConfig:  api,
		persona:    persona,
		maxHistory: 10, // 默认10条历史记录
	}

	s.sessionConfigs[sessionID] = config
	return config, nil
}

// createClient 创建OpenAI客户端
func (s *llmServiceImpl) createClient(apiConfig *database.UserAPI) *openai.Client {
	config := openai.DefaultConfig(apiConfig.APIKey)
	if apiConfig.BaseURL != "" {
		config.BaseURL = apiConfig.BaseURL
	}
	return openai.NewClientWithConfig(config)
}

// TestConnection 测试API连接
func (s *llmServiceImpl) TestConnection(apiConfig *database.UserAPI) (bool, error) {
	client := s.createClient(apiConfig)

	// 发送一个简单的请求测试
	req := openai.ChatCompletionRequest{
		Model: apiConfig.ModelName,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "user",
				Content: "Hello",
			},
		},
	}

	_, err := client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		return false, fmt.Errorf("连接测试失败: %w", err)
	}

	return true, nil
}
