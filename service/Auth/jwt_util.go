package Auth

import (
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"platfrom/Config"
	"strconv"
	"time"
)

type Claims struct {
	UserID   uint   `json:"sub"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT令牌（兼容旧版本，默认角色为 user）
func GenerateToken(UserID uint, username string, role ...string) (string, error) {
	userRole := "user" // 默认角色
	if len(role) > 0 && role[0] != "" {
		userRole = role[0]
	}

	expirationTime := time.Now().Add(time.Duration(Config.Cfg.TokenExpiry) * time.Minute)
	claims := &Claims{
		UserID:   UserID,
		Username: username,
		Role:     userRole, // ← 加入角色
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   strconv.FormatUint(uint64(UserID), 10),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(Config.Cfg.SecretKey))
}

// ValidateToken 验证JWT令牌
func ValidateToken(tokenString string) (*Claims, error) {

	if Config.Cfg.SecretKey == "" {
		return nil, errors.New("配置未初始化")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(Config.Cfg.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}
