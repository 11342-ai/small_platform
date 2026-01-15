package LLM_Chat_Service

import (
	"testing"

	"platfrom/database"
	"platfrom/service/LLM_Chat"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建测试数据库（使用 SQLite 内存数据库）
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法创建测试数据库: %v", err)
	}

	// 自动迁移所有表
	err = db.AutoMigrate(&database.UserAPI{})
	if err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}

// setupUserAPIService 创建用户API服务实例
func setupUserAPIService(t *testing.T) (LLM_Chat.UserAPIServiceInterface, func()) {
	db := setupTestDB(t)
	service, err := LLM_Chat.NewUserAPIService(db)
	if err != nil {
		t.Fatalf("创建用户API服务失败: %v", err)
	}

	// 返回清理函数
	cleanup := func() {
		// SQLite 内存数据库会自动清理
	}

	return service, cleanup
}

// TestCreateAPI 测试创建API配置
func TestCreateAPI(t *testing.T) {
	service, cleanup := setupUserAPIService(t)
	defer cleanup()

	tests := []struct {
		name        string
		userID      uint
		api         *database.UserAPI
		wantErr     bool
		errContains string
	}{
		{
			name:   "成功创建API配置",
			userID: 1,
			api: &database.UserAPI{
				APIName:   "openai-api",
				APIKey:    "sk-test-key-123",
				ModelName: "gpt-4",
				BaseURL:   "https://api.openai.com/v1",
			},
			wantErr: false,
		},
		{
			name:   "API名称已存在",
			userID: 1,
			api: &database.UserAPI{
				APIName:   "openai-api", // 与上面重复
				APIKey:    "sk-another-key",
				ModelName: "gpt-3.5-turbo",
			},
			wantErr:     true,
			errContains: "API名称已存在",
		},
		{
			name:   "用户ID为空",
			userID: 0,
			api: &database.UserAPI{
				APIName: "test-api",
				APIKey:  "sk-test-key",
			},
			wantErr:     true,
			errContains: "用户ID不能为空",
		},
		{
			name:   "API名称为空",
			userID: 1,
			api: &database.UserAPI{
				APIName: "",
				APIKey:  "sk-test-key",
			},
			wantErr:     true,
			errContains: "API名称不能为空",
		},
		{
			name:   "API密钥为空",
			userID: 1,
			api: &database.UserAPI{
				APIName: "test-api",
				APIKey:  "",
			},
			wantErr:     true,
			errContains: "API密钥不能为空",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api, err := service.CreateAPI(tt.userID, tt.api)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateAPI() 期望返回错误，但没有")
					return
				}
				if tt.errContains != "" &&
					!contains(err.Error(), tt.errContains) {
					t.Errorf("错误消息应包含 '%s', 实际: %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("CreateAPI() 意外返回错误: %v", err)
				return
			}

			if api == nil {
				t.Error("CreateAPI() 返回的API配置为 nil")
				return
			}

			if api.UserID != tt.userID {
				t.Errorf("用户ID不匹配: 得到 %v, 期望 %v", api.UserID, tt.userID)
			}

			if api.APIName != tt.api.APIName {
				t.Errorf("API名称不匹配: 得到 %v, 期望 %v", api.APIName, tt.api.APIName)
			}

			if api.ModelName != tt.api.ModelName {
				t.Errorf("模型名称不匹配: 得到 %v, 期望 %v", api.ModelName, tt.api.ModelName)
			}
		})
	}
}

// TestGetAPIByID 测试根据ID获取API配置
func TestGetAPIByID(t *testing.T) {
	service, cleanup := setupUserAPIService(t)
	defer cleanup()

	// 先创建一个测试API配置
	testAPI := &database.UserAPI{
		APIName:   "test-api",
		APIKey:    "sk-test-key-123",
		ModelName: "gpt-4",
	}
	createdAPI, err := service.CreateAPI(1, testAPI)
	if err != nil {
		t.Fatalf("创建测试API配置失败: %v", err)
	}

	tests := []struct {
		name    string
		apiID   uint
		wantErr bool
	}{
		{
			name:    "成功获取API配置",
			apiID:   createdAPI.ID,
			wantErr: false,
		},
		{
			name:    "API配置不存在",
			apiID:   9999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api, err := service.GetAPIByID(tt.apiID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetAPIByID() 期望返回错误，但没有")
				}
				return
			}

			if err != nil {
				t.Errorf("GetAPIByID() 意外返回错误: %v", err)
				return
			}

			if api.ID != createdAPI.ID {
				t.Errorf("API ID不匹配: 得到 %v, 期望 %v", api.ID, createdAPI.ID)
			}

			if api.APIName != createdAPI.APIName {
				t.Errorf("API名称不匹配: 得到 %v, 期望 %v", api.APIName, createdAPI.APIName)
			}
		})
	}
}

// TestGetAPIByName 测试根据名称获取用户的API配置
func TestGetAPIByName(t *testing.T) {
	service, cleanup := setupUserAPIService(t)
	defer cleanup()

	// 先创建一个测试API配置
	testAPI := &database.UserAPI{
		APIName:   "openai-api",
		APIKey:    "sk-test-key-123",
		ModelName: "gpt-4",
	}
	createdAPI, err := service.CreateAPI(1, testAPI)
	if err != nil {
		t.Fatalf("创建测试API配置失败: %v", err)
	}

	tests := []struct {
		name    string
		userID  uint
		apiName string
		wantErr bool
	}{
		{
			name:    "成功获取API配置",
			userID:  1,
			apiName: "openai-api",
			wantErr: false,
		},
		{
			name:    "API配置不存在",
			userID:  1,
			apiName: "nonexistent-api",
			wantErr: true,
		},
		{
			name:    "用户不匹配",
			userID:  999,
			apiName: "openai-api",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api, err := service.GetAPIByName(tt.userID, tt.apiName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetAPIByName() 期望返回错误，但没有")
				}
				return
			}

			if err != nil {
				t.Errorf("GetAPIByName() 意外返回错误: %v", err)
				return
			}

			if api.ID != createdAPI.ID {
				t.Errorf("API ID不匹配: 得到 %v, 期望 %v", api.ID, createdAPI.ID)
			}
		})
	}
}

// TestGetAPIByModelName 测试根据模型名称获取用户的API配置
func TestGetAPIByModelName(t *testing.T) {
	service, cleanup := setupUserAPIService(t)
	defer cleanup()

	// 先创建一个测试API配置
	testAPI := &database.UserAPI{
		APIName:   "openai-api",
		APIKey:    "sk-test-key-123",
		ModelName: "gpt-4",
	}
	createdAPI, err := service.CreateAPI(1, testAPI)
	if err != nil {
		t.Fatalf("创建测试API配置失败: %v", err)
	}

	tests := []struct {
		name      string
		userID    uint
		modelName string
		wantErr   bool
	}{
		{
			name:      "成功获取API配置",
			userID:    1,
			modelName: "gpt-4",
			wantErr:   false,
		},
		{
			name:      "API配置不存在",
			userID:    1,
			modelName: "gpt-5",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api, err := service.GetAPIByModelName(tt.userID, tt.modelName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetAPIByModelName() 期望返回错误，但没有")
				}
				return
			}

			if err != nil {
				t.Errorf("GetAPIByModelName() 意外返回错误: %v", err)
				return
			}

			if api.ID != createdAPI.ID {
				t.Errorf("API ID不匹配: 得到 %v, 期望 %v", api.ID, createdAPI.ID)
			}
		})
	}
}

// TestGetUserAPIs 测试获取用户的所有API配置
func TestGetUserAPIs(t *testing.T) {
	service, cleanup := setupUserAPIService(t)
	defer cleanup()

	// 为用户1创建多个API配置
	api1 := &database.UserAPI{
		APIName:   "openai-api",
		APIKey:    "sk-key-1",
		ModelName: "gpt-4",
	}
	api2 := &database.UserAPI{
		APIName:   "anthropic-api",
		APIKey:    "sk-key-2",
		ModelName: "claude-3",
	}
	api3 := &database.UserAPI{
		APIName:   "google-api",
		APIKey:    "sk-key-3",
		ModelName: "gemini-pro",
	}

	_, err := service.CreateAPI(1, api1)
	if err != nil {
		t.Fatalf("创建API配置1失败: %v", err)
	}
	_, err = service.CreateAPI(1, api2)
	if err != nil {
		t.Fatalf("创建API配置2失败: %v", err)
	}
	_, err = service.CreateAPI(2, api3) // 另一个用户
	if err != nil {
		t.Fatalf("创建API配置3失败: %v", err)
	}

	t.Run("成功获取用户1的所有API配置", func(t *testing.T) {
		apis, err := service.GetUserAPIs(1)

		if err != nil {
			t.Errorf("GetUserAPIs() 意外返回错误: %v", err)
			return
		}

		if len(apis) != 2 {
			t.Errorf("期望获取2个API配置, 实际得到 %d 个", len(apis))
		}
	})

	t.Run("成功获取用户2的所有API配置", func(t *testing.T) {
		apis, err := service.GetUserAPIs(2)

		if err != nil {
			t.Errorf("GetUserAPIs() 意外返回错误: %v", err)
			return
		}

		if len(apis) != 1 {
			t.Errorf("期望获取1个API配置, 实际得到 %d 个", len(apis))
		}
	})

	t.Run("获取不存在用户的API配置", func(t *testing.T) {
		apis, err := service.GetUserAPIs(999)

		if err != nil {
			t.Errorf("GetUserAPIs() 意外返回错误: %v", err)
		}

		if len(apis) != 0 {
			t.Errorf("期望获取0个API配置, 实际得到 %d 个", len(apis))
		}
	})
}

// TestUpdateAPI 测试更新API配置
func TestUpdateAPI(t *testing.T) {
	service, cleanup := setupUserAPIService(t)
	defer cleanup()

	// 先创建一个测试API配置
	testAPI := &database.UserAPI{
		APIName:   "openai-api",
		APIKey:    "sk-old-key",
		ModelName: "gpt-3.5-turbo",
	}
	createdAPI, err := service.CreateAPI(1, testAPI)
	if err != nil {
		t.Fatalf("创建测试API配置失败: %v", err)
	}

	// 创建另一个API配置用于测试重名
	anotherAPI := &database.UserAPI{
		APIName:   "anthropic-api",
		APIKey:    "sk-another-key",
		ModelName: "claude-3",
	}
	_, err = service.CreateAPI(1, anotherAPI)
	if err != nil {
		t.Fatalf("创建另一个API配置失败: %v", err)
	}

	tests := []struct {
		name        string
		apiID       uint
		updates     map[string]interface{}
		wantErr     bool
		errContains string
	}{
		{
			name:  "成功更新API配置",
			apiID: createdAPI.ID,
			updates: map[string]interface{}{
				"api_key":    "sk-new-key",
				"model_name": "gpt-4",
			},
			wantErr: false,
		},
		{
			name:        "更新内容为空",
			apiID:       createdAPI.ID,
			updates:     map[string]interface{}{},
			wantErr:     true,
			errContains: "更新内容不能为空",
		},
		{
			name:  "API配置不存在",
			apiID: 9999,
			updates: map[string]interface{}{
				"api_key": "sk-new-key",
			},
			wantErr:     true,
			errContains: "API配置不存在",
		},
		{
			name:  "API名称与其他API重复",
			apiID: createdAPI.ID,
			updates: map[string]interface{}{
				"api_name": "anthropic-api", // 与另一个API重名
			},
			wantErr:     true,
			errContains: "API名称已存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.UpdateAPI(tt.apiID, tt.updates)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateAPI() 期望返回错误，但没有")
					return
				}
				if tt.errContains != "" &&
					!contains(err.Error(), tt.errContains) {
					t.Errorf("错误消息应包含 '%s', 实际: %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("UpdateAPI() 意外返回错误: %v", err)
				return
			}

			// 验证更新是否成功
			updatedAPI, err := service.GetAPIByID(tt.apiID)
			if err != nil {
				t.Errorf("获取更新后的API配置失败: %v", err)
				return
			}

			if newKey, ok := tt.updates["api_key"].(string); ok {
				if updatedAPI.APIKey != newKey {
					t.Errorf("API密钥未更新: 得到 %v, 期望 %v", updatedAPI.APIKey, newKey)
				}
			}

			if newModel, ok := tt.updates["model_name"].(string); ok {
				if updatedAPI.ModelName != newModel {
					t.Errorf("模型名称未更新: 得到 %v, 期望 %v", updatedAPI.ModelName, newModel)
				}
			}
		})
	}
}

// TestDeleteAPI 测试删除API配置
func TestDeleteAPI(t *testing.T) {
	service, cleanup := setupUserAPIService(t)
	defer cleanup()

	// 先创建一个测试API配置
	testAPI := &database.UserAPI{
		APIName:   "test-api",
		APIKey:    "sk-test-key-123",
		ModelName: "gpt-4",
	}
	createdAPI, err := service.CreateAPI(1, testAPI)
	if err != nil {
		t.Fatalf("创建测试API配置失败: %v", err)
	}

	tests := []struct {
		name        string
		apiID       uint
		wantErr     bool
		errContains string
	}{
		{
			name:    "成功删除API配置",
			apiID:   createdAPI.ID,
			wantErr: false,
		},
		{
			name:        "API配置不存在",
			apiID:       9999,
			wantErr:     true,
			errContains: "API配置不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DeleteAPI(tt.apiID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteAPI() 期望返回错误，但没有")
					return
				}
				if tt.errContains != "" &&
					!contains(err.Error(), tt.errContains) {
					t.Errorf("错误消息应包含 '%s', 实际: %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("DeleteAPI() 意外返回错误: %v", err)
				return
			}

			// 验证删除是否成功
			_, err = service.GetAPIByID(tt.apiID)
			if err == nil {
				t.Error("API配置应该已被删除，但仍然可以获取到")
			}
		})
	}
}

// TestGetFirstAvailableAPI 测试获取用户第一个可用的API配置
func TestGetFirstAvailableAPI(t *testing.T) {
	service, cleanup := setupUserAPIService(t)
	defer cleanup()

	// 先创建一个测试API配置
	testAPI := &database.UserAPI{
		APIName:   "openai-api",
		APIKey:    "sk-test-key-123",
		ModelName: "gpt-4",
	}
	createdAPI, err := service.CreateAPI(1, testAPI)
	if err != nil {
		t.Fatalf("创建测试API配置失败: %v", err)
	}

	tests := []struct {
		name    string
		userID  uint
		wantErr bool
	}{
		{
			name:    "成功获取第一个可用API配置",
			userID:  1,
			wantErr: false,
		},
		{
			name:    "用户没有配置任何API",
			userID:  999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api, err := service.GetFirstAvailableAPI(tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetFirstAvailableAPI() 期望返回错误，但没有")
				}
				return
			}

			if err != nil {
				t.Errorf("GetFirstAvailableAPI() 意外返回错误: %v", err)
				return
			}

			if api.ID != createdAPI.ID {
				t.Errorf("API ID不匹配: 得到 %v, 期望 %v", api.ID, createdAPI.ID)
			}
		})
	}
}

// TestAPIConnection 测试API连接
func TestAPIConnection(t *testing.T) {
	service, cleanup := setupUserAPIService(t)
	defer cleanup()

	t.Run("测试API连接（简单实现）", func(t *testing.T) {
		success, err := service.TestAPIConnection()

		if err != nil {
			t.Errorf("TestAPIConnection() 意外返回错误: %v", err)
			return
		}

		if !success {
			t.Error("TestAPIConnection() 应该返回 true")
		}
	})
}

// 辅助函数：检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(substr) == 0 ||
			len(s) > 0 && (s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
