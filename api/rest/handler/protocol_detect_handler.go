package handler

import (
	"gomailapi2/api/rest/dto"
	"gomailapi2/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// HandleDetectProtocolType 处理协议类型检测请求
func HandleDetectProtocolType(protocolService *service.ProtocolService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 解析请求
		request, err := parseDetectProtocolTypeRequest(c)
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
			Msg("收到协议类型检测请求")

		// 检测协议类型
		result, err := protocolService.DetectProtocolType(request.MailInfo)
		if err != nil {
			log.Error().Err(err).Str("email", request.MailInfo.Email).Msg("协议类型检测失败")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 构建响应
		response := dto.DetectProtocolTypeResponse{
			ProtocolType: result.ProtocolType,
		}

		log.Info().
			Str("email", request.MailInfo.Email).
			Str("detectedType", string(result.ProtocolType)).
			Msg("协议类型检测成功")

		c.JSON(http.StatusOK, response)
	}
}

// HandleBatchDetectProtocolType 处理批量协议类型检测请求
func HandleBatchDetectProtocolType(protocolService *service.ProtocolService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 解析请求
		request, err := parseBatchDetectProtocolTypeRequest(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 验证 MailInfos
		if request.MailInfos == nil || len(request.MailInfos) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "MailInfos 不能为空"})
			return
		}

		log.Info().
			Int("count", len(request.MailInfos)).
			Msg("收到批量协议类型检测请求")

		// 批量检测协议类型（并发处理）
		result, err := protocolService.BatchDetectProtocolType(request.MailInfos)
		if err != nil {
			log.Error().Err(err).Int("count", len(request.MailInfos)).Msg("批量协议类型检测失败")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		log.Info().
			Int("total", len(request.MailInfos)).
			Int("success", result.SuccessCount).
			Int("fail", result.FailCount).
			Msg("批量协议类型检测完成")

		c.JSON(http.StatusOK, result)
	}
}

// parseDetectProtocolTypeRequest 解析协议类型检测请求
func parseDetectProtocolTypeRequest(c *gin.Context) (*dto.DetectProtocolTypeRequest, error) {
	var request dto.DetectProtocolTypeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Msg("解析协议类型检测请求失败")
		return nil, err
	}
	return &request, nil
}

// parseBatchDetectProtocolTypeRequest 解析批量协议类型检测请求
func parseBatchDetectProtocolTypeRequest(c *gin.Context) (*dto.BatchDetectProtocolTypeRequest, error) {
	var request dto.BatchDetectProtocolTypeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Msg("解析批量协议类型检测请求失败")
		return nil, err
	}
	return &request, nil
}
