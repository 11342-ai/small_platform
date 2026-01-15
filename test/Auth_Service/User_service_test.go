package Auth_Service

import (
	"testing"
	"time"

	"platfrom/service/Auth"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"platfrom/database"
)

// setupTestDB 创建测试数据库（使用 SQLite 内存数据库）
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法创建测试数据库: %v", err)
	}

	// 自动迁移所有表
	err = db.AutoMigrate(&database.User{}, &database.VerificationCode{})
	if err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}

// setupUserService 创建用户服务实例
func setupUserService(t *testing.T) (Auth.UserService, func()) {
	db := setupTestDB(t)
	service, err := Auth.NewUserService(db)
	if err != nil {
		t.Fatalf("创建用户服务失败: %v", err)
	}

	// 返回清理函数
	cleanup := func() {
		// SQLite 内存数据库会自动清理
	}

	return service, cleanup
}

// TestCreateUser 测试创建用户
func TestCreateUser(t *testing.T) {
	service, cleanup := setupUserService(t)
	defer cleanup()

	tests := []struct {
		name        string
		request     database.RegisterRequest
		wantErr     bool
		errContains string
	}{
		{
			name: "成功创建用户",
			request: database.RegisterRequest{
				Username: "testuser_all_all",
				Password: "password123",
				Email:    "test@example.com",
			},
			wantErr: false,
		},
		{
			name: "用户名已存在",
			request: database.RegisterRequest{
				Username: "testuser_all", // 与上面重复
				Password: "password456",
				Email:    "test2@example.com",
			},
			wantErr:     true,
			errContains: "用户名已存在",
		},
		{
			name: "空用户名",
			request: database.RegisterRequest{
				Username: "",
				Password: "password123",
			},
			wantErr: true, // GORM 验证会失败
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.CreateUser(tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateUser() 期望返回错误，但没有")
					return
				}
				if tt.errContains != "" &&
					!contains(err.Error(), tt.errContains) {
					t.Errorf("错误消息应包含 '%s', 实际: %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("CreateUser() 意外返回错误: %v", err)
				return
			}

			if user == nil {
				t.Error("CreateUser() 返回的用户为 nil")
				return
			}

			if user.Username != tt.request.Username {
				t.Errorf("用户名不匹配: 得到 %v, 期望 %v", user.Username, tt.request.Username)
			}

			if user.Email != tt.request.Email {
				t.Errorf("邮箱不匹配: 得到 %v, 期望 %v", user.Email, tt.request.Email)
			}
		})
	}
}

// TestGetUserByUsername 测试根据用户名获取用户
func TestGetUserByUsername(t *testing.T) {
	service, cleanup := setupUserService(t)
	defer cleanup()

	// 先创建一个测试用户
	testUser := database.RegisterRequest{
		Username: "testuser_all",
		Password: "password123",
		Email:    "test@example.com",
	}
	createdUser, err := service.CreateUser(testUser)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{
			name:     "成功获取用户",
			username: "testuser_all",
			wantErr:  false,
		},
		{
			name:     "用户不存在",
			username: "nonexistent",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.GetUserByUsername(tt.username)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetUserByUsername() 期望返回错误，但没有")
				}
				return
			}

			if err != nil {
				t.Errorf("GetUserByUsername() 意外返回错误: %v", err)
				return
			}

			if user.ID != createdUser.ID {
				t.Errorf("用户ID不匹配: 得到 %v, 期望 %v", user.ID, createdUser.ID)
			}
		})
	}
}

// TestUpdatePassword 测试修改密码
func TestUpdatePassword(t *testing.T) {
	service, cleanup := setupUserService(t)
	defer cleanup()

	// 创建测试用户
	testUser := database.RegisterRequest{
		Username: "testuser_all",
		Password: "oldpassword",
	}
	createdUser, err := service.CreateUser(testUser)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	tests := []struct {
		name        string
		userID      uint
		oldPassword string
		newPassword string
		wantErr     bool
		errContains string
	}{
		{
			name:        "成功修改密码",
			userID:      createdUser.ID,
			oldPassword: "oldpassword",
			newPassword: "newpassword",
			wantErr:     false,
		},
		{
			name:        "旧密码错误",
			userID:      createdUser.ID,
			oldPassword: "wrongpassword",
			newPassword: "newpassword",
			wantErr:     true,
			errContains: "旧密码不正确",
		},
		{
			name:        "用户不存在",
			userID:      999,
			oldPassword: "oldpassword",
			newPassword: "newpassword",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.UpdatePassword(tt.userID, tt.oldPassword, tt.newPassword)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdatePassword() 期望返回错误，但没有")
					return
				}
				if tt.errContains != "" &&
					!contains(err.Error(), tt.errContains) {
					t.Errorf("错误消息应包含 '%s', 实际: %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("UpdatePassword() 意外返回错误: %v", err)
			}
		})
	}
}

// TestSendVerificationCode 测试发送验证码
func TestSendVerificationCode(t *testing.T) {
	service, cleanup := setupUserService(t)
	defer cleanup()

	// 创建测试用户
	testUser := database.RegisterRequest{
		Username: "testuser_all",
		Password: "password123",
	}
	_, err := service.CreateUser(testUser)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	t.Run("成功发送验证码", func(t *testing.T) {
		code, err := service.SendVerificationCode("testuser_all", "password_reset")
		if err != nil {
			t.Errorf("SendVerificationCode() 意外返回错误: %v", err)
			return
		}

		if code.Code == "" {
			t.Error("验证码不应为空")
		}

		if time.Now().After(code.ExpiresAt) {
			t.Error("验证码过期时间不正确")
		}
	})
}

// TestVerifyCode 测试验证验证码
func TestVerifyCode(t *testing.T) {
	service, cleanup := setupUserService(t)
	defer cleanup()

	// 创建测试用户
	testUser := database.RegisterRequest{
		Username: "testuser_all",
		Password: "password123",
	}
	_, err := service.CreateUser(testUser)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 发送验证码
	codeRecord, err := service.SendVerificationCode("testuser_all", "password_reset")
	if err != nil {
		t.Fatalf("发送验证码失败: %v", err)
	}

	tests := []struct {
		name      string
		username  string
		code      string
		codeType  string
		wantValid bool
		wantErr   bool
	}{
		{
			name:      "验证码正确",
			username:  "testuser_all",
			code:      codeRecord.Code,
			codeType:  "password_reset",
			wantValid: true,
			wantErr:   false,
		},
		{
			name:      "验证码错误",
			username:  "testuser_all",
			code:      "000000",
			codeType:  "password_reset",
			wantValid: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := service.VerifyCode(tt.username, tt.code, tt.codeType)

			if tt.wantErr {
				if err == nil {
					t.Errorf("VerifyCode() 期望返回错误，但没有")
				}
				return
			}

			if err != nil {
				t.Errorf("VerifyCode() 意外返回错误: %v", err)
				return
			}

			if valid != tt.wantValid {
				t.Errorf("验证结果不匹配: 得到 %v, 期望 %v", valid, tt.wantValid)
			}
		})
	}
}

// TestResetPassword 测试重置密码
func TestResetPassword(t *testing.T) {
	service, cleanup := setupUserService(t)
	defer cleanup()

	// 创建测试用户
	testUser := database.RegisterRequest{
		Username: "testuser_all",
		Password: "oldpassword",
	}
	_, err := service.CreateUser(testUser)
	if err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}

	// 发送验证码
	codeRecord, err := service.SendVerificationCode("testuser_all", "password_reset")
	if err != nil {
		t.Fatalf("发送验证码失败: %v", err)
	}

	t.Run("成功重置密码", func(t *testing.T) {
		err := service.ResetPassword("testuser_all", codeRecord.Code, "newpassword")
		if err != nil {
			t.Errorf("ResetPassword() 意外返回错误: %v", err)
		}

		// 验证新密码
		user, err := service.GetUserByUsername("testuser_all")
		if err != nil {
			t.Errorf("获取用户失败: %v", err)
		}

		valid := Auth.VerifyPassword("newpassword", user.PasswordHash)
		if !valid {
			t.Error("新密码验证失败")
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
