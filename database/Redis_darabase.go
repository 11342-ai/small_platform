package database

import (
	"context"
	"github.com/go-redis/redis/v8"
	"log"
)

var RedisClient *redis.Client
var RedisAvailable bool = false // 新增：标记Redis是否可用

func InitRedis(addr, password string, db int) error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// 测试连接
	ctx := context.Background()
	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		log.Printf("Redis连接失败，将使用降级模式: %v", err)
		RedisAvailable = false
		return nil // 不返回错误，让程序继续运行
	}

	RedisAvailable = true
	log.Println("Redis连接成功")
	return nil
}

func GetRedis() *redis.Client {
	if !RedisAvailable {
		return nil
	}
	return RedisClient
}

// IsRedisAvailable 新增：检查Redis是否可用
func IsRedisAvailable() bool {
	return RedisAvailable
}
