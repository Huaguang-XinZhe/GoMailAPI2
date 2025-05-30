package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"gomailapi2/api/rest/dto"
	"gomailapi2/internal/client/graph"
	"gomailapi2/internal/notification"
	"gomailapi2/internal/provider/token"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// todo IP 自动化设置（初始化时设置，作为全局变量）
var notificationURL = "https://cd2c-2408-8948-2001-47a9-bd66-1a7e-8f31-8d34.ngrok-free.app/api/v1/graph/webhook"

// 超时配置常量
const (
	SSETimeoutMinutes          = 3  // SSE 连接超时时间（分钟）
	SubscriptionTimeoutMinutes = 5  // 订阅过期时间（分钟）
	HeartbeatIntervalSeconds   = 60 // 心跳间隔（秒）
)

// HandleGraphSubscribeSSE 使用 Server-Sent Events 的订阅处理器
func HandleGraphSubscribeSSE(tokenProvider *token.TokenProvider, nfManager *notification.NotificationManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request dto.SubscribeMailRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			log.Error().Err(err).Msg("解析订阅请求失败")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Info().
			Str("email", request.NewMailInfo.Email).
			Str("prevSubscribeID", request.PrevSubScribeID).
			Bool("refreshNeeded", request.RefreshNeeded).
			Msg("收到 SSE Graph 订阅请求")

		// 设置 SSE headers
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		// 获取 token
		var accessToken, refreshToken string
		var err error

		if request.RefreshNeeded {
			accessToken, refreshToken, err = tokenProvider.GetBothTokens(request.NewMailInfo)
		} else {
			accessToken, err = tokenProvider.GetAccessToken(request.NewMailInfo)
		}

		if err != nil {
			log.Error().Err(err).Msg("获取 token 失败")
			sendSSEError(c, "获取 token 失败: "+err.Error())
			return
		}

		// 删除旧订阅（如果存在）
		if request.PrevSubScribeID != "" {
			if err := graph.DeleteSubscription(context.Background(), accessToken, request.PrevSubScribeID); err != nil {
				log.Warn().Err(err).Str("subscriptionID", request.PrevSubScribeID).Msg("删除旧订阅失败")
			}
		}

		// 创建新订阅
		response, err := graph.CreateSubscription(context.Background(), accessToken, notificationURL)
		if err != nil {
			log.Error().Err(err).Str("email", request.NewMailInfo.Email).Msg("创建订阅失败")
			sendSSEError(c, "创建订阅失败: "+err.Error())
			return
		}

		log.Info().
			Str("email", request.NewMailInfo.Email).
			Str("subscriptionID", response.ID).
			Int64("createdAt", time.Now().Unix()).
			Msg("成功创建 Graph 订阅")

		// 先发送订阅成功的消息
		subscriptionSuccess := make(map[string]any)
		subscriptionSuccess["subscriptionID"] = response.ID

		// 只有在需要刷新且 refreshToken 不为空时才添加 refreshToken 字段
		if request.RefreshNeeded && refreshToken != "" {
			subscriptionSuccess["refreshToken"] = refreshToken
		}

		sendSSEEvent(c, "subscription", subscriptionSuccess)

		// 注册邮件通知通道
		notifyChan := nfManager.RegisterChannel(response.ID)

		// 清理函数
		defer func() {
			nfManager.RemoveChannel(response.ID)
			log.Info().
				Str("subscriptionID", response.ID).
				Str("email", request.NewMailInfo.Email).
				Msg("清理订阅通知通道")
		}()

		// 设置超时（3分钟）
		timeout := time.NewTimer(SSETimeoutMinutes * time.Minute)
		defer timeout.Stop()

		// 定期发送心跳包
		heartbeat := time.NewTicker(HeartbeatIntervalSeconds * time.Second) // 使用配置常量
		defer heartbeat.Stop()

		log.Info().
			Str("subscriptionID", response.ID).
			Str("email", request.NewMailInfo.Email).
			Msg("开始 SSE 等待新邮件通知")

		for {
			select {
			case emailID := <-notifyChan:
				log.Info().
					Str("subscriptionID", response.ID).
					Msg("通过 SSE 收到新邮件通知")

				email, err := graph.GetEmailByID(context.Background(), accessToken, emailID)
				if err != nil {
					log.Error().Err(err).Msg("获取邮件详情失败")
					continue
				}

				// 发送邮件数据，直接使用 email 对象
				sendSSEEvent(c, "email", email)

				// 发送完成消息
				completeMessage := map[string]any{
					"message": "邮件推送完成",
				}
				sendSSEEvent(c, "complete", completeMessage)
				return

			case <-timeout.C:
				log.Info().
					Str("subscriptionID", response.ID).
					Str("email", request.NewMailInfo.Email).
					Msg("SSE 等待邮件超时")

				timeoutMessage := map[string]any{
					"message": "等待邮件超时，订阅已过期",
				}
				sendSSEEvent(c, "timeout", timeoutMessage)
				return

			case <-c.Request.Context().Done():
				log.Info().
					Str("subscriptionID", response.ID).
					Str("email", request.NewMailInfo.Email).
					Msg("SSE 客户端连接断开")
				return

			case <-heartbeat.C:
				// 发送心跳包保持连接活跃
				heartbeatMessage := map[string]any{
					"timestamp": time.Now().Unix(),
				}
				sendSSEEvent(c, "heartbeat", heartbeatMessage)
			}
		}
	}
}

// sendSSEEvent 发送 SSE 事件（data 为 json 格式）
func sendSSEEvent(c *gin.Context, event string, data any) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, string(jsonData))
	c.Writer.Flush()
}

// sendSSEError 发送 SSE 错误消息
func sendSSEError(c *gin.Context, message string) {
	errorData := map[string]any{
		"type":    "error",
		"message": message,
	}
	sendSSEEvent(c, "error", errorData)
}
