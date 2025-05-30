package tokencache

import (
	"fmt"
	"time"

	"gomailapi2/internal/utils"

	lru "github.com/hashicorp/golang-lru/v2"
)

// 确保 LocalCache 实现了 Cache 接口
var _ Cache = (*LocalCache)(nil)

// CacheItem 缓存项结构
type CacheItem struct {
	Value     string
	ExpiresAt time.Time
}

// LocalCache 本地内存缓存
// 注意：hashicorp/golang-lru 本身就是线程安全的，无需额外的锁
// LRU 会自动管理容量，超出容量时自动淘汰最少使用的项
type LocalCache struct {
	lru *lru.Cache[string, *CacheItem]
}

// NewLocalCache 创建新的本地缓存实例
func NewLocalCache(size int) (*LocalCache, error) {
	if size <= 0 {
		size = 1000 // 默认大小，约占用 ~236KB 内存
	}

	cache, err := lru.New[string, *CacheItem](size)
	if err != nil {
		return nil, fmt.Errorf("failed to create LRU cache: %w", err)
	}

	return &LocalCache{
		lru: cache,
	}, nil
}

// GetAccessToken 使用 refresh token 的短哈希获取 access token
func (l *LocalCache) GetAccessToken(refreshToken string) (string, error) {
	key := l.generateCacheKey(refreshToken)

	item, found := l.lru.Get(key)
	if !found {
		return "", fmt.Errorf("cache miss")
	}

	// 检查是否过期
	if time.Now().After(item.ExpiresAt) {
		l.lru.Remove(key)
		return "", fmt.Errorf("cache expired")
	}

	return item.Value, nil
}

// SetAccessToken 使用 refresh token 的短哈希作为键缓存 access token
func (l *LocalCache) SetAccessToken(refreshToken string, token string, expiration time.Duration) error {
	key := l.generateCacheKey(refreshToken)

	item := &CacheItem{
		Value:     token,
		ExpiresAt: time.Now().Add(expiration),
	}

	// LRU 会自动处理容量管理，超出容量时淘汰最少使用的项
	l.lru.Add(key, item)
	return nil
}

// DeleteAccessToken 使用 refresh token 的短哈希删除 access token
func (l *LocalCache) DeleteAccessToken(refreshToken string) error {
	key := l.generateCacheKey(refreshToken)
	l.lru.Remove(key)
	return nil
}

// Close 关闭本地缓存
func (l *LocalCache) Close() error {
	l.lru.Purge()
	return nil
}

func (l *LocalCache) generateCacheKey(refreshToken string) string {
	return utils.GenerateCacheKey(refreshToken)
}
