package outlook

import (
	"gomailapi2/internal/client/imap"
)

type Credentials struct {
	Email        string // 邮箱（Graph API 不需要，可能为空）
	ClientID     string // 这里如果为空字符串要补全
	RefreshToken string
}

type OutlookImapClient struct {
	*imap.CommonImapClient
	credentials *Credentials
}

func NewOutlookImapClient(credentials *Credentials, accessToken string) *OutlookImapClient {
	// 创建 IMAP 配置（微软特有）
	imapConfig := &imap.ImapConfig{
		Host:     "outlook.office365.com:993",
		Username: credentials.Email,
		UseTLS:   true,
	}

	authProvider := NewOutlookAuthProvider(credentials.Email, accessToken)

	return &OutlookImapClient{
		CommonImapClient: imap.NewCommonImapClient(imapConfig, authProvider),
		credentials:      credentials,
	}
}
