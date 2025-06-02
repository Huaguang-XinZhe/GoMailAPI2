package handler

import (
	"gomailapi2/api/rest/dto"
	"gomailapi2/internal/provider/token"
	"gomailapi2/internal/types"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// HandleRefreshToken 处理单个 Token 刷新请求
func HandleRefreshToken(tokenProvider *token.TokenProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 解析请求
		request, err := parseRefreshTokenRequest(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 验证 MailInfo
		if request.MailInfo == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "MailInfo 不能为空"})
			return
		}

		log.Info().
			Str("email", request.MailInfo.Email).
			Str("provider", string(request.MailInfo.ServiceProvider)).
			Msg("收到刷新 Token 请求")

		// 刷新 Token
		newRefreshToken, err := tokenProvider.GetRefreshToken(request.MailInfo)
		if err != nil {
			log.Error().Err(err).Str("email", request.MailInfo.Email).Msg("刷新 Token 失败")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		log.Info().Str("email", request.MailInfo.Email).Msg("成功刷新 Token")
		c.JSON(http.StatusOK, gin.H{
			"newRefreshToken": newRefreshToken,
		})
	}
}

// HandleBatchRefreshToken 处理批量 Token 刷新请求（并发处理，限制每次最多 100 个）
func HandleBatchRefreshToken(tokenProvider *token.TokenProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 解析请求
		request, err := parseBatchRefreshTokenRequest(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 验证请求
		if len(request.MailInfos) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Token 列表不能为空"})
			return
		}

		// 限制每次最多处理 100 个
		const maxBatchSize = 100
		mailInfoCount := len(request.MailInfos)
		if mailInfoCount > maxBatchSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "每次最多只能处理 100 个 Token"})
			return
		}

		log.Info().
			Int("count", mailInfoCount).
			Msg("收到批量刷新 Token 请求")

		// 使用 WaitGroup 等待所有 goroutine 完成
		var wg sync.WaitGroup
		// 使用 Mutex 保护共享变量
		var mu sync.Mutex
		var results []dto.BatchRefreshResult
		var successCount, failCount int

		// 并发处理每个 Token
		for _, mailInfo := range request.MailInfos {
			wg.Add(1)

			// 启动 goroutine 处理单个 Token
			go func(mailInfo *types.MailInfo) {
				defer wg.Done()

				result := dto.BatchRefreshResult{
					Email: mailInfo.Email,
				}

				// 刷新 Token
				newRefreshToken, err := tokenProvider.GetRefreshToken(mailInfo)
				if err != nil {
					log.Error().Err(err).Str("email", mailInfo.Email).Msg("批量刷新 Token 失败")
					errorMsg := err.Error()
					result.Error = errorMsg

					// 线程安全地更新失败计数
					mu.Lock()
					failCount++
					mu.Unlock()
				} else {
					log.Info().Str("email", mailInfo.Email).Msg("刷新 Token 成功")
					result.NewRefreshToken = newRefreshToken

					// 线程安全地更新成功计数
					mu.Lock()
					successCount++
					mu.Unlock()
				}

				// 线程安全地添加结果
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}(mailInfo)
		}

		// 等待所有 goroutine 完成
		wg.Wait()

		log.Info().
			Int("successCount", successCount).
			Int("failCount", failCount).
			Msg("批量刷新 Token 完成")

		// 构建响应数据
		responseData := dto.BatchRefreshTokenData{
			SuccessCount: successCount,
			FailCount:    failCount,
			Results:      results,
		}

		c.JSON(http.StatusOK, responseData)
	}
}

// parseRefreshTokenRequest 解析刷新 Token 请求
func parseRefreshTokenRequest(c *gin.Context) (*dto.RefreshTokenRequest, error) {
	var request dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Msg("解析刷新 Token 请求失败")
		return nil, err
	}
	return &request, nil
}

// parseBatchRefreshTokenRequest 解析批量刷新 Token 请求
func parseBatchRefreshTokenRequest(c *gin.Context) (*dto.BatchRefreshTokenRequest, error) {
	var request dto.BatchRefreshTokenRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Msg("解析批量刷新 Token 请求失败")
		return nil, err
	}
	return &request, nil
}
