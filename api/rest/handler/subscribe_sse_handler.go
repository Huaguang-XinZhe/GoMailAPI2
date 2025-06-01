package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"gomailapi2/api/rest/dto"
	"gomailapi2/internal/client/graph"
	"gomailapi2/internal/client/imap/outlook"
	"gomailapi2/internal/manager"
	"gomailapi2/internal/provider/token"
	"gomailapi2/internal/types"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// SSE 订阅配置常量
const (
	SSETimeoutMinutes           = 3  // SSE 连接超时时间（分钟）
	SSEHeartbeatIntervalSeconds = 60 // SSE 心跳间隔（秒）
	// todo IP 自动化设置（初始化时设置，作为全局变量）
	graphNotificationURL = "https://cd2c-2408-8948-2001-47a9-bd66-1a7e-8f31-8d34.ngrok-free.app/api/v1/graph/webhook"
)

// HandleUnifiedSubscribeSSE 统一的邮件订阅 SSE 处理器，支持 IMAP 和 Graph 协议
func HandleUnifiedSubscribeSSE(
	tokenProvider *token.TokenProvider,
	nfManager *manager.NotificationManager,
	imapManager *manager.ImapSubscriptionManager,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置 SSE headers
		setupSSEHeaders(c)

		// 解析请求
		request, err := parseSubscribeRequest(c)
		if err != nil {
			return
		}

		log.Info().
			Str("email", request.MailInfo.Email).
			Str("protocol", string(request.MailInfo.ProtoType)).
			Bool("refreshNeeded", request.RefreshNeeded).
			Msg("收到统一订阅请求")

		// 获取 token
		accessToken, refreshToken, err := getTokens(tokenProvider, request.RefreshNeeded, request.MailInfo)
		if err != nil {
			log.Error().Err(err).Msg("获取 token 失败")
			sendSSEError(c, err.Error())
			return
		}

		// 根据协议类型选择处理方式
		switch request.MailInfo.ProtoType {
		case types.ProtocolTypeIMAP:
			handleImapSubscription(c, request, accessToken, refreshToken, imapManager)
		case types.ProtocolTypeGraph:
			handleGraphSubscription(c, request, accessToken, refreshToken, nfManager)
		default:
			log.Error().
				Str("protocol", string(request.MailInfo.ProtoType)).
				Msg("不支持的协议类型")
			sendSSEError(c, "不支持的协议类型: "+string(request.MailInfo.ProtoType))
		}
	}
}

// handleImapSubscription 处理 IMAP 协议订阅
func handleImapSubscription(
	c *gin.Context,
	request *dto.SubscribeMailRequest,
	accessToken, refreshToken string,
	imapManager *manager.ImapSubscriptionManager,
) {
	// 创建 IMAP 客户端
	imapClient := outlook.NewOutlookImapClient(mailInfoToCredentials(request.MailInfo), accessToken)

	// 创建新订阅
	subscription, err := imapManager.CreateSubscription(imapClient, request.MailInfo.Email)
	if err != nil {
		log.Error().Err(err).Msg("创建 IMAP 订阅失败")
		sendSSEError(c, err.Error())
		return
	}

	log.Info().
		Str("email", request.MailInfo.Email).
		Str("subscriptionID", subscription.ID).
		Msg("成功创建 IMAP 订阅")

	// 清理函数
	defer func() {
		imapManager.CancelSubscription(subscription.ID)
		log.Info().
			Str("subscriptionID", subscription.ID).
			Str("email", request.MailInfo.Email).
			Msg("清理 IMAP 订阅")
	}()

	// 启动订阅监听
	if err := imapManager.StartSubscription(subscription); err != nil {
		log.Error().Err(err).Msg("启动 IMAP 订阅监听失败")
		sendSSEError(c, err.Error())
		return
	}

	// 发送订阅成功消息
	sendSubscriptionSuccess(c, request.RefreshNeeded, refreshToken)

	// 开始监听 IMAP 邮件
	listenForImapEmails(c, subscription, request.MailInfo.Email)
}

// handleGraphSubscription 处理 Graph API 协议订阅
func handleGraphSubscription(
	c *gin.Context,
	request *dto.SubscribeMailRequest,
	accessToken, refreshToken string,
	nfManager *manager.NotificationManager,
) {
	// 创建新订阅
	response, err := graph.CreateSubscription(context.Background(), accessToken, graphNotificationURL)
	if err != nil {
		log.Error().Err(err).Str("email", request.MailInfo.Email).Msg("创建 Graph 订阅失败")
		sendSSEError(c, "创建订阅失败: "+err.Error())
		return
	}

	// 结束后，自动取消当前订阅
	defer func() {
		if err := graph.DeleteSubscription(context.Background(), accessToken, response.ID); err != nil {
			log.Warn().Err(err).Str("subscriptionID", response.ID).Msg("删除 Graph 订阅失败")
		}
	}()

	log.Info().
		Str("email", request.MailInfo.Email).
		Str("subscriptionID", response.ID).
		Msg("成功创建 Graph 订阅")

	// 发送订阅成功消息
	sendSubscriptionSuccess(c, request.RefreshNeeded, refreshToken)

	// 注册邮件通知通道
	notifyChan := nfManager.RegisterChannel(response.ID)

	// 清理函数
	defer func() {
		nfManager.RemoveChannel(response.ID)
		log.Info().
			Str("subscriptionID", response.ID).
			Str("email", request.MailInfo.Email).
			Msg("清理 Graph 订阅通知通道")
	}()

	// 开始监听 Graph 通知
	listenForGraphNotifications(c, notifyChan, response.ID, request.MailInfo.Email, accessToken)
}

// listenForImapEmails 监听 IMAP 邮件
func listenForImapEmails(
	c *gin.Context,
	subscription *manager.ImapSubscription,
	email string,
) {
	// 设置超时和心跳
	timeout := createSSETimeout()
	defer timeout.Stop()

	heartbeat := createHeartbeatTicker()
	defer heartbeat.Stop()

	log.Info().
		Str("subscriptionID", subscription.ID).
		Str("email", email).
		Msg("开始 SSE 等待新邮件 (IMAP)")

	for {
		select {
		case emailData := <-subscription.EmailChan:
			if emailData != nil {
				log.Info().
					Str("subscriptionID", subscription.ID).
					Msg("通过 SSE 收到新邮件 (IMAP)")

				// 发送邮件数据
				sendSSEEvent(c, "email", emailData)

				// 发送完成消息
				sendSSEEvent(c, "complete", gin.H{
					"message": "邮件推送完成 (IMAP)",
				})
			}
			return

		case <-timeout.C:
			log.Info().
				Str("subscriptionID", subscription.ID).
				Str("email", email).
				Msg("SSE 等待邮件超时 (IMAP)")

			sendSSEEvent(c, "timeout", gin.H{
				"message": "等待邮件超时，订阅已过期 (IMAP)",
			})
			return

		case <-c.Request.Context().Done():
			log.Info().
				Str("subscriptionID", subscription.ID).
				Str("email", email).
				Msg("SSE 客户端连接断开 (IMAP)")
			return

		case <-heartbeat.C:
			// 发送心跳包保持连接活跃
			sendSSEEvent(c, "heartbeat", gin.H{
				"timestamp": time.Now().Unix(),
				"protocol":  "imap",
			})
		}
	}
}

// listenForGraphNotifications 监听 Graph 通知
func listenForGraphNotifications(
	c *gin.Context,
	notifyChan chan string,
	subscriptionID, email, accessToken string,
) {
	// 设置超时和心跳
	timeout := createSSETimeout()
	defer timeout.Stop()

	heartbeat := createHeartbeatTicker()
	defer heartbeat.Stop()

	log.Info().
		Str("subscriptionID", subscriptionID).
		Str("email", email).
		Msg("开始 SSE 等待新邮件通知 (Graph)")

	for {
		select {
		case emailID := <-notifyChan:
			log.Info().
				Str("subscriptionID", subscriptionID).
				Msg("通过 SSE 收到新邮件通知 (Graph)")

			emailData, err := graph.GetEmailByID(context.Background(), accessToken, emailID)
			if err != nil {
				log.Error().Err(err).Msg("获取邮件详情失败 (Graph)")
				continue
			}

			// 发送邮件数据
			sendSSEEvent(c, "email", emailData)

			// 发送完成消息
			sendSSEEvent(c, "complete", gin.H{
				"message": "邮件推送完成 (Graph)",
			})
			return

		case <-timeout.C:
			log.Info().
				Str("subscriptionID", subscriptionID).
				Str("email", email).
				Msg("SSE 等待邮件超时 (Graph)")

			sendSSEEvent(c, "timeout", gin.H{
				"message": "等待邮件超时，订阅已过期 (Graph)",
			})
			return

		case <-c.Request.Context().Done():
			log.Info().
				Str("subscriptionID", subscriptionID).
				Str("email", email).
				Msg("SSE 客户端连接断开 (Graph)")
			return

		case <-heartbeat.C:
			// 发送心跳包保持连接活跃
			sendSSEEvent(c, "heartbeat", gin.H{
				"timestamp": time.Now().Unix(),
				"protocol":  "graph",
			})
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
