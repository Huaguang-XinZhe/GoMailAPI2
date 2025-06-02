package common

import (
	"gomailapi2/internal/client/imap/outlook"
	"gomailapi2/internal/provider/token"
	"gomailapi2/internal/types"
)

// 订阅配置常量
const (
	TimeoutMinutes           = 3  // 连接超时时间（分钟）
	HeartbeatIntervalSeconds = 60 // 心跳间隔（秒）
	// todo IP 自动化设置（初始化时设置，作为全局变量）
	GraphNotificationURL = "https://8e77-2408-8948-2011-5678-a96a-ba3e-7315-342.ngrok-free.app/gomailapi2/graph/webhook"
)

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
