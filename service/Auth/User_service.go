package Auth

import (
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"math/rand"
	"platfrom/database"
	"strings"
	"time"
)

// GlobalUserService 全局 UserService 实例
var GlobalUserService UserService

// UserService 用户服务接口
type UserService interface {
	CreateUser(req database.RegisterRequest) (*database.User, error)
	GetUserByUsername(username string) (*database.User, error)
	GetUserByID(id uint) (*database.User, error)

	// SendVerificationCode 验证码相关功能
	SendVerificationCode(username, codeType string) (*database.VerificationCode, error)
	VerifyCode(username, code, codeType string) (bool, error)

	// ResetPassword 密码相关功能
	ResetPassword(username, code, newPassword string) error            // 忘记密码重置（通过验证码）
	UpdatePassword(userID uint, oldPassword, newPassword string) error // 修改密码（需要旧密码）

	// StartCleanupTask 启动验证码清理任务
	StartCleanupTask()
}

// 用户服务实现
type userService struct {
	db *gorm.DB
}

func NewUserService() UserService {
	userService := &userService{db: database.DB}
	GlobalUserService = userService
	return userService
}

// CreateUser 创建用户
func (s *userService) CreateUser(req database.RegisterRequest) (*database.User, error) {
	var existingUser database.User
	err := s.db.Where("username = ?", req.Username).First(&existingUser).Error
	if err == nil {
		return nil, errors.New("用户名已存在")
	}
	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &database.User{
		Username:     req.Username,
		PasswordHash: hashedPassword,
		Email:        req.Email,
	}
	err = s.db.Create(user).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *userService) GetUserByUsername(username string) (*database.User, error) {
	var user database.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID 根据ID获取用户
func (s *userService) GetUserByID(id uint) (*database.User, error) {
	var user database.User
	err := s.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// 生成随机验证码
func generateRandomCode() string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	// 生成6位数字验证码
	return fmt.Sprintf("%06d", rng.Intn(1000000))
}

// SendVerificationCode 发送验证码
func (s *userService) SendVerificationCode(username, codeType string) (*database.VerificationCode, error) {
	// 检查用户是否存在
	user, err := s.GetUserByUsername(username)
	if err != nil {
		return nil, errors.New("用户不存在")
	}

	// 清理该用户之前的同类型验证码
	s.db.Where("username = ? AND code_type = ?", username, codeType).Delete(&database.VerificationCode{})

	// 生成验证码
	code := generateRandomCode()
	expiresAt := time.Now().Add(5 * time.Minute) // 5分钟有效期

	verificationCode := &database.VerificationCode{
		Username:  username,
		Code:      code,
		ExpiresAt: expiresAt,
		CodeType:  codeType,
		Used:      false,
	}

	// 保存验证码到数据库
	if err := s.db.Create(verificationCode).Error; err != nil {
		return nil, err
	}

	// 打印验证码到控制台（生产环境应该发送短信或邮件）
	fmt.Printf("用户 %s 的验证码: %s (有效期至: %s)\n",
		user.Username, code, expiresAt.Format("2006-01-02 15:04:05"))

	return verificationCode, nil
}

// VerifyCode 验证验证码
func (s *userService) VerifyCode(username, code, codeType string) (bool, error) {
	var verificationCode database.VerificationCode

	// 查找未使用的验证码
	err := s.db.Where("username = ? AND code = ? AND code_type = ? AND used = ?",
		username, code, codeType, false).First(&verificationCode).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, errors.New("验证码无效或已使用")
		}
		return false, err
	}

	// 检查是否过期
	if time.Now().After(verificationCode.ExpiresAt) {
		return false, errors.New("验证码已过期")
	}

	return true, nil
}

// ResetPassword 忘记密码重置（通过验证码）
func (s *userService) ResetPassword(username, code, newPassword string) error {
	// 验证验证码
	isValid, err := s.VerifyCode(username, code, "password_reset")
	if err != nil {
		return err
	}

	if !isValid {
		return errors.New("验证码验证失败")
	}

	// 查找用户
	user, err := s.GetUserByUsername(username)
	if err != nil {
		return errors.New("用户不存在")
	}

	// 哈希新密码
	hashedPassword, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	// 更新密码
	user.PasswordHash = hashedPassword
	if err := s.db.Save(user).Error; err != nil {
		return err
	}

	// 标记验证码为已使用
	s.db.Model(&database.VerificationCode{}).
		Where("username = ? AND code = ?", username, code).
		Update("used", true)

	// 清理该用户的所有验证码
	s.db.Where("username = ?", username).Delete(&database.VerificationCode{})

	return nil
}

// UpdatePassword 修改密码（需要旧密码验证）
func (s *userService) UpdatePassword(userID uint, oldPassword, newPassword string) error {
	// 查找用户
	user, err := s.GetUserByID(userID)
	if err != nil {
		return errors.New("用户不存在")
	}

	// 验证旧密码
	if !VerifyPassword(oldPassword, user.PasswordHash) {
		return errors.New("旧密码错误")
	}

	// 验证新密码不能与旧密码相同
	if strings.TrimSpace(oldPassword) == strings.TrimSpace(newPassword) {
		return errors.New("新密码不能与旧密码相同")
	}

	// 哈希新密码
	hashedPassword, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	// 更新密码
	user.PasswordHash = hashedPassword
	if err := s.db.Save(user).Error; err != nil {
		return err
	}

	return nil
}

// StartCleanupTask 启动验证码清理任务
func (s *userService) StartCleanupTask() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour) // 每小时清理一次
		defer ticker.Stop()

		for range ticker.C {
			cleanupExpiredCodes()
		}
	}()
}

// 清理过期验证码
func cleanupExpiredCodes() {
	db := database.DB
	now := time.Now()

	// 删除已过期的验证码
	db.Where("expires_at < ?", now).Delete(&database.VerificationCode{})

	// 删除24小时前的已使用验证码
	twentyFourHoursAgo := now.Add(-24 * time.Hour)
	db.Where("used = ? AND updated_at < ?", true, twentyFourHoursAgo).Delete(&database.VerificationCode{})
}

// HashPassword 将密码哈希化
func HashPassword(password string) (string, error) {
	if len(password) > 72 {
		password = password[:72]
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// VerifyPassword 验证哈希密码
func VerifyPassword(password, hash string) bool {
	if len(password) > 72 {
		password = password[:72]
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
