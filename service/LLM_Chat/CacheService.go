package LLM_Chat

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-redis/redis/v8"
	"github.com/sashabaranov/go-openai"
	"log"
	"platfrom/database"
	"time"
)

type CacheService struct {
	redisClient *redis.Client
	available   bool
}

type CachedSession struct {
	Session     *database.ChatSession          `json:"session"`
	Messages    []openai.ChatCompletionMessage `json:"messages"`
	ModelAPIKey string                         `json:"model_api_key"`
	BaseUrl     string                         `json:"baseUrl"`
}

// CacheServiceInterface 缓存服务接口
type CacheServiceInterface interface {
	CacheChatSession(sessionID string, session *database.ChatSession, expiration time.Duration) error
	GetCachedChatSession(sessionID string) (*database.ChatSession, error)
	CacheModelConfig(modelName string, model *database.UserAPI) error
	CacheFullSession(sessionID string, cachedSession *CachedSession, expiration time.Duration) error
	GetCachedFullSession(sessionID string) (*CachedSession, error)
}

var GlobalCacheService CacheServiceInterface

func NewCacheService(client *redis.Client, available bool) CacheServiceInterface {
	service := &CacheService{
		redisClient: client,
		available:   available,
	}
	GlobalCacheService = service
	return service
}

// CacheChatSession 缓存聊天会话
func (cs *CacheService) CacheChatSession(sessionID string, session *database.ChatSession, expiration time.Duration) error {
	if cs.redisClient == nil {
		log.Printf("Redis不可用，跳过缓存会话: %s", sessionID)
		return nil // 降级：直接返回成功
	}

	ctx := context.Background()
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return cs.redisClient.Set(ctx, "session:"+sessionID, data, expiration).Err()
}

// GetCachedChatSession 从缓存获取会话
func (cs *CacheService) GetCachedChatSession(sessionID string) (*database.ChatSession, error) {
	if cs.redisClient == nil {
		return nil, errors.New("redis不可用")
	}

	ctx := context.Background()
	data, err := cs.redisClient.Get(ctx, "session:"+sessionID).Result()
	if err != nil {
		return nil, err
	}

	var session database.ChatSession
	err = json.Unmarshal([]byte(data), &session)
	return &session, err
}

// CacheModelConfig 缓存模型配置
func (cs *CacheService) CacheModelConfig(modelName string, model *database.UserAPI) error {
	if cs.redisClient == nil {
		log.Printf("Redis不可用，跳过缓存模型配置: %s", modelName)
		return nil // 降级：直接返回成功
	}

	ctx := context.Background()
	data, err := json.Marshal(model)
	if err != nil {
		return err
	}
	return cs.redisClient.Set(ctx, "model:"+modelName, data, 24*time.Hour).Err()
}

// CacheFullSession 缓存完整会话状态
func (cs *CacheService) CacheFullSession(sessionID string, cachedSession *CachedSession, expiration time.Duration) error {
	if cs.redisClient == nil {
		log.Printf("Redis不可用，跳过缓存完整会话: %s", sessionID)
		return nil // 降级：直接返回成功
	}

	ctx := context.Background()
	data, err := json.Marshal(cachedSession)
	if err != nil {
		return err
	}
	return cs.redisClient.Set(ctx, "full_session:"+sessionID, data, expiration).Err()
}

// GetCachedFullSession 获取完整会话状态
func (cs *CacheService) GetCachedFullSession(sessionID string) (*CachedSession, error) {
	if cs.redisClient == nil {
		return nil, errors.New("redis不可用")
	}

	ctx := context.Background()
	data, err := cs.redisClient.Get(ctx, "full_session:"+sessionID).Result()
	if err != nil {
		return nil, err
	}

	var cachedSession CachedSession
	err = json.Unmarshal([]byte(data), &cachedSession)
	return &cachedSession, err
}
