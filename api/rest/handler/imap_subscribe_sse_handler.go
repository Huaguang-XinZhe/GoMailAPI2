package handler

import (
	"gomailapi2/api/rest/dto"
	"gomailapi2/internal/client/imap/outlook"
	"gomailapi2/internal/manager"
	"gomailapi2/internal/provider/token"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"
)

// 超时配置常量
const (
	ImapSSETimeoutMinutes        = 3  // IMAP SSE 连接超时时间（分钟）
	ImapHeartbeatIntervalSeconds = 60 // IMAP 心跳间隔（秒）
)

func HandleImapSubscribeSSE(tokenProvider *token.TokenProvider, imapManager *manager.ImapSubscriptionManager) gin.HandlerFunc {
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
			Msg("收到 IMAP 订阅请求")

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
			sendSSEError(c, err.Error())
			return
		}

		// 创建 IMAP 客户端
		imapClient := outlook.NewOutlookImapClient(&outlook.Credentials{
			Email:        request.NewMailInfo.Email,
			ClientID:     request.NewMailInfo.ClientID,
			RefreshToken: refreshToken,
		}, accessToken)

		// 创建新订阅
		subscription, err := imapManager.CreateSubscription(imapClient, request.NewMailInfo.Email)
		if err != nil {
			log.Error().Err(err).Msg("创建 IMAP 订阅失败")
			sendSSEError(c, err.Error())
			return
		}

		log.Info().
			Str("email", request.NewMailInfo.Email).
			Str("subscriptionID", subscription.ID).
			Msg("成功创建 IMAP 订阅")

		// 清理函数
		defer func() {
			imapManager.CancelSubscription(subscription.ID)
			log.Info().
				Str("subscriptionID", subscription.ID).
				Str("email", request.NewMailInfo.Email).
				Msg("清理 IMAP 订阅")
		}()

		// 启动订阅监听
		if err := imapManager.StartSubscription(subscription); err != nil {
			log.Error().Err(err).Msg("启动 IMAP 订阅监听失败")
			sendSSEError(c, err.Error())
			return
		}

		// 先发送订阅成功的消息
		subscriptionSuccess := gin.H{
			"message": "订阅成功",
		}

		// 只有在需要刷新且 refreshToken 不为空时才添加 refreshToken 字段
		if request.RefreshNeeded && refreshToken != "" {
			subscriptionSuccess["refreshToken"] = refreshToken
		}

		sendSSEEvent(c, "subscription", subscriptionSuccess)

		// 设置超时
		timeout := time.NewTimer(ImapSSETimeoutMinutes * time.Minute)
		defer timeout.Stop()

		// 定期发送心跳包
		heartbeat := time.NewTicker(ImapHeartbeatIntervalSeconds * time.Second)
		defer heartbeat.Stop()

		log.Info().
			Str("subscriptionID", subscription.ID).
			Str("email", request.NewMailInfo.Email).
			Msg("开始 SSE 等待新邮件")

		for { // 这里加 for 的话方便之后扩展连续收件（非必要）
			select {
			case email := <-subscription.EmailChan:
				if email != nil {
					log.Info().
						Str("subscriptionID", subscription.ID).
						Msg("通过 SSE 收到新邮件")

					// 发送邮件数据
					sendSSEEvent(c, "email", email)

					// 发送完成消息
					sendSSEEvent(c, "complete", gin.H{
						"message": "邮件推送完成",
					})
				}
				return

			case <-timeout.C:
				log.Info().
					Str("subscriptionID", subscription.ID).
					Str("email", request.NewMailInfo.Email).
					Msg("SSE 等待邮件超时")

				sendSSEEvent(c, "timeout", gin.H{
					"message": "等待邮件超时，订阅已过期",
				})
				return

			case <-c.Request.Context().Done():
				log.Info().
					Str("subscriptionID", subscription.ID).
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
