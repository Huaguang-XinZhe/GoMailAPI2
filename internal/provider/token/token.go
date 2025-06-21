package token

import (
	"fmt"
	"time"

	"gomailapi2/internal/cache/tokencache"
	"gomailapi2/internal/origin/auth"
	"gomailapi2/internal/types"

	"github.com/rs/zerolog/log"
)

// TokenProvider 协调缓存和原始数据层的 token 提供者
type TokenProvider struct {
	cache tokencache.Cache
}

// NewTokenProvider 创建新的 TokenProvider 实例
func NewTokenProvider(cache tokencache.Cache) *TokenProvider {
	return &TokenProvider{
		cache: cache,
	}
}

// GetAccessToken 获取 access token（优先从缓存获取）
func (p *TokenProvider) GetAccessToken(mailInfo *types.MailInfo) (string, error) {
	log.Debug().
		Str("email", mailInfo.Email).
		Str("protocol", string(mailInfo.ProtocolType)).
		Msg("开始获取 access token")

	// 1. 尝试从缓存获取
	if token, err := p.cache.GetAccessToken(mailInfo.RefreshToken); err == nil && token != "" {
		log.Debug().
			Str("email", mailInfo.Email).
			Msg("从缓存获取到 access token")
		return token, nil
	}

	// 2. 缓存未命中，调用原始数据层
	log.Info().
		Str("email", mailInfo.Email).
		Msg("缓存中没有 access token，正在从原始数据层获取")

	token, err := auth.GetAccessToken(mailInfo)
	if err != nil {
		return "", fmt.Errorf("从原始数据层获取 access token 失败: %w", err)
	}

	// 3. 将结果写入缓存（假设 token 有效期 50 分钟，留 10 分钟缓冲）
	// todo 过期时间配置化
	if err := p.cache.SetAccessToken(mailInfo.RefreshToken, token, 50*time.Minute); err != nil {
		log.Warn().
			Err(err).
			Str("email", mailInfo.Email).
			Msg("写入缓存失败，但不影响返回结果")
	}

	log.Info().
		Str("email", mailInfo.Email).
		Msg("成功获取并缓存 access token")

	return token, nil
}

// GetRefreshToken 获取 refresh token（直接调用原始数据层）
func (p *TokenProvider) GetRefreshToken(mailInfo *types.MailInfo) (string, error) {
	log.Debug().
		Str("email", mailInfo.Email).
		Msg("开始获取 refresh token")

	// refresh token 通常不缓存，直接调用原始数据层
	token, err := auth.GetRefreshToken(mailInfo)
	if err != nil {
		return "", fmt.Errorf("获取 refresh token 失败: %w", err)
	}

	// // 获取新的 refresh token 后，清除旧的 access token 缓存
	// // ! 这个我倒是没想到
	// if err := p.cache.DeleteAccessToken(mailInfo.RefreshToken); err != nil {
	// 	log.Warn().
	// 		Err(err).
	// 		Str("email", mailInfo.Email).
	// 		Msg("清除旧 access token 缓存失败")
	// }

	log.Info().
		Str("email", mailInfo.Email).
		Msg("成功获取 refresh token")

	return token, nil
}

// GetBothTokens 同时获取 access token 和 refresh token（直接发网络请求）
func (p *TokenProvider) GetBothTokens(mailInfo *types.MailInfo) (string, string, error) {
	log.Debug().
		Str("email", mailInfo.Email).
		Str("protocol", string(mailInfo.ProtocolType)).
		Msg("开始同时获取 access token 和 refresh token")

	// 直接调用原始数据层获取两个 token
	accessToken, refreshToken, err := auth.GetBothTokens(mailInfo)
	if err != nil {
		return "", "", fmt.Errorf("同时获取两个 token 失败: %w", err)
	}

	// 缓存新的 access token
	if err := p.cache.SetAccessToken(mailInfo.RefreshToken, accessToken, 50*time.Minute); err != nil {
		log.Warn().
			Err(err).
			Str("email", mailInfo.Email).
			Msg("缓存新 access token 失败")
	}

	log.Info().
		Str("email", mailInfo.Email).
		Bool("hasNewRefreshToken", refreshToken != "").
		Msg("成功同时获取两个 token")

	return accessToken, refreshToken, nil
}

// Close 关闭 TokenProvider，释放资源
func (p *TokenProvider) Close() error {
	if p.cache != nil {
		return p.cache.Close()
	}
	return nil
}
