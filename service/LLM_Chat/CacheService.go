package LLM_Chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	// AppendStreamResponse 新增：流式响应缓存相关
	AppendStreamResponse(sessionID string, chunk string) error                               // 增量追加数据
	GetStreamResponse(sessionID string) (string, error)                                      // 获取完整响应
	DeleteStreamResponse(sessionID string) error                                             // 删除缓存
	SaveWithRetry(sessionID string, role, content string, userID uint, maxRetries int) error // 带重试的保存
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

// AppendStreamResponse 增量追加流式响应到 Redis
func (cs *CacheService) AppendStreamResponse(sessionID string, chunk string) error {
	if cs.redisClient == nil {
		return nil // 降级：Redis 不可用时不报错
	}

	ctx := context.Background()
	key := "stream_response:" + sessionID

	// 先检查 key 是否存在，不存在则设置过期时间
	exists, _ := cs.redisClient.Exists(ctx, key).Result()
	if err := cs.redisClient.Append(ctx, key, chunk).Err(); err != nil {
		return err
	}

	// 首次写入时设置 1 小时过期
	if exists == 0 {
		cs.redisClient.Expire(ctx, key, 10*time.Minute)
	}

	return cs.redisClient.Append(ctx, key, chunk).Err()
}

// GetStreamResponse 获取完整的流式响应
func (cs *CacheService) GetStreamResponse(sessionID string) (string, error) {
	if cs.redisClient == nil {
		return "", errors.New("redis不可用")
	}

	ctx := context.Background()
	key := "stream_response:" + sessionID
	result, err := cs.redisClient.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil // 缓存不存在，返回空字符串
	}
	return result, err
}

// DeleteStreamResponse 删除流式响应缓存
func (cs *CacheService) DeleteStreamResponse(sessionID string) error {
	if cs.redisClient == nil {
		return nil
	}

	ctx := context.Background()
	return cs.redisClient.Del(ctx, "stream_response:"+sessionID).Err()
}

// SaveWithRetry 带重试的消息保存
func (cs *CacheService) SaveWithRetry(sessionID string, role, content string, userID uint, maxRetries int) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		err := GlobalSessionManager.SaveMessage(sessionID, role, content, userID)
		if err == nil {
			return nil // 成功则退出
		}
		lastErr = err
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond) // 指数退避
		}
	}
	return fmt.Errorf("保存消息失败，重试 %d 次后仍失败: %v", maxRetries, lastErr)
}
