package utils

import (
	"fmt"

	"github.com/emersion/go-sasl"
)

// XOAuth2Client 实现 sasl.Client 接口
type XOAuth2Client struct {
	Username string // 用户名（邮箱地址）
	Token    string // OAuth2 令牌
	Initial  []byte // 初始响应
	Done     bool   // 是否完成认证
}

// Start 开始认证过程，返回认证机制名称、初始响应和可能的错误
func (a *XOAuth2Client) Start() (mech string, ir []byte, err error) {
	a.Initial = fmt.Appendf(nil, "user=%s\x01auth=Bearer %s\x01\x01", a.Username, a.Token)
	return "XOAUTH2", a.Initial, nil
}

// Next 处理服务器的挑战，返回响应和可能的错误
func (a *XOAuth2Client) Next(challenge []byte) (response []byte, err error) {
	if a.Done {
		return nil, sasl.ErrUnexpectedServerChallenge
	}
	a.Done = true
	return []byte{}, nil
}

// NewXOAuth2Client 创建 XOAuth2 客户端（username 为邮箱地址，token 为 access token）
func NewXOAuth2Client(username, token string) sasl.Client {
	return &XOAuth2Client{Username: username, Token: token}
}
