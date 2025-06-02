package grpc

import (
	"context"
	"gomailapi2/api/common"
	"gomailapi2/internal/client/graph"
	"gomailapi2/internal/client/imap/outlook"
	"gomailapi2/internal/domain"
	pb "gomailapi2/proto/pb"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetLatestMail 获取最新邮件
func (s *MailServer) GetLatestMail(ctx context.Context, req *pb.GetNewMailRequest) (*pb.GetNewMailResponse, error) {
	// 验证请求
	if req.MailInfo == nil {
		log.Error().Msg("MailInfo 不能为空")
		return nil, status.Error(codes.InvalidArgument, "MailInfo 不能为空")
	}

	log.Info().
		Str("email", req.MailInfo.Email).
		Str("protocol", req.MailInfo.ProtoType.String()).
		Str("provider", req.MailInfo.ServiceProvider.String()).
		Bool("refreshNeeded", req.RefreshNeeded).
		Msg("gRPC 收到获取最新邮件请求")

	// 转换 MailInfo
	mailInfo := protoToMailInfo(req.MailInfo)

	// 获取令牌
	accessToken, refreshToken, err := common.GetTokens(s.tokenProvider, req.RefreshNeeded, mailInfo)
	if err != nil {
		log.Error().Err(err).Str("email", req.MailInfo.Email).Msg("获取访问令牌失败")
		return nil, status.Error(codes.Internal, err.Error())
	}

	// 根据协议类型处理请求
	var email *domain.Email
	switch req.MailInfo.ProtoType {
	case pb.ProtocolType_GRAPH:
		email, err = graph.GetLatestEmail(ctx, accessToken)
	case pb.ProtocolType_IMAP:
		imapClient := outlook.NewOutlookImapClient(common.MailInfoToCredentials(mailInfo), accessToken)
		email, err = imapClient.FetchLatestEmail()
	default:
		return nil, status.Error(codes.InvalidArgument, "不支持的协议类型")
	}

	if err != nil {
		log.Error().Err(err).Str("email", req.MailInfo.Email).Msg("获取最新邮件失败")
		return nil, status.Error(codes.Internal, err.Error())
	}

	// 构建响应
	response := &pb.GetNewMailResponse{
		Email: domainEmailToProto(email),
	}

	if req.RefreshNeeded && refreshToken != "" {
		log.Debug().
			Str("email", req.MailInfo.Email).
			Msg("设置 RefreshToken 字段")
		response.RefreshToken = &refreshToken
	}

	log.Info().Str("email", req.MailInfo.Email).Msg("成功获取最新邮件")
	return response, nil
}

// FindMail 查找特定邮件
func (s *MailServer) FindMail(ctx context.Context, req *pb.FindMailRequest) (*pb.FindMailResponse, error) {
	// 验证请求
	if req.MailInfo == nil {
		return nil, status.Error(codes.InvalidArgument, "MailInfo 不能为空")
	}
	if req.EmailId == "" {
		return nil, status.Error(codes.InvalidArgument, "邮件 ID 不能为空")
	}

	log.Info().
		Str("email", req.MailInfo.Email).
		Str("protocol", req.MailInfo.ProtoType.String()).
		Str("provider", req.MailInfo.ServiceProvider.String()).
		Str("emailID", req.EmailId).
		Msg("gRPC 收到查找邮件请求")

	// 转换 MailInfo
	mailInfo := protoToMailInfo(req.MailInfo)

	// 获取访问令牌
	accessToken, err := s.tokenProvider.GetAccessToken(mailInfo)
	if err != nil {
		log.Error().Err(err).Str("email", req.MailInfo.Email).Msg("获取访问令牌失败")
		return nil, status.Error(codes.Internal, err.Error())
	}

	// 根据协议类型处理请求
	var email *domain.Email
	switch req.MailInfo.ProtoType {
	case pb.ProtocolType_GRAPH:
		email, err = graph.GetEmailByID(ctx, accessToken, req.EmailId)
	case pb.ProtocolType_IMAP:
		imapClient := outlook.NewOutlookImapClient(common.MailInfoToCredentials(mailInfo), accessToken)
		email, err = imapClient.FetchEmailByID(req.EmailId)
	default:
		return nil, status.Error(codes.InvalidArgument, "不支持的协议类型")
	}

	if err != nil {
		log.Error().Err(err).Str("email", req.MailInfo.Email).Str("emailID", req.EmailId).Msg("查找邮件失败")
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Info().Str("email", req.MailInfo.Email).Str("emailID", req.EmailId).Msg("成功查找邮件")
	return &pb.FindMailResponse{
		Email: domainEmailToProto(email),
	}, nil
}

// GetJunkMail 获取垃圾邮件
func (s *MailServer) GetJunkMail(ctx context.Context, req *pb.GetNewJunkMailRequest) (*pb.GetNewJunkMailResponse, error) {
	// 验证请求
	if req.MailInfo == nil {
		return nil, status.Error(codes.InvalidArgument, "MailInfo 不能为空")
	}

	log.Info().
		Str("email", req.MailInfo.Email).
		Str("protocol", req.MailInfo.ProtoType.String()).
		Str("provider", req.MailInfo.ServiceProvider.String()).
		Msg("gRPC 收到获取垃圾邮件请求")

	// 转换 MailInfo
	mailInfo := protoToMailInfo(req.MailInfo)

	// 获取访问令牌
	accessToken, err := s.tokenProvider.GetAccessToken(mailInfo)
	if err != nil {
		log.Error().Err(err).Str("email", req.MailInfo.Email).Msg("获取访问令牌失败")
		return nil, status.Error(codes.Internal, err.Error())
	}

	// 根据协议类型处理请求
	var email *domain.Email
	switch req.MailInfo.ProtoType {
	case pb.ProtocolType_GRAPH:
		email, err = graph.GetLatestEmailFromJunk(ctx, accessToken)
	case pb.ProtocolType_IMAP:
		imapClient := outlook.NewOutlookImapClient(common.MailInfoToCredentials(mailInfo), accessToken)
		email, err = imapClient.FetchLatestJunkEmail()
	default:
		return nil, status.Error(codes.InvalidArgument, "不支持的协议类型")
	}

	if err != nil {
		log.Error().Err(err).Str("email", req.MailInfo.Email).Msg("获取垃圾邮件失败")
		return nil, status.Error(codes.Internal, err.Error())
	}

	log.Info().Str("email", req.MailInfo.Email).Msg("成功获取垃圾邮件")
	return &pb.GetNewJunkMailResponse{
		Email: domainEmailToProto(email),
	}, nil
}
