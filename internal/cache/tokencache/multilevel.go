package tokencache

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// 确保 MultiLevelCache 实现了 Cache 接口
var _ Cache = (*MultiLevelCache)(nil)

// MultiLevelCache 多级缓存
type MultiLevelCache struct {
	l1Cache Cache // 本地缓存（L1）
	l2Cache Cache // Redis 缓存（L2）
}

// NewMultiLevelCache 创建新的多级缓存实例
func NewMultiLevelCache(l1Cache, l2Cache Cache) *MultiLevelCache {
	return &MultiLevelCache{
		l1Cache: l1Cache,
		l2Cache: l2Cache,
	}
}

// GetAccessToken 多级缓存获取 access token
// 流程：L1 → L2 → 未命中
func (m *MultiLevelCache) GetAccessToken(refreshToken string) (string, error) {
	// 1. 先尝试从 L1（本地缓存）获取
	if token, err := m.l1Cache.GetAccessToken(refreshToken); err == nil {
		return token, nil
	}

	// 2. L1 未命中，尝试从 L2（Redis）获取
	token, err := m.l2Cache.GetAccessToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("cache miss in both L1 and L2: %w", err)
	}

	// 3. L2 命中，回填到 L1 缓存
	// 使用较短的过期时间，避免 L1 缓存过期时间比 L2 长
	if err := m.l1Cache.SetAccessToken(refreshToken, token, 50*time.Minute); err != nil {
		log.Error().Err(err).Msg("Failed to backfill L1 cache")
		// 不影响返回结果，只记录日志
	}

	return token, nil
}

// SetAccessToken 多级缓存设置 access token
// 流程：同时写入 L1 和 L2
func (m *MultiLevelCache) SetAccessToken(refreshToken string, token string, expiration time.Duration) error {
	var l1Err, l2Err error

	// 1. 写入 L1 缓存（本地）
	if m.l1Cache != nil {
		// L1 使用较短的过期时间或原始时间，取最小值
		l1Expiration := min(expiration, 50*time.Minute)
		l1Err = m.l1Cache.SetAccessToken(refreshToken, token, l1Expiration)
	}

	// 2. 写入 L2 缓存（Redis）
	if m.l2Cache != nil {
		l2Err = m.l2Cache.SetAccessToken(refreshToken, token, expiration)
	}

	// 3. 处理错误
	if l1Err != nil && l2Err != nil {
		return fmt.Errorf("failed to write to both caches - L1: %v, L2: %v", l1Err, l2Err)
	}

	if l2Err != nil {
		log.Error().Err(l2Err).Msg("Failed to write to L2 cache (Redis)")
		// L2 失败不影响整体，L1 可以继续工作
	}

	if l1Err != nil {
		log.Error().Err(l1Err).Msg("Failed to write to L1 cache (Local)")
		// L1 失败也不影响整体，L2 可以继续工作
	}

	return nil
}

// // DeleteAccessToken 多级缓存删除 access token
// // 流程：同时从 L1 和 L2 删除
// func (m *MultiLevelCache) DeleteAccessToken(refreshToken string) error {
// 	var l1Err, l2Err error

// 	// 1. 从 L1 缓存删除
// 	if m.l1Cache != nil {
// 		l1Err = m.l1Cache.DeleteAccessToken(refreshToken)
// 	}

// 	// 2. 从 L2 缓存删除
// 	if m.l2Cache != nil {
// 		l2Err = m.l2Cache.DeleteAccessToken(refreshToken)
// 	}

// 	// 3. 处理错误
// 	if l1Err != nil {
// 		log.Error().Err(l1Err).Msg("Failed to delete from L1 cache")
// 	}

// 	if l2Err != nil {
// 		log.Error().Err(l2Err).Msg("Failed to delete from L2 cache")
// 	}

// 	// 即使部分删除失败也返回成功，因为缓存最终会过期
// 	return nil
// }

// Close 关闭多级缓存
func (m *MultiLevelCache) Close() error {
	var l1Err, l2Err error

	// 关闭 L1 缓存
	if m.l1Cache != nil {
		l1Err = m.l1Cache.Close()
	}

	// 关闭 L2 缓存
	if m.l2Cache != nil {
		l2Err = m.l2Cache.Close()
	}

	// 处理错误
	if l1Err != nil && l2Err != nil {
		return fmt.Errorf("failed to close both caches - L1: %v, L2: %v", l1Err, l2Err)
	}

	if l1Err != nil {
		return fmt.Errorf("failed to close L1 cache: %w", l1Err)
	}

	if l2Err != nil {
		return fmt.Errorf("failed to close L2 cache: %w", l2Err)
	}

	return nil
}
