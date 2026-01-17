package Note

import (
	"errors"
	"gorm.io/gorm"
	"platfrom/database"
	"strings"
)

type NoteServiceInterface interface {
	CreateNote(note *database.Note) error
	UpdateNote(UserID uint, id uint, note *database.Note) error
	DeleteNote(UserID uint, id uint) error
	GetNoteByID(UserID uint, id uint) (*database.Note, error)
	GetAllNotes(UserID uint) ([]database.Note, error)
	GetNotesByCategory(UserID uint, category string) ([]database.Note, error)
	GetNotesByTag(UserID uint, tag string) ([]database.Note, error)
	SearchNotes(UserID uint, keyword string) ([]database.Note, error)

	// RootGetAllNotes ← 新增：管理员功能
	RootGetAllNotes(userID uint, page, pageSize int) ([]database.Note, int64, error)
	RootDeleteNote(noteID uint) error
	RootGetNoteByID(noteID uint) (*database.Note, error)
}

var GlobalNoteService NoteServiceInterface

type NoteService struct {
	db *gorm.DB
}

func NewNoteService() NoteServiceInterface {
	service := &NoteService{
		db: database.DB,
	}
	GlobalNoteService = service
	return service
}

// CreateNote 创建笔记
func (s *NoteService) CreateNote(note *database.Note) error {
	if note.Title == "" {
		return errors.New("标题不能为空")
	}
	return s.db.Create(note).Error
}

// UpdateNote 更新笔记
func (s *NoteService) UpdateNote(UserID uint, id uint, note *database.Note) error {
	if note.Title == "" {
		return errors.New("标题不能为空")
	}

	// 直接更新，不需要先查询
	result := s.db.Model(&database.Note{}).
		Where("user_id = ? AND id = ?", UserID, id).
		Updates(note)

	if result.Error != nil {
		return result.Error
	}

	// 如果影响行数为 0，说明笔记不存在
	if result.RowsAffected == 0 {
		return errors.New("笔记不存在或无权限修改")
	}

	return nil
}

// DeleteNote 删除笔记
func (s *NoteService) DeleteNote(UserID uint, id uint) error {
	// 检查笔记是否存在
	var note database.Note
	err := s.db.Where("user_id = ? AND id = ?", UserID, id).First(&note, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("笔记不存在")
		}
		return err
	}
	return s.db.Delete(&note).Error
}

// GetNoteByID 根据ID获取笔记
func (s *NoteService) GetNoteByID(UserID uint, id uint) (*database.Note, error) {
	var note database.Note
	err := s.db.Where("user_id = ? AND id = ?", UserID, id).First(&note).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("笔记不存在")
		}
		return nil, err
	}
	return &note, nil
}

// GetAllNotes 获取所有笔记
func (s *NoteService) GetAllNotes(UserID uint) ([]database.Note, error) {
	var notes []database.Note
	err := s.db.Where("user_id = ? ", UserID).Order("created_at DESC").Find(&notes).Error
	return notes, err
}

// GetNotesByCategory 根据分类获取笔记
func (s *NoteService) GetNotesByCategory(UserID uint, category string) ([]database.Note, error) {
	var notes []database.Note
	err := s.db.Where("category = ? AND user_id = ?", category, UserID).Order("created_at DESC").Find(&notes).Error
	return notes, err
}

// GetNotesByTag 根据标签获取笔记
func (s *NoteService) GetNotesByTag(UserID uint, tag string) ([]database.Note, error) {
	var notes []database.Note
	// 由于SQLite不支持数组查询，我们使用LIKE查询JSON数组
	err := s.db.Where("tags LIKE ? AND user_id = ?", "%\""+tag+"\"%", UserID).Order("created_at DESC").Find(&notes).Error
	return notes, err
}

// SearchNotes 搜索笔记
func (s *NoteService) SearchNotes(UserID uint, keyword string) ([]database.Note, error) {
	var notes []database.Note
	searchPattern := "%" + strings.ToLower(keyword) + "%"
	err := s.db.Where("(LOWER(title) LIKE ? OR LOWER(content) LIKE ?) AND user_id = ?",
		searchPattern, searchPattern, UserID).Order("created_at DESC").Find(&notes).Error
	return notes, err
}

// ========== ROOT ==========

// RootGetAllNotes 管理员获取所有笔记（可按用户筛选）
func (s *NoteService) RootGetAllNotes(userID uint, page, pageSize int) ([]database.Note, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var notes []database.Note
	var total int64

	query := database.DB.Model(&database.Note{})

	// 如果指定了用户ID，则筛选该用户的笔记
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询，按更新时间倒序
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).
		Order("updated_at DESC").
		Find(&notes).Error; err != nil {
		return nil, 0, err
	}

	return notes, total, nil
}

// RootGetNoteByID 管理员获取指定笔记详情（无需权限验证）
func (s *NoteService) RootGetNoteByID(noteID uint) (*database.Note, error) {
	var note database.Note
	if err := database.DB.First(&note, noteID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("笔记不存在")
		}
		return nil, err
	}
	return &note, nil
}

// RootDeleteNote 管理员删除笔记（硬删除）
func (s *NoteService) RootDeleteNote(noteID uint) error {
	// 先检查笔记是否存在
	var note database.Note
	if err := database.DB.First(&note, noteID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("笔记不存在")
		}
		return err
	}

	// 删除笔记（软删除，因为 Note 内嵌了 gorm.Model）
	if err := database.DB.Delete(&note).Error; err != nil {
		return err
	}

	return nil
}
