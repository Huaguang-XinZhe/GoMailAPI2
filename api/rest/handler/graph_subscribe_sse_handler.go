package handler

import (
	"context"
	"gomailapi2/api/rest/dto"
	"gomailapi2/internal/client/graph"
	"gomailapi2/internal/notification"
	"gomailapi2/internal/provider/token"
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
		// 设置 SSE headers
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		var request dto.SubscribeMailRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			log.Error().Err(err).Msg("解析订阅请求失败")
			sendSSEError(c, err.Error())
			return
		}

		log.Info().
			Str("email", request.NewMailInfo.Email).
			Bool("refreshNeeded", request.RefreshNeeded).
			Msg("收到 SSE Graph 订阅请求")

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

		// 创建新订阅
		response, err := graph.CreateSubscription(context.Background(), accessToken, notificationURL)
		if err != nil {
			log.Error().Err(err).Str("email", request.NewMailInfo.Email).Msg("创建订阅失败")
			sendSSEError(c, "创建订阅失败: "+err.Error())
			return
		}

		// 结束后，自动取消当前订阅
		defer func() {
			if err := graph.DeleteSubscription(context.Background(), accessToken, response.ID); err != nil {
				log.Warn().Err(err).Str("subscriptionID", response.ID).Msg("删除订阅失败")
			}
		}()

		log.Info().
			Str("email", request.NewMailInfo.Email).
			Str("subscriptionID", response.ID).
			Msg("成功创建 Graph 订阅")

		// 先发送订阅成功的消息
		subscriptionSuccess := gin.H{
			"message": "订阅成功",
		}

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
				sendSSEEvent(c, "complete", gin.H{
					"message": "邮件推送完成",
				})
				return

			case <-timeout.C:
				log.Info().
					Str("subscriptionID", response.ID).
					Str("email", request.NewMailInfo.Email).
					Msg("SSE 等待邮件超时")

				sendSSEEvent(c, "timeout", gin.H{
					"message": "等待邮件超时，订阅已过期",
				})
				return

			case <-c.Request.Context().Done():
				log.Info().
					Str("subscriptionID", response.ID).
					Str("email", request.NewMailInfo.Email).
					Msg("SSE 客户端连接断开")
				return

			case <-heartbeat.C:
				// 发送心跳包保持连接活跃
				sendSSEEvent(c, "heartbeat", gin.H{
					"timestamp": time.Now().Unix(),
				})
			}
		}
	}
}
