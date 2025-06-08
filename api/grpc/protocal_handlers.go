package grpc

import (
	"context"
	"gomailapi2/internal/types"
	pb "gomailapi2/proto/pb"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DetectProtocolType 检测协议类型
func (s *MailServer) DetectProtocolType(ctx context.Context, req *pb.DetectProtocolTypeRequest) (*pb.DetectProtocolTypeResponse, error) {
	// 验证请求
	if req.MailInfo == nil {
		return nil, status.Error(codes.InvalidArgument, "MailInfo 不能为空")
	}

	log.Info().
		Str("email", req.MailInfo.Email).
		Str("provider", req.MailInfo.ServiceProvider.String()).
		Msg("gRPC 收到协议类型检测请求")

	// 转换为内部类型
	mailInfo := protoToMailInfo(req.MailInfo)

	// 检测协议类型
	result, err := s.protocolService.DetectProtocolType(mailInfo)
	if err != nil {
		log.Error().Err(err).Str("email", req.MailInfo.Email).Msg("协议类型检测失败")
		return nil, status.Error(codes.Internal, err.Error())
	}

	// 转换协议类型为 protobuf 枚举
	protoType := typesToProtoProtocolType(result.ProtoType)

	log.Info().
		Str("email", req.MailInfo.Email).
		Str("detectedType", string(result.ProtoType)).
		Msg("协议类型检测成功")

	return &pb.DetectProtocolTypeResponse{
		ProtoType: protoType,
	}, nil
}

// BatchDetectProtocolType 批量检测协议类型
func (s *MailServer) BatchDetectProtocolType(ctx context.Context, req *pb.BatchDetectProtocolTypeRequest) (*pb.BatchDetectProtocolTypeResponse, error) {
	// 验证请求
	if len(req.MailInfos) == 0 {
		return nil, status.Error(codes.InvalidArgument, "MailInfos 不能为空")
	}

	log.Info().
		Int("count", len(req.MailInfos)).
		Msg("gRPC 收到批量协议类型检测请求")

	// 转换为内部类型
	var mailInfos []*types.MailInfo
	for _, pbMailInfo := range req.MailInfos {
		mailInfo := protoToMailInfo(pbMailInfo)
		mailInfos = append(mailInfos, mailInfo)
	}

	// 批量检测协议类型
	result, err := s.protocolService.BatchDetectProtocolType(mailInfos)
	if err != nil {
		log.Error().Err(err).Int("count", len(req.MailInfos)).Msg("批量协议类型检测失败")
		return nil, status.Error(codes.Internal, err.Error())
	}

	// 转换结果为 protobuf 类型
	var pbResults []*pb.BatchDetectProtocolTypeResult
	for _, item := range result.Results {
		pbResult := &pb.BatchDetectProtocolTypeResult{
			Email: item.Email,
		}

		// 如果有错误，设置错误信息
		if item.Error != "" {
			pbResult.Error = &item.Error
		} else {
			// 转换协议类型为 protobuf 枚举
			pbResult.ProtoType = typesToProtoProtocolType(item.ProtoType)
		}

		pbResults = append(pbResults, pbResult)
	}

	log.Info().
		Int("total", len(req.MailInfos)).
		Int("success", result.SuccessCount).
		Int("fail", result.FailCount).
		Msg("批量协议类型检测完成")

	return &pb.BatchDetectProtocolTypeResponse{
		SuccessCount: int32(result.SuccessCount),
		FailCount:    int32(result.FailCount),
		Results:      pbResults,
	}, nil
}
