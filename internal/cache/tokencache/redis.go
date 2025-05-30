package tokencache

import (
	"context"
	"fmt"
	"time"

	"gomailapi2/internal/config"
	"gomailapi2/internal/utils"

	"github.com/redis/go-redis/v9"
)

// 确保 RedisClient 实现了 Cache 接口
var _ Cache = (*RedisClient)(nil)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(redisConfig config.RedisConfig) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisConfig.Host, redisConfig.Port),
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{client: rdb}, nil
}

// GetAccessToken 使用 refresh token 的短哈希获取 access token
func (r *RedisClient) GetAccessToken(refreshToken string) (string, error) {
	key := r.generateCacheKey(refreshToken)
	ctx := context.Background()
	return r.client.Get(ctx, key).Result()
}

// SetAccessToken 使用 refresh token 的短哈希作为键缓存 access token
func (r *RedisClient) SetAccessToken(refreshToken string, token string, expiration time.Duration) error {
	key := r.generateCacheKey(refreshToken)
	ctx := context.Background()
	return r.client.Set(ctx, key, token, expiration).Err()
}

// DeleteAccessToken 使用 refresh token 的短哈希删除 access token
func (r *RedisClient) DeleteAccessToken(refreshToken string) error {
	key := r.generateCacheKey(refreshToken)
	ctx := context.Background()
	return r.client.Del(ctx, key).Err()
}

// Close 关闭 Redis 连接
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// generateCacheKey 生成基于 refresh token 短哈希的缓存键
func (r *RedisClient) generateCacheKey(refreshToken string) string {
	return utils.GenerateCacheKey(refreshToken)
}
