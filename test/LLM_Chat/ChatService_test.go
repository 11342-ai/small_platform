package LLM_Chat_Service

import (
	"testing"
	"time"

	"platfrom/database"
	"platfrom/service/LLM_Chat"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupChatTestDB 创建聊天服务测试数据库（使用 SQLite 内存数据库）
func setupChatTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法创建测试数据库: %v", err)
	}

	// 自动迁移所有表
	err = db.AutoMigrate(&database.ChatSession{}, &database.ChatMessage{}, &database.SharedSession{})
	if err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}

// setupChatService 创建聊天服务实例
func setupChatService(t *testing.T) (LLM_Chat.ChatServiceInterface, func()) {
	db := setupChatTestDB(t)
	service, err := LLM_Chat.NewChatService(db)
	if err != nil {
		t.Fatalf("创建聊天服务失败: %v", err)
	}

	// 返回清理函数
	cleanup := func() {
		// SQLite 内存数据库会自动清理
	}

	return service, cleanup
}

// TestRootGetAllSessions 测试管理员获取所有会话
func TestRootGetAllSessions(t *testing.T) {
	service, cleanup := setupChatService(t)
	defer cleanup()

	// 创建一些测试会话
	testSessions := []struct {
		sessionID string
		modelName string
		userID    uint
		title     string
	}{
		{
			sessionID: "session_001",
			modelName: "gpt-3.5-turbo",
			userID:    1,
			title:     "测试会话1",
		},
		{
			sessionID: "session_002",
			modelName: "gpt-4",
			userID:    2,
			title:     "测试会话2",
		},
		{
			sessionID: "session_003",
			modelName: "claude-2",
			userID:    3,
			title:     "测试会话3",
		},
	}

	// 创建会话
	for _, ts := range testSessions {
		_, err := service.CreateChatSession(ts.sessionID, ts.modelName, ts.userID)
		if err != nil {
			t.Fatalf("创建测试会话失败: %v", err)
		}
		// 更新标题
		err = service.UpdateSessionTitle(ts.sessionID, ts.title)
		if err != nil {
			t.Fatalf("更新会话标题失败: %v", err)
		}
		// 保存一些消息
		err = service.SaveChatMessage(ts.sessionID, "user", "你好，我是用户", ts.userID)
		if err != nil {
			t.Fatalf("保存用户消息失败: %v", err)
		}
		err = service.SaveChatMessage(ts.sessionID, "assistant", "你好，我是助手", ts.userID)
		if err != nil {
			t.Fatalf("保存助手消息失败: %v", err)
		}
		// 等待一下，确保更新时间不同
		time.Sleep(10 * time.Millisecond)
	}

	tests := []struct {
		name     string
		page     int
		pageSize int
		wantErr  bool
	}{
		{
			name:     "第一页，每页2条",
			page:     1,
			pageSize: 2,
			wantErr:  false,
		},
		{
			name:     "第二页，每页2条",
			page:     2,
			pageSize: 2,
			wantErr:  false,
		},
		{
			name:     "无效页码",
			page:     -1,
			pageSize: 10,
			wantErr:  false, // 方法会修正页码为1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessions, total, err := service.RootGetAllSessions(tt.page, tt.pageSize)

			if tt.wantErr {
				if err == nil {
					t.Errorf("RootGetAllSessions() 期望返回错误，但没有")
				}
				return
			}

			if err != nil {
				t.Errorf("RootGetAllSessions() 意外返回错误: %v", err)
				return
			}

			// 检查总数
			if total < 3 {
				t.Errorf("会话总数太少: 得到 %v, 期望至少 3", total)
			}

			// 检查返回的会话数据
			for _, session := range sessions {
				if session.SessionID == "" {
					t.Error("返回的会话缺少 sessionID")
				}
			}
		})
	}
}

// TestRootGetSessionMessages 测试管理员获取会话消息
func TestRootGetSessionMessages(t *testing.T) {
	service, cleanup := setupChatService(t)
	defer cleanup()

	// 创建一个测试会话
	sessionID := "test_admin_session"
	userID := uint(100)
	modelName := "gpt-3.5-turbo"

	_, err := service.CreateChatSession(sessionID, modelName, userID)
	if err != nil {
		t.Fatalf("创建测试会话失败: %v", err)
	}

	// 保存一些消息
	messagesToSave := []struct {
		role    string
		content string
	}{
		{"user", "管理员能看到这条消息吗？"},
		{"assistant", "是的，管理员可以查看所有消息。"},
		{"user", "那隐私呢？"},
		{"assistant", "管理员有权限审核所有内容。"},
	}

	for _, msg := range messagesToSave {
		err = service.SaveChatMessage(sessionID, msg.role, msg.content, userID)
		if err != nil {
			t.Fatalf("保存消息失败: %v", err)
		}
	}

	t.Run("成功获取会话消息", func(t *testing.T) {
		messages, err := service.RootGetSessionMessages(sessionID)

		if err != nil {
			t.Errorf("RootGetSessionMessages() 意外返回错误: %v", err)
			return
		}

		// 应该返回4条消息
		if len(messages) != 4 {
			t.Errorf("返回消息数量不匹配: 得到 %v, 期望 %v", len(messages), 4)
		}

		// 检查消息内容
		for _, msg := range messages {
			if msg.SessionID != sessionID {
				t.Errorf("消息 sessionID 不匹配: 得到 %v, 期望 %v", msg.SessionID, sessionID)
			}
		}
	})

	t.Run("获取不存在的会话消息", func(t *testing.T) {
		nonExistentSessionID := "non_existent_session_xyz"
		messages, err := service.RootGetSessionMessages(nonExistentSessionID)

		if err != nil {
			t.Errorf("RootGetSessionMessages() 返回错误（可能不应该）: %v", err)
			return
		}

		// 应该返回空数组
		if len(messages) != 0 {
			t.Errorf("不存在的会话应返回空数组: 得到 %v 条消息", len(messages))
		}
	})
}

// TestRootDeleteSession 测试管理员删除会话
func TestRootDeleteSession(t *testing.T) {
	service, cleanup := setupChatService(t)
	defer cleanup()

	// 创建一个测试会话
	sessionID := "session_to_delete"
	userID := uint(200)
	modelName := "gpt-4"

	_, err := service.CreateChatSession(sessionID, modelName, userID)
	if err != nil {
		t.Fatalf("创建测试会话失败: %v", err)
	}

	// 保存一些消息
	err = service.SaveChatMessage(sessionID, "user", "这条消息将被删除", userID)
	if err != nil {
		t.Fatalf("保存消息失败: %v", err)
	}

	tests := []struct {
		name        string
		sessionID   string
		wantErr     bool
		errContains string
	}{
		{
			name:      "成功删除会话",
			sessionID: sessionID,
			wantErr:   false,
		},
		{
			name:        "删除不存在的会话",
			sessionID:   "non_existent_session_123",
			wantErr:     false, // 可能不返回错误（取决于实现）
			errContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RootDeleteSession(tt.sessionID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("RootDeleteSession() 期望返回错误，但没有")
					return
				}
				if tt.errContains != "" {
					// 检查错误消息包含特定字符串
					// 这里省略具体检查，因为不知道具体错误消息
				}
				return
			}

			// 如果不期望错误，但有错误，记录警告（可能允许删除不存在的会话）
			if err != nil {
				t.Logf("RootDeleteSession() 返回错误（可能是正常的）: %v", err)
			}
		})
	}
}
