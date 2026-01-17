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

	RedisHost     string `mapstructure:"REDIS_HOST"`
	RedisPort     string `mapstructure:"REDIS_PORT"`
	RedisPassword string `mapstructure:"REDIS_PASSWORD"`
	RedisDB       int    `mapstructure:"REDIS_DB"`

	AdminUsername string `mapstructure:"ADMIN_USERNAME"`
	AdminPassword string `mapstructure:"ADMIN_PASSWORD"`
	AdminEmail    string `mapstructure:"ADMIN_EMAIL"`
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

	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return fmt.Errorf("配置文件未找到，使用环境变量")
		}
	}

	if err := viper.Unmarshal(&Cfg); err != nil {
		return fmt.Errorf("解析配置失败:%s", err)
	}

	// 必须配置项验证
	if Cfg.SecretKey == "" {
		return fmt.Errorf("SECRET_KEY 必须配置")
	}
	return nil
}
