package LLM_Chat

import (
	"sync"
)

// SessionManager 会话管理器，用于管理多个聊天会话
type SessionManager struct {
	sessions       map[string]LLMSessionInterface
	mu             sync.RWMutex
	chatService    ChatServiceInterface
	cacheService   CacheServiceInterface
	modelService   UserAPIServiceInterface
	personaManager PersonaManagerInterface
	sessionCreator SessionCreatorInterface
}

var GlobalSessionManager *SessionManager
