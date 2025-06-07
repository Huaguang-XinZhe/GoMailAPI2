package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"gomailapi2/internal/types"

	"github.com/rs/zerolog/log"
)

// TokenResponse Token 响应
// ! 调试发现，json 的 key 写错了，解析有问题，但却没报错！请求是成功的，字段值为空。
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"` // 避免 refreshToken 为空值时返回
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

// GetAccessToken 获取 accessToken
func GetAccessToken(mailInfo *types.MailInfo) (string, error) {
	switch mailInfo.ProtoType {
	case types.ProtocolTypeIMAP:
		// IMAP: accessToken 过期 -> GetTokensWithScope(includeScope=false)，取 accessToken
		tokenResp, err := GetTokensWithScope(mailInfo, false)
		if err != nil {
			return "", err
		}
		return tokenResp.AccessToken, nil

	case types.ProtocolTypeGraph:
		// Graph: accessToken 过期 -> GetTokensWithScope(includeScope=true)，取 accessToken
		tokenResp, err := GetTokensWithScope(mailInfo, true)
		if err != nil {
			return "", err
		}
		return tokenResp.AccessToken, nil

	default:
		return "", fmt.Errorf("不支持的协议类型: %s", mailInfo.ProtoType)
	}
}

// GetRefreshToken 获取 refreshToken
func GetRefreshToken(mailInfo *types.MailInfo) (string, error) {
	// 不论 IMAP 还是 Graph，只要刷新都是 GetTokensWithScope(includeScope=false)，取 refreshToken
	tokenResp, err := GetTokensWithScope(mailInfo, false)
	if err != nil {
		return "", err
	}
	return tokenResp.RefreshToken, nil
}

// GetBothTokens 同时获取 accessToken 和 refreshToken
func GetBothTokens(mailInfo *types.MailInfo) (string, string, error) {
	switch mailInfo.ProtoType {
	case types.ProtocolTypeIMAP:
		// IMAP: 要求刷新，同时获取新邮件/监听 -> 同时获取 accessToken 和 refreshToken（GetTokensWithScope(includeScope=false)）
		tokenResp, err := GetTokensWithScope(mailInfo, false)
		if err != nil {
			return "", "", err
		}
		return tokenResp.AccessToken, tokenResp.RefreshToken, nil

	case types.ProtocolTypeGraph:
		// Graph: 要求刷新，同时获取新邮件/监听 -> 并发获取 accessToken 和 refreshToken
		return getBothTokensConcurrently(mailInfo)

	default:
		return "", "", fmt.Errorf("不支持的协议类型: %s", mailInfo.ProtoType)
	}
}

// getBothTokensConcurrently 并发获取 accessToken 和 refreshToken（仅用于 Graph 协议）
func getBothTokensConcurrently(mailInfo *types.MailInfo) (string, string, error) {
	type tokenResult struct {
		token string
		err   error
	}

	// 创建两个 channel 来接收结果
	accessTokenCh := make(chan tokenResult, 1)
	refreshTokenCh := make(chan tokenResult, 1)

	// 并发获取 accessToken（带 scope）
	go func() {
		token, err := GetTokensWithScope(mailInfo, true)
		accessTokenCh <- tokenResult{token: token.AccessToken, err: err}
	}()

	// 并发获取 refreshToken（不带 scope）
	go func() {
		token, err := GetTokensWithScope(mailInfo, false)
		refreshTokenCh <- tokenResult{token: token.RefreshToken, err: err}
	}()

	// 等待两个请求完成
	accessResult := <-accessTokenCh
	refreshResult := <-refreshTokenCh

	// 检查错误
	if accessResult.err != nil {
		return "", "", fmt.Errorf("获取 accessToken 失败: %w", accessResult.err)
	}
	if refreshResult.err != nil {
		return "", "", fmt.Errorf("获取 refreshToken 失败: %w", refreshResult.err)
	}

	// 合并结果
	return accessResult.token, refreshResult.token, nil
}

// GetTokensWithScope 带 scope 返回 graph api 所需的 accessToken，不带，返回 refreshToken 和 IMAP API 所需的 accessToken（Graph API 无法使用）
func GetTokensWithScope(mailInfo *types.MailInfo, includeScope bool) (*TokenResponse, error) {
	// 目前只处理微软邮箱
	if mailInfo.ServiceProvider != types.ServiceProviderMicrosoft {
		return nil, fmt.Errorf("暂时只支持微软邮箱，不支持: %s", mailInfo.ServiceProvider)
	}

	// 微软的 token endpoint
	tokenURL := "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"

	// 构建请求参数
	data := url.Values{}
	data.Set("client_id", mailInfo.ClientID)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", mailInfo.RefreshToken)

	// 根据 includeScope 参数决定是否添加 scope
	if includeScope {
		data.Set("scope", "https://graph.microsoft.com/.default")
	}

	// // 创建请求
	// req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	// if err != nil {
	// 	return nil, fmt.Errorf("创建请求失败: %w", err)
	// }
	// req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 发送请求
	// resp, err := client.NewProxyClient().Do(req)
	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("获取令牌失败 (状态码: %d): %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	log.Info().
		Str("email", mailInfo.Email).
		Bool("hasRefreshToken", tokenResp.RefreshToken != "").
		Int64("expiresIn", tokenResp.ExpiresIn).
		Msg("成功获取 token")

	return &tokenResp, nil
}
