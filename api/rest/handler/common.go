package handler

import (
	"gomailapi2/internal/client/imap/outlook"
	"gomailapi2/internal/provider/token"
	"gomailapi2/internal/types"
)

// getTokens 获取访问令牌和刷新令牌
func getTokens(tokenProvider *token.TokenProvider, refreshNeeded bool, mailInfo *types.MailInfo) (string, string, error) {
	var accessToken, refreshToken string
	var err error

	if refreshNeeded {
		accessToken, refreshToken, err = tokenProvider.GetBothTokens(mailInfo)
	} else {
		accessToken, err = tokenProvider.GetAccessToken(mailInfo)
	}

	return accessToken, refreshToken, err
}

// mailInfoToCredentials 将 mailInfo 转换为 credentials
func mailInfoToCredentials(mailInfo *types.MailInfo) *outlook.Credentials {
	return &outlook.Credentials{
		Email:        mailInfo.Email,
		ClientID:     mailInfo.ClientID,
		RefreshToken: mailInfo.RefreshToken,
	}
}
