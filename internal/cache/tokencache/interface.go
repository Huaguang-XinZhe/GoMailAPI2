package tokencache

import "time"

// Cache 定义缓存接口
type Cache interface {
	// GetAccessToken 使用 refresh token 获取 access token
	GetAccessToken(refreshToken string) (string, error)

	// SetAccessToken 缓存 access token
	SetAccessToken(refreshToken string, token string, expiration time.Duration) error

	// // DeleteAccessToken 删除 access token
	// DeleteAccessToken(refreshToken string) error

	// Close 关闭缓存连接
	Close() error
}
