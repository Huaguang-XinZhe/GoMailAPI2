package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gomailapi2/api/grpc"
	"gomailapi2/api/rest"
	"gomailapi2/internal/config"
	"gomailapi2/internal/manager"
	"gomailapi2/internal/provider/token"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// 设置日志
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// 加载配置
	cfg := config.LoadConfig()

	log.Info().Msg("启动统一邮件服务器 (gRPC + REST)")

	// 初始化 token provider
	tokenProvider, err := token.NewTokenProvider(cfg.Cache)
	if err != nil {
		log.Fatal().Err(err).Msg("初始化 token provider 失败")
	}
	defer tokenProvider.Close()

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

	mailServer := grpc.NewMailServer(tokenProvider, nfManager, imapManager)

	// 启动 gRPC 服务器（在 goroutine 中）
	go func() {
		if err := mailServer.StartServer(grpcPort); err != nil {
			log.Fatal().Err(err).Msg("gRPC 服务器启动失败")
		}
	}()

	// 启动 REST 服务器（在 goroutine 中）
	restServer := &http.Server{}

	router := rest.SetupRouter(tokenProvider, nfManager, imapManager)

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
