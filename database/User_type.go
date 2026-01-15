package database

import (
	"gorm.io/gorm"
	"time"
)

// User 用户数据存储结构
type User struct {
	gorm.Model
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"uniqueIndex;not null;size:50"`
	PasswordHash string `gorm:"not null;size:255"`
	Email        string `gorm:"size:100"`
	LastLogin    time.Time
}

// RegisterRequest 注册时候的请求结构体
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6,max=100"`
	Email    string `json:"email" binding:"omitempty,email"`
}

// LoginRequest 登录请求结构体
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UpdateProfileRequest 更新资料请求结构体
type UpdateProfileRequest struct {
	Email    string `json:"email" binding:"omitempty,email"`
	Grade    string `json:"grade"`
	Subjects string `json:"subjects"`
}

// UserResponse 用户响应结构体
type UserResponse struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// LoginResponse 登录响应结构体
type LoginResponse struct {
	Message string       `json:"message"`
	Token   string       `json:"token"`
	User    UserResponse `json:"user"`
}

// SendCodeRequest 发送验证码请求
type SendCodeRequest struct {
	Username string `json:"username" binding:"required"`
	CodeType string `json:"code_type" binding:"required,oneof=password_reset"`
}

// VerifyCodeRequest 验证验证码请求
type VerifyCodeRequest struct {
	Username string `json:"username" binding:"required"`
	Code     string `json:"code" binding:"required,len=6"`
	CodeType string `json:"code_type" binding:"required,oneof=password_reset"`
}

// CodeResponse 验证码响应结构体
type CodeResponse struct {
	Message string `json:"message"`
	Expires int    `json:"expires_in"` // 有效时间（分钟）
}

// ResetPasswordRequest 忘记密码重置请求
type ResetPasswordRequest struct {
	Username    string `json:"username" binding:"required"`
	Code        string `json:"code" binding:"required,len=6"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=100"`
}

// UpdatePasswordRequest 修改密码请求（需要旧密码）
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=100"`
}

// VerificationCode 验证码表
type VerificationCode struct {
	gorm.Model
	Username  string    `gorm:"not null;size:50;index"`
	Code      string    `gorm:"not null;size:6"`
	ExpiresAt time.Time `gorm:"not null"`
	Used      bool      `gorm:"default:false"`
	CodeType  string    `gorm:"size:20"` // 验证码类型: password_reset, register, etc.
}
