package rest

import (
	"gomailapi2/api/rest/handler"
	"gomailapi2/internal/manager"
	"gomailapi2/internal/provider/token"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func SetupRouter(tokenProvider *token.TokenProvider, nfManager *manager.NotificationManager, imapManager *manager.ImapSubscriptionManager) *gin.Engine {
	if !gin.IsDebugging() { // ? gin Mode 是什么？
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// 中间件
	router.Use(gin.LoggerWithWriter(log.Logger))
	router.Use(gin.Recovery()) // 捕获 panic 并返回 500 错误

	// API 路由
	apiGroup := router.Group("gomailapi2")

	// 统一邮件端点 - 推荐使用
	{
		// 统一获取最新邮件端点（支持 IMAP 和 Graph 协议）
		apiGroup.POST("/mail/latest", handler.HandleUnifiedLatestMail(tokenProvider))
		// 统一查找邮件端点（支持 IMAP 和 Graph 协议）
		apiGroup.POST("/mail/find/:emailID", handler.HandleUnifiedFindMail(tokenProvider))
		// 统一获取垃圾邮件端点（支持 IMAP 和 Graph 协议）
		apiGroup.POST("/mail/junk/latest", handler.HandleUnifiedJunkMail(tokenProvider))
		// 统一邮件订阅路由（支持 IMAP 和 Graph 协议）
		apiGroup.POST("/subscribe-sse", handler.HandleUnifiedSubscribeSSE(tokenProvider, nfManager, imapManager))
	}

	// Token 相关端点
	tokenGroup := apiGroup.Group("/token")
	{
		// 刷新单个 Token
		tokenGroup.POST("/refresh", handler.HandleRefreshToken(tokenProvider))
		// 批量刷新 Token
		tokenGroup.POST("/batch/refresh", handler.HandleBatchRefreshToken(tokenProvider))
	}

	// Graph API 相关路由
	graphGroup := apiGroup.Group("/graph")
	{
		// Graph Webhook 路由
		graphGroup.POST("/webhook", handler.HandleGraphWebhook(nfManager))
	}

	return router
}
