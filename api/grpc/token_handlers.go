package grpc

import (
	"context"
	pb "gomailapi2/proto/pb"
	"sync"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RefreshToken 刷新单个 Token
func (s *MailServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	// 验证请求
	if req.MailInfo == nil {
		return nil, status.Error(codes.InvalidArgument, "MailInfo 不能为空")
	}

	log.Info().
		Str("email", req.MailInfo.Email).
		Str("provider", req.MailInfo.ServiceProvider.String()).
		Msg("gRPC 收到刷新 Token 请求")

	// 转换为内部类型
	mailInfo := protoToMailInfo(req.MailInfo)

	// 刷新 Token
	newRefreshToken, err := s.tokenProvider.GetRefreshToken(mailInfo)
	if err != nil {
		log.Error().Err(err).Str("email", req.MailInfo.Email).Msg("刷新 Token 失败")
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Info().Str("email", req.MailInfo.Email).Msg("成功刷新 Token")
	return &pb.RefreshTokenResponse{
		NewRefreshToken: newRefreshToken,
	}, nil
}

// BatchRefreshToken 批量刷新 Token（并发处理，限制每次最多 100 个）
func (s *MailServer) BatchRefreshToken(ctx context.Context, req *pb.BatchRefreshTokenRequest) (*pb.BatchRefreshTokenResponse, error) {
	// 验证请求
	if len(req.MailInfos) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Token 列表不能为空")
	}

	// 限制每次最多处理 100 个
	const maxBatchSize = 100
	mailInfoCount := len(req.MailInfos)
	if mailInfoCount > maxBatchSize {
		return nil, status.Error(codes.InvalidArgument, "每次最多只能处理 100 个 Token")
	}

	log.Info().
		Int("count", mailInfoCount).
		Msg("gRPC 收到批量刷新 Token 请求")

	// 使用 WaitGroup 等待所有 goroutine 完成
	var wg sync.WaitGroup
	// 使用 Mutex 保护共享变量
	var mu sync.Mutex
	var results []*pb.BatchRefreshResult
	var successCount, failCount int32

	// 并发处理每个 Token
	for _, mailInfoProto := range req.MailInfos {
		wg.Add(1)

		// 启动 goroutine 处理单个 Token
		go func(mailInfoProto *pb.MailInfo) {
			defer wg.Done()

			result := &pb.BatchRefreshResult{
				Email: mailInfoProto.Email,
			}

			// 转换为内部类型
			mailInfo := protoToMailInfo(mailInfoProto)

			// 刷新 Token
			newRefreshToken, err := s.tokenProvider.GetRefreshToken(mailInfo)
			if err != nil {
				log.Error().Err(err).Str("email", mailInfoProto.Email).Msg("批量刷新 Token 失败")
				errorMsg := err.Error()
				result.Error = &errorMsg

				// 线程安全地更新失败计数
				mu.Lock()
				failCount++
				mu.Unlock()
			} else {
				log.Info().Str("email", mailInfoProto.Email).Msg("刷新 Token 成功")
				result.NewRefreshToken = newRefreshToken

				// 线程安全地更新成功计数
				mu.Lock()
				successCount++
				mu.Unlock()
			}

			// 线程安全地添加结果
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(mailInfoProto)
	}

	// 等待所有 goroutine 完成
	wg.Wait()

	log.Info().
		Int32("successCount", successCount).
		Int32("failCount", failCount).
		Msg("批量刷新 Token 完成")

	return &pb.BatchRefreshTokenResponse{
		SuccessCount: successCount,
		FailCount:    failCount,
		Results:      results,
	}, nil
}
