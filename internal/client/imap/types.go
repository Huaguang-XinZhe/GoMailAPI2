package imap

import (
	"github.com/emersion/go-sasl"
)

// ImapConfig IMAP 连接配置
type ImapConfig struct {
	Host     string // IMAP 服务器地址，如 "outlook.office365.com:993"
	Username string // 用户名/邮箱地址
	UseTLS   bool   // 是否使用 TLS
}

// AuthProvider 认证提供者接口
type AuthProvider interface {
	// GetSASLClient 获取 SASL 认证客户端
	GetSASLClient() (sasl.Client, error)
}
