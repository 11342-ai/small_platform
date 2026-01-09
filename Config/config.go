package Config

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
)

type Config struct {
	ServerPort  string `mapstructure:"SERVER_PORT"`
	DatabaseURL string `mapstructure:"DATABASE_URL"`
	SecretKey   string `mapstructure:"SECRET_KEY"`
	TokenExpiry int    `mapstructure:"TOKEN_EXPIRY_MINUTES"`
}

var Cfg Config

func InitConfig() error {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	// 设置默认值
	viper.SetDefault("SERVER_PORT", "8000")
	viper.SetDefault("DATABASE_URL", "sqlite://k12_platform.db")
	viper.SetDefault("TOKEN_EXPIRY_MINUTES", 1440) // 24小时

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return fmt.Errorf("配置文件未找到，使用环境变量")
		}
	}

	if err := viper.Unmarshal(&Cfg); err != nil {
		return fmt.Errorf("解析配置失败:", err)
	}

	// 必须配置项验证
	if Cfg.SecretKey == "" {
		return fmt.Errorf("SECRET_KEY 必须配置")
	}
	return nil
}
