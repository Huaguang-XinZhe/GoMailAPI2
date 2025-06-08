package service

import (
	"fmt"
	"strings"
	"time"

	"gomailapi2/api/rest/dto"
	"gomailapi2/internal/cache/tokencache"
	"gomailapi2/internal/origin/auth"
	"gomailapi2/internal/types"

	"github.com/rs/zerolog/log"
)

// ProtocolDetectResult 协议检测结果
type ProtocolDetectResult struct {
	ProtoType types.ProtocolType `json:"protoType"` // 检测到的协议类型
}

// ProtocolService 协议检测服务
type ProtocolService struct {
	cache tokencache.Cache
}

// NewProtocolService 创建新的协议检测服务
func NewProtocolService(cache tokencache.Cache) *ProtocolService {
	return &ProtocolService{
		cache: cache,
	}
}

// DetectProtocolType 检测邮件协议类型
func (s *ProtocolService) DetectProtocolType(mailInfo *types.MailInfo) (*ProtocolDetectResult, error) {
	// 验证必填字段
	// todo 可空，为空就不打印邮箱
	if mailInfo.Email == "" {
		return nil, fmt.Errorf("email 不能为空")
	}
	// todo 可空，为空默认雷鸟 clientId
	if mailInfo.ClientID == "" {
		return nil, fmt.Errorf("clientId 不能为空")
	}
	if mailInfo.RefreshToken == "" {
		return nil, fmt.Errorf("refreshToken 不能为空")
	}
	// todo 可空，为空自助判断
	if mailInfo.ServiceProvider == "" {
		return nil, fmt.Errorf("serviceProvider 不能为空")
	}

	log.Info().
		Str("email", mailInfo.Email).
		Str("provider", string(mailInfo.ServiceProvider)).
		Msg("开始检测邮件协议类型")

	// 调用带 scope 的 GetTokensWithScope 方法
	tokenResp, err := auth.GetTokensWithScope(mailInfo, true)
	if err != nil {
		log.Error().Err(err).Str("email", mailInfo.Email).Msg("获取 token 失败")
		return nil, fmt.Errorf("获取 token 失败: %w", err)
	}

	// 根据 scope 判断协议类型
	var detectedType types.ProtocolType
	if tokenResp.Scope != "" && s.isGraphScope(tokenResp.Scope) {
		// Graph 协议：accessToken 有效，缓存起来
		detectedType = types.ProtocolTypeGraph

		if tokenResp.AccessToken != "" {
			// 缓存 accessToken，假设有效期 50 分钟，留 10 分钟缓冲
			if err := s.cache.SetAccessToken(mailInfo.RefreshToken, tokenResp.AccessToken, 50*time.Minute); err != nil {
				log.Warn().
					Err(err).
					Str("email", mailInfo.Email).
					Msg("缓存 Graph accessToken 失败，但不影响返回结果")
			} else {
				log.Info().
					Str("email", mailInfo.Email).
					Msg("成功缓存 Graph accessToken")
			}
		}

		log.Info().
			Str("email", mailInfo.Email).
			Str("detectedType", string(types.ProtocolTypeGraph)).
			Str("scope", tokenResp.Scope).
			Msg("检测到 Graph 协议")
	} else {
		// IMAP 协议：此 token 对 Graph API 无效，舍弃
		detectedType = types.ProtocolTypeIMAP

		log.Info().
			Str("email", mailInfo.Email).
			Str("detectedType", string(types.ProtocolTypeIMAP)).
			Str("scope", tokenResp.Scope).
			Msg("检测到 IMAP 协议")
	}

	result := &ProtocolDetectResult{
		ProtoType: detectedType,
	}

	log.Info().
		Str("email", mailInfo.Email).
		Str("detectedType", string(detectedType)).
		Bool("accessTokenCached", detectedType == types.ProtocolTypeGraph).
		Msg("协议类型检测完成")

	return result, nil
}

// isGraphScope 检查 scope 是否包含 Graph API 权限
func (s *ProtocolService) isGraphScope(scope string) bool {
	return strings.Contains(scope, "https://graph.microsoft.com/Mail.ReadWrite")
}

// BatchDetectProtocolType 批量检测邮件协议类型（并发处理）
func (s *ProtocolService) BatchDetectProtocolType(mailInfos []*types.MailInfo) (*dto.BatchDetectProtocolTypeResponse, error) {
	if len(mailInfos) == 0 {
		return &dto.BatchDetectProtocolTypeResponse{
			SuccessCount: 0,
			FailCount:    0,
			Results:      []dto.BatchDetectProtocolTypeResult{},
		}, nil
	}

	log.Info().
		Int("count", len(mailInfos)).
		Msg("开始批量检测邮件协议类型")

	// 创建结果通道
	resultChan := make(chan dto.BatchDetectProtocolTypeResult, len(mailInfos))

	// 使用 goroutine 并发处理每个邮箱
	for _, mailInfo := range mailInfos {
		go func(mi *types.MailInfo) {
			result := dto.BatchDetectProtocolTypeResult{
				Email: mi.Email,
			}

			// 检测协议类型
			detectResult, err := s.DetectProtocolType(mi)
			if err != nil {
				result.Error = err.Error()
				log.Error().
					Err(err).
					Str("email", mi.Email).
					Msg("批量检测中单个邮箱协议类型检测失败")
			} else {
				result.ProtoType = detectResult.ProtoType
				log.Info().
					Str("email", mi.Email).
					Str("detectedType", string(detectResult.ProtoType)).
					Msg("批量检测中单个邮箱协议类型检测成功")
			}

			resultChan <- result
		}(mailInfo)
	}

	// 收集所有结果
	var results []dto.BatchDetectProtocolTypeResult
	successCount := 0
	failCount := 0

	for range len(mailInfos) {
		result := <-resultChan
		results = append(results, result)

		if result.Error == "" {
			successCount++
		} else {
			failCount++
		}
	}

	close(resultChan)

	batchResult := &dto.BatchDetectProtocolTypeResponse{
		SuccessCount: successCount,
		FailCount:    failCount,
		Results:      results,
	}

	log.Info().
		Int("total", len(mailInfos)).
		Int("success", successCount).
		Int("fail", failCount).
		Msg("批量检测邮件协议类型完成")

	return batchResult, nil
}
