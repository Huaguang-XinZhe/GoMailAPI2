package grpc

import (
	"context"
	"gomailapi2/api/common"
	"gomailapi2/internal/client/graph"
	"gomailapi2/internal/client/imap/outlook"
	"gomailapi2/internal/manager"
	"gomailapi2/internal/types"
	pb "gomailapi2/proto/pb"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SubscribeMail 邮件订阅流
func (s *MailServer) SubscribeMail(req *pb.SubscribeMailRequest, stream pb.MailService_SubscribeMailServer) error {
	// 验证请求
	if req.MailInfo == nil {
		return status.Error(codes.InvalidArgument, "MailInfo 不能为空")
	}

	log.Info().
		Str("email", req.MailInfo.Email).
		Str("protocol", req.MailInfo.ProtoType.String()).
		Bool("refreshNeeded", req.RefreshNeeded).
		Msg("gRPC 收到邮件订阅请求")

	// 转换 MailInfo
	mailInfo := protoToMailInfo(req.MailInfo)

	// 获取令牌
	accessToken, refreshToken, err := common.GetTokens(s.tokenProvider, req.RefreshNeeded, mailInfo)
	if err != nil {
		log.Error().Err(err).Msg("获取 token 失败")
		return status.Error(codes.Unauthenticated, err.Error())
	}

	// 根据协议类型选择处理方式
	switch req.MailInfo.ProtoType {
	case pb.ProtocolType_IMAP:
		return s.handleImapSubscriptionStream(stream, req, accessToken, refreshToken, mailInfo)
	case pb.ProtocolType_GRAPH:
		return s.handleGraphSubscriptionStream(stream, req, accessToken, refreshToken)
	default:
		return status.Error(codes.InvalidArgument, "不支持的协议类型: "+req.MailInfo.ProtoType.String())
	}
}

// handleImapSubscriptionStream 处理 IMAP 协议订阅流
func (s *MailServer) handleImapSubscriptionStream(
	stream pb.MailService_SubscribeMailServer,
	req *pb.SubscribeMailRequest,
	accessToken, refreshToken string,
	mailInfo *types.MailInfo,
) error {
	// 创建 IMAP 客户端
	imapClient := outlook.NewOutlookImapClient(common.MailInfoToCredentials(mailInfo), accessToken)

	// 创建新订阅
	subscription, err := s.imapManager.CreateSubscription(imapClient, req.MailInfo.Email)
	if err != nil {
		log.Error().Err(err).Msg("创建 IMAP 订阅失败")
		return status.Error(codes.Internal, err.Error())
	}

	log.Info().
		Str("email", req.MailInfo.Email).
		Str("subscriptionID", subscription.ID).
		Msg("成功创建 IMAP 订阅")

	// 清理函数
	defer func() {
		s.imapManager.CancelSubscription(subscription.ID)
		log.Info().
			Str("subscriptionID", subscription.ID).
			Str("email", req.MailInfo.Email).
			Msg("清理 IMAP 订阅")
	}()

	// 启动订阅监听
	if err := s.imapManager.StartSubscription(subscription); err != nil {
		log.Error().Err(err).Msg("启动 IMAP 订阅监听失败")
		return status.Error(codes.Internal, err.Error())
	}

	// 发送订阅成功消息
	if err := s.sendSubscriptionSuccess(stream, req.RefreshNeeded, refreshToken); err != nil {
		return err
	}

	// 开始监听 IMAP 邮件
	return s.listenForImapEmailsStream(stream, subscription, req.MailInfo.Email)
}

// handleGraphSubscriptionStream 处理 Graph API 协议订阅流
func (s *MailServer) handleGraphSubscriptionStream(
	stream pb.MailService_SubscribeMailServer,
	req *pb.SubscribeMailRequest,
	accessToken, refreshToken string,
) error {
	// 创建新订阅
	response, err := graph.CreateSubscription(context.Background(), accessToken, common.GraphNotificationURL)
	if err != nil {
		log.Error().Err(err).Str("email", req.MailInfo.Email).Msg("创建 Graph 订阅失败")
		return status.Error(codes.Internal, "创建订阅失败: "+err.Error())
	}

	// 结束后，自动取消当前订阅
	defer func() {
		if err := graph.DeleteSubscription(context.Background(), accessToken, response.ID); err != nil {
			log.Warn().Err(err).Str("subscriptionID", response.ID).Msg("删除 Graph 订阅失败")
		}
	}()

	log.Info().
		Str("email", req.MailInfo.Email).
		Str("subscriptionID", response.ID).
		Msg("成功创建 Graph 订阅")

	// 发送订阅成功消息
	if err := s.sendSubscriptionSuccess(stream, req.RefreshNeeded, refreshToken); err != nil {
		return err
	}

	// 注册邮件通知通道
	notifyChan := s.nfManager.RegisterChannel(response.ID)

	// 清理函数
	defer func() {
		s.nfManager.RemoveChannel(response.ID)
		log.Info().
			Str("subscriptionID", response.ID).
			Str("email", req.MailInfo.Email).
			Msg("清理 Graph 订阅通知通道")
	}()

	// 开始监听 Graph 通知
	return s.listenForGraphNotificationsStream(stream, notifyChan, response.ID, req.MailInfo.Email, accessToken)
}

// listenForImapEmailsStream 监听 IMAP 邮件流
func (s *MailServer) listenForImapEmailsStream(
	stream pb.MailService_SubscribeMailServer,
	subscription *manager.ImapSubscription,
	email string,
) error {
	// 设置超时和心跳
	timeout := time.NewTimer(common.TimeoutMinutes * time.Minute)
	defer timeout.Stop()

	heartbeat := time.NewTicker(common.HeartbeatIntervalSeconds * time.Second)
	defer heartbeat.Stop()

	log.Info().
		Str("subscriptionID", subscription.ID).
		Str("email", email).
		Msg("开始 gRPC 等待新邮件 (IMAP)")

	for {
		select {
		case emailData := <-subscription.EmailChan:
			if emailData != nil {
				log.Info().
					Str("subscriptionID", subscription.ID).
					Msg("通过 gRPC 流收到新邮件 (IMAP)")

				// 发送邮件数据
				if err := s.sendEmailEvent(stream, emailData); err != nil {
					return err
				}

				// 发送完成消息
				if err := s.sendCompleteEvent(stream, "邮件推送完成 (IMAP)"); err != nil {
					return err
				}
			}
			return nil

		case <-timeout.C:
			log.Info().
				Str("subscriptionID", subscription.ID).
				Str("email", email).
				Msg("gRPC IMAP 订阅超时")
			return status.Error(codes.DeadlineExceeded, "订阅超时")

		case <-heartbeat.C:
			// 发送心跳
			if err := s.sendHeartbeatEvent(stream); err != nil {
				return err
			}

		case <-stream.Context().Done():
			log.Info().
				Str("subscriptionID", subscription.ID).
				Str("email", email).
				Msg("gRPC IMAP 客户端断开连接")
			return nil
		}
	}
}

// listenForGraphNotificationsStream 监听 Graph 通知流
func (s *MailServer) listenForGraphNotificationsStream(
	stream pb.MailService_SubscribeMailServer,
	notifyChan chan string,
	subscriptionID, email, accessToken string,
) error {
	// 设置超时和心跳
	timeout := time.NewTimer(common.TimeoutMinutes * time.Minute)
	defer timeout.Stop()

	heartbeat := time.NewTicker(common.HeartbeatIntervalSeconds * time.Second)
	defer heartbeat.Stop()

	log.Info().
		Str("subscriptionID", subscriptionID).
		Str("email", email).
		Msg("开始 gRPC 等待新邮件 (Graph)")

	for {
		select {
		case emailID := <-notifyChan:
			log.Info().
				Str("subscriptionID", subscriptionID).
				Str("emailID", emailID).
				Msg("通过 gRPC 流收到新邮件通知 (Graph)")

			// 获取邮件详情
			emailData, err := graph.GetEmailByID(context.Background(), accessToken, emailID)
			if err != nil {
				log.Error().Err(err).Str("emailID", emailID).Msg("获取邮件详情失败")
				if err := s.sendErrorEvent(stream, err.Error()); err != nil {
					return err
				}
				continue
			}

			// 发送邮件数据
			if err := s.sendEmailEvent(stream, emailData); err != nil {
				return err
			}

			// 发送完成消息
			if err := s.sendCompleteEvent(stream, "邮件推送完成 (Graph)"); err != nil {
				return err
			}
			return nil

		case <-timeout.C:
			log.Info().
				Str("subscriptionID", subscriptionID).
				Str("email", email).
				Msg("gRPC Graph 订阅超时")
			return status.Error(codes.DeadlineExceeded, "订阅超时")

		case <-heartbeat.C:
			// 发送心跳
			if err := s.sendHeartbeatEvent(stream); err != nil {
				return err
			}

		case <-stream.Context().Done():
			log.Info().
				Str("subscriptionID", subscriptionID).
				Str("email", email).
				Msg("gRPC Graph 客户端断开连接")
			return nil
		}
	}
}
