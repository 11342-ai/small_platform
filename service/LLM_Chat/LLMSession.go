package LLM_Chat

import (
	"context"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"io"
	"strings"
)

// LLMSessionInterface 消息管理
type LLMSessionInterface interface {
	SetSessionID(sessionID string)
	GetMessages() []openai.ChatCompletionMessage
	SendMessage(message string) (string, error)
	SendMessageStream(message string, onChunk func(chunk string) error) (string, error)
	SetSystemPrompt(prompt string)
}

type AdvancedChatSession struct {
	APIKey       string
	Client       *openai.Client
	Messages     []openai.ChatCompletionMessage
	MaxHistory   int
	SystemPrompt string
	SessionID    string
}

var GlobalLLMSession LLMSessionInterface

func NewLLMSession() LLMSessionInterface {
	service := &AdvancedChatSession{}
	GlobalLLMSession = service
	return service
}

func NewAdvancedChatSession(apiKey, systemPrompt, BaseUrl string, maxHistory int) LLMSessionInterface {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = BaseUrl
	session := &AdvancedChatSession{
		APIKey:       apiKey,
		Client:       openai.NewClientWithConfig(config),
		Messages:     make([]openai.ChatCompletionMessage, 0),
		MaxHistory:   maxHistory,
		SystemPrompt: systemPrompt,
	}

	GlobalLLMSession = session
	return session
}

func NewAdvancedChatSessionFromHistory(apiKey, systemPrompt, BaseUrl string, maxHistory int, existingMessages []openai.ChatCompletionMessage) LLMSessionInterface {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = BaseUrl
	session := &AdvancedChatSession{
		APIKey:       apiKey,
		Client:       openai.NewClientWithConfig(config),
		Messages:     existingMessages,
		MaxHistory:   maxHistory,
		SystemPrompt: systemPrompt,
	}

	GlobalLLMSession = session
	return session
}

// SetSessionID 新增：设置会话ID
func (s *AdvancedChatSession) SetSessionID(sessionID string) {
	s.SessionID = sessionID
}

// GetMessages 新增：获取当前消息历史
func (s *AdvancedChatSession) GetMessages() []openai.ChatCompletionMessage {
	return s.Messages
}

// SendMessage 原有的同步发送消息方法
func (s *AdvancedChatSession) SendMessage(message string) (string, error) {
	// 添加用户消息
	s.Messages = append(s.Messages, openai.ChatCompletionMessage{
		Role:    "user",
		Content: message,
	})

	// 限制历史记录长度（保留系统消息）
	startIndex := 0
	if len(s.Messages) > 0 && s.Messages[0].Role == "system" {
		startIndex = 1 // 保留系统消息
	}

	if len(s.Messages) > s.MaxHistory*2+startIndex {
		// 保留系统消息和最近的对话
		s.Messages = append(
			s.Messages[:startIndex],
			s.Messages[len(s.Messages)-s.MaxHistory*2:]...,
		)
	}

	resp, err := s.Client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    "deepseek-chat",
			Messages: s.Messages,
		},
	)

	if err != nil {
		s.Messages = s.Messages[:len(s.Messages)-1] // 移除失败的用户消息
		return "", fmt.Errorf("ChatCompletion error: %v", err)
	}

	aiResponse := resp.Choices[0].Message.Content

	// 添加AI回复
	s.Messages = append(s.Messages, openai.ChatCompletionMessage{
		Role:    "assistant",
		Content: aiResponse,
	})

	return aiResponse, nil
}

// SendMessageStream 新增：流式发送消息
func (s *AdvancedChatSession) SendMessageStream(message string, onChunk func(chunk string) error) (string, error) {
	// 添加用户消息
	s.Messages = append(s.Messages, openai.ChatCompletionMessage{
		Role:    "user",
		Content: message,
	})

	// 限制历史记录长度（保留系统消息）
	startIndex := 0
	if len(s.Messages) > 0 && s.Messages[0].Role == "system" {
		startIndex = 1 // 保留系统消息
	}

	if len(s.Messages) > s.MaxHistory*2+startIndex {
		// 保留系统消息和最近的对话
		s.Messages = append(
			s.Messages[:startIndex],
			s.Messages[len(s.Messages)-s.MaxHistory*2:]...,
		)
	}

	// 创建流式请求
	req := openai.ChatCompletionRequest{
		Model:    "deepseek-chat",
		Messages: s.Messages,
		Stream:   true,
	}

	stream, err := s.Client.CreateChatCompletionStream(context.Background(), req)
	if err != nil {
		s.Messages = s.Messages[:len(s.Messages)-1] // 移除失败的用户消息
		return "", fmt.Errorf("ChatCompletionStream error: %v", err)
	}
	defer stream.Close()

	var fullResponse strings.Builder

	for {
		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("Stream error: %v", err)
		}

		if len(response.Choices) > 0 {
			chunk := response.Choices[0].Delta.Content
			fullResponse.WriteString(chunk)

			// 调用回调函数处理每个chunk
			if onChunk != nil {
				if err := onChunk(chunk); err != nil {
					return "", err
				}
			}
		}
	}

	aiResponse := fullResponse.String()

	// 添加AI回复到消息历史
	s.Messages = append(s.Messages, openai.ChatCompletionMessage{
		Role:    "assistant",
		Content: aiResponse,
	})

	return aiResponse, nil
}

// SetSystemPrompt 设置系统提示词（新增方法）
func (s *AdvancedChatSession) SetSystemPrompt(prompt string) {
	s.SystemPrompt = prompt

	// 重建消息列表，保留系统消息之后的消息
	if len(s.Messages) > 0 && s.Messages[0].Role == "system" {
		// 替换第一个系统消息
		s.Messages[0] = openai.ChatCompletionMessage{
			Role:    "system",
			Content: prompt,
		}
	} else {
		// 在开头插入系统消息
		newMessages := []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: prompt,
			},
		}
		s.Messages = append(newMessages, s.Messages...)
	}
}
