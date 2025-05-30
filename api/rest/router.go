package rest

import (
	"gomailapi2/api/rest/handler"
	"gomailapi2/api/rest/middleware"
	"gomailapi2/internal/notification"
	"gomailapi2/internal/provider/token"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func SetupRouter(tokenProvider *token.TokenProvider, notificationManager *notification.NotificationManager) *gin.Engine {
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
	// Graph Webhook 路由
	apiGroup.POST("/graph/webhook", handler.HandleGraphWebhook(notificationManager))
	// Graph 订阅路由（SSE 流式响应）
	apiGroup.POST("/graph/subscribe-sse", handler.HandleGraphSubscribeSSE(tokenProvider, notificationManager))
	// 获取最新一封邮件
	apiGroup.POST("/mail/new", handler.HandleNewMail(tokenProvider))
	// 根据邮件 ID 获取邮件详情
	apiGroup.POST("/mail/find/:emailID", handler.HandleFindMail(tokenProvider))
	// 从垃圾箱获取最新一封邮件
	apiGroup.POST("/mail/junk/new", handler.HandleNewJunkMail(tokenProvider))
	// setupMailRoutes(apiGroup, mailService)

	return router
}

// // setupMailRoutes 设置邮件相关路由
// func setupMailRoutes(router *gin.RouterGroup, mailService *service.MailService) {
// 	mailGroup := router.Group("/mail")
// 	{
// 		mailGroup.POST("/new", handler.HandleGetNewMails(mailService))
// 		mailGroup.POST("/subscribe", handler.HandleSubscribeToMails(mailService))
// 	}
// }
