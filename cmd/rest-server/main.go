package main

import (
	"fmt"
	"os"

	"gomailapi2/api/common"
	"gomailapi2/api/rest"
	"gomailapi2/internal/config"
	"gomailapi2/internal/manager"
	"gomailapi2/internal/provider/token"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// 加载配置
	cfg := config.LoadConfig()

	// 初始化全局配置
	common.InitGraphNotificationURL(&cfg.Webhook)

	// 初始化日志
	initLogger(cfg.Log.Level)

	log.Info().Msg("正在启动邮件服务器...")

	// 初始化通知管理器
	notificationManager := manager.NewNotificationManager()
	log.Info().Msg("通知管理器初始化完成")

	// 初始化 IMAP 订阅管理器
	imapManager := manager.NewImapSubscriptionManager()
	log.Info().Msg("IMAP 订阅管理器初始化完成")

	// 初始化 TokenProvider
	tokenProvider, err := token.NewTokenProvider(cfg.Cache)
	if err != nil {
		log.Fatal().Err(err).Msg("初始化 TokenProvider 失败")
	}
	defer tokenProvider.Close()

	// 初始化路由
	router := rest.SetupRouter(tokenProvider, notificationManager, imapManager)

	// 启动服务器
	address := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Info().Str("address", address).Msg("服务器启动中...")

	if err := router.Run(address); err != nil {
		log.Fatal().Err(err).Msg("服务器启动失败")
	}
}

func initLogger(level string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// 设置日志级别
	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// 开发环境使用更友好的输出格式
	if gin.IsDebugging() { // ? 这里会影响哪些区域？只是 gin 路由部分吗？
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}
