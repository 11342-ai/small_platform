package LLM_Chat

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"platfrom/database"
	"strings"
)

type FileServiceInterface interface {
	SaveFile(file *database.UploadedFile) error
	GetFilesBySession(sessionID string) ([]database.UploadedFile, error)
	GetFileByID(id uint) (*database.UploadedFile, error)
	DeleteFile(id uint) error
	ProcessFileContent(file *database.UploadedFile) (string, error)
}

// GlobalFileService 全局FileService实例
var GlobalFileService FileServiceInterface

// fileService 文件服务实现
type FileService struct {
	db *gorm.DB
}

// NewFileService 创建新的文件服务
func NewFileService(db *gorm.DB) (FileServiceInterface, error) {

	if db == nil {
		return nil, errors.New("数据库连接不能为空")
	}

	service := &FileService{
		db,
	}
	GlobalFileService = service
	return service, nil
}

func (s *FileService) SaveFile(file *database.UploadedFile) error {
	return s.db.Create(file).Error
}

func (s *FileService) GetFilesBySession(sessionID string) ([]database.UploadedFile, error) {
	var files []database.UploadedFile
	result := s.db.Where("session_id = ?", sessionID).Order("created_at DESC").Find(&files)
	return files, result.Error
}

func (s *FileService) GetFileByID(id uint) (*database.UploadedFile, error) {
	var file database.UploadedFile
	result := s.db.First(&file, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &file, nil
}

func (s *FileService) DeleteFile(id uint) error {
	return s.db.Delete(&database.UploadedFile{}, id).Error
}

// ProcessFileContent 处理文件内容，支持文本文件直接读取
func (s *FileService) ProcessFileContent(file *database.UploadedFile) (string, error) {
	// 如果是文本文件，直接读取内容
	textExtensions := []string{".txt", ".py", ".go", ".c", ".cpp", ".h", ".hpp", ".js", ".ts", ".java", ".html", ".css", ".md", ".json", ".xml", ".yaml", ".yml"}

	ext := strings.ToLower(filepath.Ext(file.FileName))
	isTextFile := false
	for _, textExt := range textExtensions {
		if ext == textExt {
			isTextFile = true
			break
		}
	}

	if isTextFile {
		// 如果文件内容已经存在，直接返回
		if file.Content != "" {
			return file.Content, nil
		}

		// 否则从文件路径读取
		if file.FilePath != "" {
			content, err := os.ReadFile(file.FilePath)
			if err != nil {
				return "", fmt.Errorf("读取文件内容失败: %v", err)
			}
			return string(content), nil
		}
	}

	return "", fmt.Errorf("不支持的文件类型或文件内容为空")
}
