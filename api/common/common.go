package common

import (
	"gomailapi2/internal/client/imap/outlook"
	"gomailapi2/internal/config"
	"gomailapi2/internal/provider/token"
	"gomailapi2/internal/types"
)

// 订阅配置常量
const (
	TimeoutMinutes           = 3  // 连接超时时间（分钟）
	HeartbeatIntervalSeconds = 60 // 心跳间隔（秒）
)

// 全局变量
var (
	GraphNotificationURL string // Graph webhook 通知 URL
)

// InitGraphNotificationURL 初始化 Graph webhook 通知 URL
func InitGraphNotificationURL(cfg *config.WebhookConfig) {
	GraphNotificationURL = cfg.BaseURL + "/gomailapi2/graph/webhook"
}

// GetTokens 获取访问令牌和刷新令牌
func GetTokens(tokenProvider *token.TokenProvider, refreshNeeded bool, mailInfo *types.MailInfo) (string, string, error) {
	var accessToken, refreshToken string
	var err error

	if refreshNeeded {
		accessToken, refreshToken, err = tokenProvider.GetBothTokens(mailInfo)
	} else {
		accessToken, err = tokenProvider.GetAccessToken(mailInfo)
	}

	return accessToken, refreshToken, err
}

// MailInfoToCredentials 将 mailInfo 转换为 credentials
func MailInfoToCredentials(mailInfo *types.MailInfo) *outlook.Credentials {
	return &outlook.Credentials{
		Email:        mailInfo.Email,
		ClientID:     mailInfo.ClientID,
		RefreshToken: mailInfo.RefreshToken,
	}
}
