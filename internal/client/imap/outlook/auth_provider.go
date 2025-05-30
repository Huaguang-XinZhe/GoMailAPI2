package outlook

import (
	"gomailapi2/internal/utils"

	"github.com/emersion/go-sasl"
)

// OutlookAuthProvider 实现 AuthProvider 接口
type OutlookAuthProvider struct {
	email       string
	accessToken string
}

func NewOutlookAuthProvider(email string, accessToken string) *OutlookAuthProvider {
	return &OutlookAuthProvider{
		email:       email,
		accessToken: accessToken,
	}
}

func (a *OutlookAuthProvider) GetSASLClient() (sasl.Client, error) {
	return utils.NewXOAuth2Client(a.email, a.accessToken), nil
}
