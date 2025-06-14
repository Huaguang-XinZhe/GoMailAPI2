package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gomailapi2/api/common"
	"gomailapi2/api/grpc"
	"gomailapi2/api/rest"
	"gomailapi2/internal/cache/factory"
	"gomailapi2/internal/config"
	"gomailapi2/internal/manager"
	"gomailapi2/internal/provider/token"
	"gomailapi2/internal/service"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// 加载配置（先加载配置再设置日志）
	cfg := config.LoadConfig()

	// 根据配置设置日志级别和格式
	setupLogger(cfg)

	// 初始化全局配置
	common.InitGraphNotificationURL(&cfg.Webhook)

	log.Info().Msg("启动统一邮件服务器 (gRPC + REST)")

	// 初始化缓存实例
	cacheInstance, err := factory.NewCache(cfg.Cache)
	if err != nil {
		log.Fatal().Err(err).Msg("初始化缓存失败")
	}
	defer cacheInstance.Close()
	log.Info().Msg("缓存初始化完成")

	// 初始化 token provider
	tokenProvider := token.NewTokenProvider(cacheInstance)

	// 初始化 protocol service
	protocolService := service.NewProtocolService(cacheInstance)
	log.Info().Msg("协议检测服务初始化完成")

	// 初始化管理器 - 这里是关键，两个服务共享同一个实例
	nfManager := manager.NewNotificationManager()
	imapManager := manager.NewImapSubscriptionManager()

	log.Info().Msg("管理器初始化完成")

	// 设置优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动 gRPC 服务器
	grpcPort := cfg.Server.GrpcPort
	log.Info().Int("port", grpcPort).Msg("启动 gRPC 服务器...")

	mailServer := grpc.NewMailServer(tokenProvider, protocolService, nfManager, imapManager)

	// 启动 gRPC 服务器（在 goroutine 中）
	go func() {
		if err := mailServer.StartServer(grpcPort); err != nil {
			log.Fatal().Err(err).Msg("gRPC 服务器启动失败")
		}
	}()

	// 启动 REST 服务器（在 goroutine 中）
	restServer := &http.Server{}

	router := rest.SetupRouter(tokenProvider, protocolService, nfManager, imapManager)

	restAddress := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	restServer.Addr = restAddress
	restServer.Handler = router

	log.Info().Str("address", restAddress).Msg("启动 REST 服务器...")

	go func() {
		if err := restServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("REST 服务器启动失败")
		}
	}()

	// 给服务器一点时间启动
	time.Sleep(100 * time.Millisecond)

	log.Info().Msg("统一服务器启动完成")

	// 等待中断信号
	<-sigChan
	log.Info().Msg("收到关闭信号，正在优雅关闭服务器...")

	// 优雅关闭 gRPC 服务器
	mailServer.StopServer()

	// 优雅关闭 REST 服务器
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := restServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("REST 服务器关闭失败")
	} else {
		log.Info().Msg("REST 服务器已关闭")
	}

	log.Info().Msg("统一服务器已关闭")
}

// setupLogger 设置日志配置
func setupLogger(cfg *config.Config) {
	// 根据环境设置日志格式
	if cfg.IsProduction() {
		// 生产环境使用 JSON 格式
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	} else {
		// 开发环境使用控制台格式
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// 设置日志级别
	switch cfg.Log.Level {
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
}
