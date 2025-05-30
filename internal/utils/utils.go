package utils

import (
	"fmt"

	"github.com/cespare/xxhash"
)

// GenerateCacheKey 生成基于 refresh token 短哈希的缓存键
func GenerateCacheKey(refreshToken string) string {
	hash := xxhash.Sum64([]byte(refreshToken))
	shortHash := fmt.Sprintf("%016x", hash) // 16字符十六进制
	return fmt.Sprintf("access_token:%s", shortHash)
}
