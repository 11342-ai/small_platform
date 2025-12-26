package LLM_Chat

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"os"
	"platfrom/database"
	"sync"
	"time"
)

// FileUploadConfigManager 文件上传配置管理器接口
type FileUploadConfigManager interface {
	LoadConfig(filePath string) error
	GetUploadDir() string
	GetMaxFileSize() int64
	GetAllowedExtensions() []string
	ReloadConfig() error
}

// FileUploadConfigManagerImpl 基于文件的配置管理器实现
type FileUploadConfigManagerImpl struct {
	config     *database.FileUploadConfig
	configPath string
	mu         sync.RWMutex
}

var GlobalFileUploadConfigManager FileUploadConfigManager

// NewFileUploadConfigManager 创建新的文件上传配置管理器
func NewFileUploadConfigManager() FileUploadConfigManager {
	manager := &FileUploadConfigManagerImpl{
		config: &database.FileUploadConfig{},
	}
	GlobalFileUploadConfigManager = manager
	return manager
}

// LoadConfig 加载配置文件
func (m *FileUploadConfigManagerImpl) LoadConfig(filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 读取文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 定义一个临时结构体来解析完整配置
	var fullConfig struct {
		FileUpload database.FileUploadConfig `yaml:"file_upload"`
	}

	// 解析YAML
	if err := yaml.Unmarshal(data, &fullConfig); err != nil {
		return fmt.Errorf("解析YAML配置失败: %w", err)
	}

	m.config = &fullConfig.FileUpload
	m.configPath = filePath

	return nil
}

// GetUploadDir 获取上传目录路径
func (m *FileUploadConfigManagerImpl) GetUploadDir() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return ""
	}
	return m.config.UploadDir
}

// GetMaxFileSize 获取最大文件大小
func (m *FileUploadConfigManagerImpl) GetMaxFileSize() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return 0
	}
	return m.config.MaxFileSize
}

// GetAllowedExtensions 获取允许的文件扩展名
func (m *FileUploadConfigManagerImpl) GetAllowedExtensions() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return []string{}
	}

	// 返回副本，避免外部修改
	extensions := make([]string, len(m.config.AllowedExtensions))
	copy(extensions, m.config.AllowedExtensions)
	return extensions
}

// ReloadConfig 重新加载配置
func (m *FileUploadConfigManagerImpl) ReloadConfig() error {
	if m.configPath == "" {
		return fmt.Errorf("未设置配置文件路径")
	}
	return m.LoadConfig(m.configPath)
}

// = = = = = = = = = = = = = = = = = = = = =

// UserAPIService 用户API配置服务接口
type UserAPIService interface {
	// API配置管理
	CreateAPI(userID uint, api *database.UserAPI) (*database.UserAPI, error)
	GetAPIByID(apiID uint) (*database.UserAPI, error)
	GetAPIByName(userID uint, apiName string) (*database.UserAPI, error)
	GetUserAPIs(userID uint) ([]database.UserAPI, error)
	UpdateAPI(apiID uint, updates map[string]interface{}) error
	DeleteAPI(apiID uint) error

	// API验证与选择
	TestAPIConnection() (bool, error)
	GetFirstAvailableAPI(userID uint) (*database.UserAPI, error)
}

// GlobalUserAPIService 全局UserAPIService实例
var GlobalUserAPIService UserAPIService

// userAPIService 用户API配置服务实现
type userAPIService struct {
	db *gorm.DB
	mu sync.RWMutex
}

// NewUserAPIService 创建新的UserAPI服务
func NewUserAPIService() UserAPIService {
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

// = = = = = = = = = = = = = = = = = = = = =

// SessionCreateRequest 辅助结构体
type SessionCreateRequest struct {
	Title       string `json:"title"`
	PersonaName string `json:"persona_name"`
}

type SessionUpdateRequest struct {
	Title       string `json:"title,omitempty"`
	PersonaName string `json:"persona_name,omitempty"`
}

type SessionResponse struct {
	SessionID     string    `json:"session_id"`
	Title         string    `json:"title"`
	PersonaName   string    `json:"persona_name"`
	LastMessageAt time.Time `json:"last_message_at"`
	MessageCount  int       `json:"message_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// = = = = = = = = = = = = = = = =

// ConfigManager 配置管理器接口
type ConfigManager interface {
	LoadConfig(filePath string) error
	GetPersona(name string) (*database.Persona, error)
	GetAllPersonas() []database.Persona
	ReloadConfig() error
	GetDefaultPersona() (*database.Persona, error)
}

// GlobalConfigManager 全局 ConfigManager 实例
var GlobalConfigManager ConfigManager

// PersonaConfigManager 基于文件的配置管理器
type PersonaConfigManager struct {
	config     *database.StyleConfig
	configPath string
	mu         sync.RWMutex
}

// NewPersonaConfigManager 创建新的配置管理器
func NewPersonaConfigManager() ConfigManager {
	manager := &PersonaConfigManager{
		config: &database.StyleConfig{},
	}
	GlobalConfigManager = manager
	return manager
}

// LoadConfig 加载配置文件
func (m *PersonaConfigManager) LoadConfig(filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 读取文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML
	var config database.StyleConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析YAML配置失败: %w", err)
	}

	m.config = &config
	m.configPath = filePath

	return nil
}

// GetPersona 根据名称获取角色配置
func (m *PersonaConfigManager) GetPersona(name string) (*database.Persona, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, persona := range m.config.Personas {
		if persona.Name == name {
			return &persona, nil
		}
	}

	return nil, fmt.Errorf("角色 '%s' 不存在", name)
}

// GetAllPersonas 获取所有角色配置
func (m *PersonaConfigManager) GetAllPersonas() []database.Persona {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回副本，避免外部修改
	personas := make([]database.Persona, len(m.config.Personas))
	copy(personas, m.config.Personas)

	return personas
}

// ReloadConfig 重新加载配置
func (m *PersonaConfigManager) ReloadConfig() error {
	if m.configPath == "" {
		return fmt.Errorf("未设置配置文件路径")
	}
	return m.LoadConfig(m.configPath)
}

// GetDefaultPersona 获取默认角色
func (m *PersonaConfigManager) GetDefaultPersona() (*database.Persona, error) {
	return m.GetPersona("默认助手")
}
