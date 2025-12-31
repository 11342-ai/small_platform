package LLM_Chat

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"platfrom/database"
	"sync"
)

// UserAPIServiceInterface 用户API配置服务接口
type UserAPIServiceInterface interface {
	// CreateAPI API配置管理
	CreateAPI(userID uint, api *database.UserAPI) (*database.UserAPI, error)
	GetAPIByID(apiID uint) (*database.UserAPI, error)
	GetAPIByName(userID uint, apiName string) (*database.UserAPI, error)
	GetAPIByModelName(userID uint, modelName string) (*database.UserAPI, error)
	GetUserAPIs(userID uint) ([]database.UserAPI, error)
	UpdateAPI(apiID uint, updates map[string]interface{}) error
	DeleteAPI(apiID uint) error

	// TestAPIConnection API验证与选择
	TestAPIConnection() (bool, error)
	GetFirstAvailableAPI(userID uint) (*database.UserAPI, error)
}

// GlobalUserAPIService 全局UserAPIService实例
var GlobalUserAPIService UserAPIServiceInterface

// userAPIService 用户API配置服务实现
type userAPIService struct {
	db *gorm.DB
	mu sync.RWMutex
}

// NewUserAPIService 创建新的UserAPI服务
func NewUserAPIService() UserAPIServiceInterface {
	service := &userAPIService{
		db: database.DB,
	}
	GlobalUserAPIService = service
	return service
}

// CreateAPI 创建新的API配置
func (s *userAPIService) CreateAPI(userID uint, api *database.UserAPI) (*database.UserAPI, error) {
	if userID == 0 {
		return nil, errors.New("用户ID不能为空")
	}
	if api.APIName == "" {
		return nil, errors.New("API名称不能为空")
	}
	if api.APIKey == "" {
		return nil, errors.New("API密钥不能为空")
	}
	// 检查同名的API是否已存在（重要：这里是修正的逻辑）
	var existingAPI database.UserAPI
	err := s.db.Where("user_id = ? AND api_name = ?", userID, api.APIName).First(&existingAPI).Error

	// 修正：如果查询成功（err == nil），说明已存在同名API
	if err == nil {
		return nil, errors.New("该API名称已存在")
	}
	// 如果错误是记录不存在，可以继续创建
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// 其他数据库错误
		return nil, fmt.Errorf("查询API配置失败: %w", err)
	}
	// 设置用户ID
	api.UserID = userID
	// 创建API配置
	if err := s.db.Create(api).Error; err != nil {
		return nil, fmt.Errorf("创建API配置失败: %w", err)
	}
	return api, nil
}

// GetAPIByID 根据ID获取API配置
func (s *userAPIService) GetAPIByID(apiID uint) (*database.UserAPI, error) {
	var api database.UserAPI
	if err := s.db.First(&api, apiID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("API配置不存在")
		}
		return nil, fmt.Errorf("查询API配置失败: %w", err)
	}

	return &api, nil
}

// GetAPIByName 根据名称获取用户的API配置
func (s *userAPIService) GetAPIByName(userID uint, apiName string) (*database.UserAPI, error) {
	var api database.UserAPI
	if err := s.db.Where("user_id = ? AND api_name = ?", userID, apiName).First(&api).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("API配置不存在")
		}
		return nil, fmt.Errorf("查询API配置失败: %w", err)
	}

	return &api, nil
}

// GetAPIByModelName 根据模型名称获取用户的API配置
func (s *userAPIService) GetAPIByModelName(userID uint, modelName string) (*database.UserAPI, error) {
	var api database.UserAPI
	if err := s.db.Where("user_id = ? AND model_name = ?", userID, modelName).First(&api).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("API配置不存在")
		}
		return nil, fmt.Errorf("查询API配置失败: %w", err)
	}

	return &api, nil
}

// GetUserAPIs 获取用户的所有API配置
func (s *userAPIService) GetUserAPIs(userID uint) ([]database.UserAPI, error) {
	var apis []database.UserAPI
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&apis).Error; err != nil {
		return nil, fmt.Errorf("查询用户API配置失败: %w", err)
	}

	return apis, nil
}

// UpdateAPI 更新API配置
func (s *userAPIService) UpdateAPI(apiID uint, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.New("更新内容不能为空")
	}
	// 检查API是否存在
	var api database.UserAPI
	if err := s.db.First(&api, apiID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("API配置不存在")
		}
		return fmt.Errorf("查询API配置失败: %w", err)
	}
	// 如果更新API名称，检查是否与其他API重名
	if newName, ok := updates["api_name"].(string); ok && newName != api.APIName {
		var existingAPI database.UserAPI
		err := s.db.Where("user_id = ? AND api_name = ? AND id != ?",
			api.UserID, newName, apiID).First(&existingAPI).Error
		if err == nil {
			return errors.New("该API名称已存在")
		}
	}
	// 执行更新
	if err := s.db.Model(&database.UserAPI{}).Where("id = ?", apiID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新API配置失败: %w", err)
	}
	return nil
}

// DeleteAPI 删除API配置
func (s *userAPIService) DeleteAPI(apiID uint) error {
	// 检查API是否存在
	var api database.UserAPI
	if err := s.db.First(&api, apiID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("API配置不存在")
		}
		return fmt.Errorf("查询API配置失败: %w", err)
	}
	// 删除API配置
	if err := s.db.Delete(&api).Error; err != nil {
		return fmt.Errorf("删除API配置失败: %w", err)
	}
	return nil
}

// TestAPIConnection 测试API连接（简单实现）
func (s *userAPIService) TestAPIConnection() (bool, error) {
	return true, nil
}

// GetFirstAvailableAPI 获取用户第一个可用的API配置
func (s *userAPIService) GetFirstAvailableAPI(userID uint) (*database.UserAPI, error) {
	var api database.UserAPI
	if err := s.db.Where("user_id = ?", userID).First(&api).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户没有配置任何API")
		}
		return nil, fmt.Errorf("获取API配置失败: %w", err)
	}
	return &api, nil
}
