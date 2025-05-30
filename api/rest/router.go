package rest

import (
	"gomailapi2/api/rest/handler"
	"gomailapi2/api/rest/middleware"
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
	router.Use(gin.Recovery())              // 捕获 panic 并返回 500 错误
	router.Use(middleware.CORSMiddleware()) // todo 考虑有没有必要

	// API 路由
	apiGroup := router.Group("/api/v1") // todo 这里要变一下了：mail-api

	// 统一订阅端点 - 推荐使用
	{
		// 统一邮件订阅路由（支持 IMAP 和 Graph 协议）
		apiGroup.POST("/subscribe-sse", handler.HandleUnifiedSubscribeSSE(tokenProvider, nfManager, imapManager))
	}

	// Graph API 相关路由
	graphGroup := apiGroup.Group("/graph")
	{
		// Graph Webhook 路由
		graphGroup.POST("/webhook", handler.HandleGraphWebhook(nfManager))
		// 获取最新一封邮件
		graphGroup.POST("/mail/new", handler.HandleNewMail(tokenProvider))
		// 根据邮件 ID 获取邮件详情
		graphGroup.POST("/mail/find/:emailID", handler.HandleFindMail(tokenProvider))
		// 从垃圾箱获取最新一封邮件
		graphGroup.POST("/mail/junk/new", handler.HandleNewJunkMail(tokenProvider))
	}

	// IMAP 相关路由
	// imapGroup := apiGroup.Group("/imap")

	return router
}
