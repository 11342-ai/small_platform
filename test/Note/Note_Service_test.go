package Note

import (
	"strings"
	"testing"
	"time"

	"platfrom/database"
	"platfrom/service/Note"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupNoteTestDB 创建笔记服务测试数据库（使用 SQLite 内存数据库）
func setupNoteTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法创建测试数据库: %v", err)
	}

	// 自动迁移笔记表
	err = db.AutoMigrate(&database.Note{})
	if err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}

// setupNoteService 创建笔记服务实例
func setupNoteService(t *testing.T) (Note.NoteServiceInterface, func()) {
	db := setupNoteTestDB(t)

	// 保存原始的全局 database.DB
	originalDB := database.DB

	// 替换全局 database.DB 为测试数据库
	database.DB = db

	// 创建新的服务实例（会使用全局的 database.DB）
	service := Note.NewNoteService()

	// 返回清理函数，恢复原始数据库连接
	cleanup := func() {
		database.DB = originalDB
		// SQLite 内存数据库会自动清理
	}

	return service, cleanup
}

// createTestNotes 创建测试笔记数据
func createTestNotes(t *testing.T, db *gorm.DB) []database.Note {
	testNotes := []database.Note{
		{
			UserID:   1,
			Title:    "用户1的测试笔记1",
			Content:  "这是用户1的第一个笔记内容",
			Tags:     nil, // 暂时不设置标签，避免SQLite序列化问题
			Category: "工作",
			IsPublic: true,
		},
		{
			UserID:   1,
			Title:    "用户1的测试笔记2",
			Content:  "这是用户1的第二个笔记内容",
			Tags:     nil,
			Category: "生活",
			IsPublic: false,
		},
		{
			UserID:   2,
			Title:    "用户2的测试笔记1",
			Content:  "这是用户2的第一个笔记内容",
			Tags:     nil,
			Category: "学习",
			IsPublic: true,
		},
		{
			UserID:   2,
			Title:    "用户2的测试笔记2",
			Content:  "这是用户2的第二个笔记内容",
			Tags:     nil,
			Category: "娱乐",
			IsPublic: true,
		},
		{
			UserID:   3,
			Title:    "用户3的测试笔记1",
			Content:  "这是用户3的第一个笔记内容",
			Tags:     nil,
			Category: "思考",
			IsPublic: false,
		},
	}

	// 创建笔记
	for i := range testNotes {
		if err := db.Create(&testNotes[i]).Error; err != nil {
			t.Fatalf("创建测试笔记失败: %v", err)
		}
		// 等待一下，确保更新时间不同
		time.Sleep(10 * time.Millisecond)
	}

	// 从数据库重新加载，获取完整的字段（如 ID、创建时间等）
	var notes []database.Note
	if err := db.Find(&notes).Error; err != nil {
		t.Fatalf("重新加载测试笔记失败: %v", err)
	}

	return notes
}

// TestRootGetAllNotes 测试管理员获取所有笔记
func TestRootGetAllNotes(t *testing.T) {
	service, cleanup := setupNoteService(t)
	defer cleanup()

	// 使用全局的 database.DB 创建测试笔记数据
	testNotes := createTestNotes(t, database.DB)

	tests := []struct {
		name     string
		userID   uint
		page     int
		pageSize int
		wantErr  bool
	}{
		{
			name:     "获取所有笔记 - 第一页，每页2条",
			userID:   0, // 0 表示获取所有用户的笔记
			page:     1,
			pageSize: 2,
			wantErr:  false,
		},
		{
			name:     "获取所有笔记 - 第二页，每页2条",
			userID:   0,
			page:     2,
			pageSize: 2,
			wantErr:  false,
		},
		{
			name:     "按用户ID筛选 - 用户1的笔记",
			userID:   1,
			page:     1,
			pageSize: 10,
			wantErr:  false,
		},
		{
			name:     "按用户ID筛选 - 用户2的笔记",
			userID:   2,
			page:     1,
			pageSize: 10,
			wantErr:  false,
		},
		{
			name:     "无效页码（自动修正为1）",
			userID:   0,
			page:     -1,
			pageSize: 10,
			wantErr:  false,
		},
		{
			name:     "过大页大小（自动限制为100）",
			userID:   0,
			page:     1,
			pageSize: 200,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes, total, err := service.RootGetAllNotes(tt.userID, tt.page, tt.pageSize)

			if tt.wantErr {
				if err == nil {
					t.Errorf("RootGetAllNotes() 期望返回错误，但没有")
				}
				return
			}

			if err != nil {
				t.Errorf("RootGetAllNotes() 意外返回错误: %v", err)
				return
			}

			// 检查总数
			if total < 0 {
				t.Errorf("笔记总数不能为负数: %v", total)
			}

			// 根据用户ID筛选期望的笔记数
			expectedCount := 0
			for _, note := range testNotes {
				if tt.userID == 0 || note.UserID == tt.userID {
					expectedCount++
				}
			}

			if int64(expectedCount) != total {
				t.Errorf("笔记总数不匹配: 得到 %v, 期望 %v", total, expectedCount)
			}

			// 检查返回的笔记数据
			for _, note := range notes {
				if note.Title == "" {
					t.Error("返回的笔记缺少标题")
				}
				if tt.userID > 0 && note.UserID != tt.userID {
					t.Errorf("返回的笔记用户ID不匹配: 得到 %v, 期望 %v", note.UserID, tt.userID)
				}
			}
		})
	}
}

// TestRootGetNoteByID 测试管理员获取笔记详情
func TestRootGetNoteByID(t *testing.T) {
	service, cleanup := setupNoteService(t)
	defer cleanup()

	// 使用全局的 database.DB 创建测试笔记数据
	testNotes := createTestNotes(t, database.DB)

	// 获取第一个笔记的ID
	var firstNoteID uint
	if len(testNotes) > 0 {
		firstNoteID = testNotes[0].ID
	} else {
		t.Fatal("没有创建测试笔记")
	}

	tests := []struct {
		name        string
		noteID      uint
		wantErr     bool
		errContains string
	}{
		{
			name:    "成功获取存在的笔记",
			noteID:  firstNoteID,
			wantErr: false,
		},
		{
			name:        "获取不存在的笔记",
			noteID:      99999,
			wantErr:     true,
			errContains: "笔记不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := service.RootGetNoteByID(tt.noteID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("RootGetNoteByID() 期望返回错误，但没有")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("错误消息不包含期望的字符串: 得到 %v, 期望包含 %v", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("RootGetNoteByID() 意外返回错误: %v", err)
				return
			}

			// 检查返回的笔记数据
			if note == nil {
				t.Error("返回的笔记为 nil")
				return
			}

			if note.ID != tt.noteID {
				t.Errorf("返回的笔记ID不匹配: 得到 %v, 期望 %v", note.ID, tt.noteID)
			}

			if note.Title == "" {
				t.Error("返回的笔记缺少标题")
			}
		})
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestRootDeleteNote 测试管理员删除笔记
func TestRootDeleteNote(t *testing.T) {
	service, cleanup := setupNoteService(t)
	defer cleanup()

	// 使用全局的 database.DB 创建测试笔记数据
	testNotes := createTestNotes(t, database.DB)

	// 获取第一个笔记的ID用于删除测试
	var firstNoteID uint
	if len(testNotes) > 0 {
		firstNoteID = testNotes[0].ID
	} else {
		t.Fatal("没有创建测试笔记")
	}

	tests := []struct {
		name        string
		noteID      uint
		wantErr     bool
		errContains string
	}{
		{
			name:    "成功删除存在的笔记",
			noteID:  firstNoteID,
			wantErr: false,
		},
		{
			name:        "删除不存在的笔记",
			noteID:      99999,
			wantErr:     true,
			errContains: "笔记不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RootDeleteNote(tt.noteID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("RootDeleteNote() 期望返回错误，但没有")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("错误消息不包含期望的字符串: 得到 %v, 期望包含 %v", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("RootDeleteNote() 意外返回错误: %v", err)
				return
			}

			// 验证笔记是否真的被删除
			if tt.name == "成功删除存在的笔记" {
				// 尝试再次获取该笔记
				_, err := service.RootGetNoteByID(tt.noteID)
				if err == nil {
					t.Error("笔记应该已被删除，但仍能获取到")
				} else if !contains(err.Error(), "笔记不存在") {
					t.Errorf("期望'笔记不存在'错误，但得到: %v", err)
				}
			}
		})
	}
}
