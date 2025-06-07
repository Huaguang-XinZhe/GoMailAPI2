package grpc

import (
	"fmt"
	"gomailapi2/internal/manager"
	"gomailapi2/internal/provider/token"
	"gomailapi2/internal/service"
	pb "gomailapi2/proto/pb"
	"net"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	// "google.golang.org/grpc/reflection"
)

// MailServer gRPC 邮件服务器
type MailServer struct {
	pb.UnimplementedMailServiceServer
	tokenProvider   *token.TokenProvider
	protocolService *service.ProtocolService
	nfManager       *manager.NotificationManager
	imapManager     *manager.ImapSubscriptionManager
	server          *grpc.Server
}

// NewMailServer 创建新的邮件服务器
func NewMailServer(
	tokenProvider *token.TokenProvider,
	protocolService *service.ProtocolService,
	nfManager *manager.NotificationManager,
	imapManager *manager.ImapSubscriptionManager,
) *MailServer {
	return &MailServer{
		tokenProvider:   tokenProvider,
		protocolService: protocolService,
		nfManager:       nfManager,
		imapManager:     imapManager,
	}
}

// StartServer 启动 gRPC 服务器
func (ms *MailServer) StartServer(port int) error {
	// 创建监听器
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Error().Err(err).Int("port", port).Msg("gRPC 服务器监听失败")
		return err
	}

	// 创建 gRPC 服务器
	ms.server = grpc.NewServer()

	// 注册邮件服务
	pb.RegisterMailServiceServer(ms.server, ms)

	// // 启用反射（用于调试和客户端发现）
	// reflection.Register(ms.server)

	log.Info().Int("port", port).Msg("gRPC 服务器启动成功")

	// 启动服务器（阻塞调用）
	if err := ms.server.Serve(lis); err != nil {
		log.Error().Err(err).Msg("gRPC 服务器运行失败")
		return err
	}

	return nil
}

// StopServer 优雅停止 gRPC 服务器
func (ms *MailServer) StopServer() {
	if ms.server != nil {
		log.Info().Msg("正在优雅关闭 gRPC 服务器...")
		ms.server.GracefulStop()
		log.Info().Msg("gRPC 服务器已优雅关闭")
	}
}
