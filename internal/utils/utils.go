package utils

import (
	"fmt"
	"net/mail"

	"gomailapi2/internal/domain"

	"github.com/cespare/xxhash"
)

// GenerateCacheKey 生成基于 refresh token 短哈希的缓存键
func GenerateCacheKey(refreshToken string) string {
	hash := xxhash.Sum64([]byte(refreshToken))
	shortHash := fmt.Sprintf("%016x", hash) // 16字符十六进制
	return fmt.Sprintf("access_token:%s", shortHash)
}

// CleanEmailAddress 清理邮件地址，支持 *mail.Address 和 *domain.EmailAddress
func CleanEmailAddress(emailAddress interface{}) *domain.EmailAddress {
	var name, address string

	switch addr := emailAddress.(type) {
	case *domain.EmailAddress:
		name = addr.Name
		address = addr.Address
	case *mail.Address:
		name = addr.Name
		address = addr.Address
	default:
		// 如果类型不匹配，返回空的 EmailAddress
		return &domain.EmailAddress{}
	}

	var optName string
	if name == address {
		optName = ""
	} else {
		optName = name
	}

	return &domain.EmailAddress{
		Name:    optName,
		Address: address,
	}
}
