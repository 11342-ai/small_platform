package LLM_Chat

import (
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"log"
	"platfrom/database"
	"time"
)

// SharedSessionServiceInterface 会话分享服务接口
type SharedSessionServiceInterface interface {
	// === 创建者操作 ===

	// CreateSharedLink 创建分享链接 // 返回: ShareID 和错误
	CreateSharedLink(sessionID string, createdBy uint, maxViews int, expiresAt *time.Time) (string, error)
	// DeleteSharedLink 删除分享链接 // 只允许创建者删除自己的分享
	DeleteSharedLink(shareID string, userID uint) error
	// UpdateSharedLink 更新分享配置 // 只允许创建者修改自己的分享
	UpdateSharedLink(shareID string, userID uint, updates map[string]interface{}) error
	// ListMySharedLinks 获取用户创建的所有分享链接 // 返回: 该用户创建的所有分享列表
	ListMySharedLinks(userID uint) ([]database.SharedSession, error)

	// === 被分享者操作 ===

	// AccessSharedLink 访问分享链接 // 返回: 完整的会话消息、SharedSession元信息、错误 // 注意: 此操作会增加 ViewCount 和更新 LastAccessAt
	AccessSharedLink(shareID string) (*database.ChatSession, []database.ChatMessage, *database.SharedSession, error)
	// GetSharedLinkInfo 获取分享链接信息（不增加访问计数） // 用于: 显示分享详情、检查有效性、预览信息等
	GetSharedLinkInfo(shareID string) (*database.SharedSession, error)
	// ValidateSharedLink 验证分享链接是否有效 // 用于: 访问前的快速检查，不修改任何数据
	ValidateSharedLink(shareID string) (bool, error)
}

// SharedSessionService 会话分享服务实现
type SharedSessionService struct {
	db *gorm.DB
}

var GlobalSharedSessionService SharedSessionServiceInterface

func NewSharedSessionService(db *gorm.DB) SharedSessionServiceInterface {
	service := &SharedSessionService{
		db: db,
	}
	GlobalSharedSessionService = service
	return service
}

// CreateSharedLink 创建分享链接
func (s *SharedSessionService) CreateSharedLink(sessionID string, createdBy uint, maxViews int, expiresAt *time.Time) (string, error) {
	// 验证会话是否存在
	var session database.ChatSession
	if err := s.db.Where("session_id = ? AND user_id = ?", sessionID, createdBy).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("会话不存在或无权限")
		}
		return "", err
	}

	// 生成唯一的 ShareID
	shareID := "share_" + uuid.New().String()[:16]

	sharedSession := database.SharedSession{
		ShareID:   shareID,
		SessionID: sessionID,
		CreatedBy: createdBy,
		IsPublic:  true,
		MaxViews:  maxViews,
		ExpiresAt: expiresAt,
	}

	if err := s.db.Create(&sharedSession).Error; err != nil {
		return "", err
	}

	log.Printf("用户 %d 创建分享链接: %s (会话: %s)", createdBy, shareID, sessionID)
	return shareID, nil
}

// DeleteSharedLink 删除分享链接
func (s *SharedSessionService) DeleteSharedLink(shareID string, userID uint) error {
	result := s.db.Where("share_id = ? AND created_by = ?", shareID, userID).Delete(&database.SharedSession{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("分享链接不存在或无权限删除")
	}
	log.Printf("用户 %d 删除分享链接: %s", userID, shareID)
	return nil
}

// UpdateSharedLink 更新分享配置
func (s *SharedSessionService) UpdateSharedLink(shareID string, userID uint, updates map[string]interface{}) error {
	// 验证权限
	var shared database.SharedSession
	if err := s.db.Where("share_id = ? AND created_by = ?", shareID, userID).First(&shared).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("分享链接不存在或无权限修改")
		}
		return err
	}

	// 限制可更新的字段
	allowedFields := map[string]bool{
		"max_views":  true,
		"expires_at": true,
		"is_public":  true,
	}

	filteredUpdates := make(map[string]interface{})
	for field, value := range updates {
		if allowedFields[field] {
			filteredUpdates[field] = value
		}
	}

	if len(filteredUpdates) == 0 {
		return errors.New("没有有效的更新字段")
	}

	if err := s.db.Model(&shared).Updates(filteredUpdates).Error; err != nil {
		return err
	}

	log.Printf("用户 %d 更新分享链接: %s, 更新内容: %v", userID, shareID, filteredUpdates)
	return nil
}

// ListMySharedLinks 获取用户创建的所有分享链接
func (s *SharedSessionService) ListMySharedLinks(userID uint) ([]database.SharedSession, error) {
	var sharedSessions []database.SharedSession
	err := s.db.Where("created_by = ?", userID).Order("created_at DESC").Find(&sharedSessions).Error
	return sharedSessions, err
}

// AccessSharedLink 访问分享链接（使用事务）
func (s *SharedSessionService) AccessSharedLink(shareID string) (*database.ChatSession, []database.ChatMessage, *database.SharedSession, error) {
	var session database.ChatSession
	var messages []database.ChatMessage
	var shared database.SharedSession

	// 使用事务保证一致性
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 查询分享链接
		if err := tx.Where("share_id = ?", shareID).First(&shared).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("分享链接不存在")
			}
			return err
		}

		// 2. 验证有效性
		now := time.Now()
		if !shared.IsPublic {
			return errors.New("分享链接已设置为私有")
		}
		if shared.ExpiresAt != nil && now.After(*shared.ExpiresAt) {
			return errors.New("分享链接已过期")
		}
		if shared.MaxViews != -1 && shared.ViewCount >= shared.MaxViews {
			return errors.New("分享链接访问次数已达上限")
		}

		// 3. 查询会话信息
		if err := tx.Where("session_id = ?", shared.SessionID).First(&session).Error; err != nil {
			return errors.New("关联的会话不存在")
		}

		// 4. 查询消息列表
		if err := tx.Where("session_id = ?", shared.SessionID).Order("created_at ASC").Find(&messages).Error; err != nil {
			return err
		}

		// 5. 更新访问计数
		updates := map[string]interface{}{
			"view_count":     shared.ViewCount + 1,
			"last_access_at": &now,
		}
		if err := tx.Model(&shared).Updates(updates).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, nil, nil, err
	}

	log.Printf("分享链接 %s 被访问 (当前计数: %d)", shareID, shared.ViewCount+1)
	return &session, messages, &shared, nil
}

// GetSharedLinkInfo 获取分享链接信息（不增加访问计数）
func (s *SharedSessionService) GetSharedLinkInfo(shareID string) (*database.SharedSession, error) {
	var shared database.SharedSession
	err := s.db.Where("share_id = ?", shareID).First(&shared).Error
	if err != nil {
		return nil, err
	}
	return &shared, nil
}

// ValidateSharedLink 验证分享链接是否有效
func (s *SharedSessionService) ValidateSharedLink(shareID string) (bool, error) {
	var shared database.SharedSession
	if err := s.db.Where("share_id = ?", shareID).First(&shared).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	// 检查有效性
	if !shared.IsPublic {
		return false, nil
	}
	if shared.ExpiresAt != nil && time.Now().After(*shared.ExpiresAt) {
		return false, nil
	}
	if shared.MaxViews != -1 && shared.ViewCount >= shared.MaxViews {
		return false, nil
	}

	return true, nil
}
