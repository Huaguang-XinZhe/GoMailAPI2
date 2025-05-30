package handler

import (
	"encoding/json"
	"fmt"
	"gomailapi2/api/rest/dto"
	"gomailapi2/internal/provider/token"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// SSE 订阅配置常量
const (
	SSETimeoutMinutes           = 3  // SSE 连接超时时间（分钟）
	SSEHeartbeatIntervalSeconds = 60 // SSE 心跳间隔（秒）
)

// sendSSEEvent 发送 SSE 事件（data 为 json 格式）
func sendSSEEvent(c *gin.Context, event string, data any) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, string(jsonData))
	c.Writer.Flush()
}

// sendSSEError 发送 SSE 错误消息
func sendSSEError(c *gin.Context, message string) {
	sendSSEEvent(c, "error", gin.H{
		"message": message,
	})
}

// setupSSEHeaders 设置 SSE 响应头
func setupSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
}

// parseSubscribeRequest 解析订阅请求
func parseSubscribeRequest(c *gin.Context) (*dto.SubscribeMailRequest, error) {
	var request dto.SubscribeMailRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Msg("解析订阅请求失败")
		sendSSEError(c, err.Error())
		return nil, err
	}
	return &request, nil
}

// getTokens 获取访问令牌和刷新令牌
func getTokens(tokenProvider *token.TokenProvider, request *dto.SubscribeMailRequest) (string, string, error) {
	var accessToken, refreshToken string
	var err error

	if request.RefreshNeeded {
		accessToken, refreshToken, err = tokenProvider.GetBothTokens(request.MailInfo)
	} else {
		accessToken, err = tokenProvider.GetAccessToken(request.MailInfo)
	}

	return accessToken, refreshToken, err
}

// sendSubscriptionSuccess 发送订阅成功消息
func sendSubscriptionSuccess(c *gin.Context, refreshNeeded bool, refreshToken string) {
	subscriptionSuccess := gin.H{
		"message": "订阅成功",
	}

	// 只有在需要刷新且 refreshToken 不为空时才添加 refreshToken 字段
	if refreshNeeded && refreshToken != "" {
		subscriptionSuccess["refreshToken"] = refreshToken
	}

	sendSSEEvent(c, "subscription", subscriptionSuccess)
}

// createHeartbeatTicker 创建心跳定时器
func createHeartbeatTicker() *time.Ticker {
	return time.NewTicker(SSEHeartbeatIntervalSeconds * time.Second)
}

// createSSETimeout 创建 SSE 超时定时器
func createSSETimeout() *time.Timer {
	return time.NewTimer(SSETimeoutMinutes * time.Minute)
}
