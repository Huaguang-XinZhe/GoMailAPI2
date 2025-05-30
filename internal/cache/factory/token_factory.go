package factory

import (
	"fmt"
	"gomailapi2/internal/config"

	"gomailapi2/internal/cache/tokencache"
)

// CacheType 缓存类型
type CacheType string

const (
	CacheTypeLocal      CacheType = "local"      // 本地缓存
	CacheTypeRedis      CacheType = "redis"      // Redis 缓存
	CacheTypeMultiLevel CacheType = "multilevel" // 多级缓存
)

// NewCache 根据配置创建缓存实例
func NewCache(cacheConfig config.CacheConfig) (tokencache.Cache, error) {
	switch CacheType(cacheConfig.Type) {
	case CacheTypeLocal:
		return newLocalCache(cacheConfig.Local.Size)

	case CacheTypeRedis:
		return newRedisCache(cacheConfig.Redis)

	case CacheTypeMultiLevel:
		// 创建本地缓存（L1）
		l1Cache, err := newLocalCache(cacheConfig.Local.Size)
		if err != nil {
			return nil, fmt.Errorf("failed to create L1 cache: %w", err)
		}

		// 创建 Redis 缓存（L2）
		l2Cache, err := newRedisCache(cacheConfig.Redis)
		if err != nil {
			return nil, fmt.Errorf("failed to create L2 cache: %w", err)
		}

		return newMultiLevelCache(l1Cache, l2Cache), nil

	default:
		return nil, fmt.Errorf("unsupported cache type: %s", cacheConfig.Type)
	}
}

// newLocalCache 创建本地缓存实例
func newLocalCache(size int) (tokencache.Cache, error) {
	return tokencache.NewLocalCache(size)
}

// newRedisCache 创建 Redis 缓存实例
func newRedisCache(redisConfig config.RedisConfig) (tokencache.Cache, error) {
	return tokencache.NewRedisClient(redisConfig)
}

// newMultiLevelCache 创建多级缓存实例
func newMultiLevelCache(l1Cache, l2Cache tokencache.Cache) tokencache.Cache {
	return tokencache.NewMultiLevelCache(l1Cache, l2Cache)
}
