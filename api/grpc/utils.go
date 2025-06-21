package grpc

import (
	"gomailapi2/internal/domain"
	"gomailapi2/internal/types"
	pb "gomailapi2/proto/pb"

	"github.com/rs/zerolog/log"
)

// protoToMailInfo 将 proto MailInfo 转换为内部 MailInfo
func protoToMailInfo(protoMailInfo *pb.MailInfo) *types.MailInfo {
	return &types.MailInfo{
		Email:           protoMailInfo.Email,
		ClientID:        protoMailInfo.ClientId,
		RefreshToken:    protoMailInfo.RefreshToken,
		ProtocolType:    protoProtocolTypeToTypes(protoMailInfo.ProtoType),
		ServiceProvider: protoServiceProviderToTypes(protoMailInfo.ServiceProvider),
	}
}

// protoProtocolTypeToTypes 将 proto ProtocolType 转换为内部 ProtocolType
func protoProtocolTypeToTypes(protoType pb.ProtocolType) types.ProtocolType {
	switch protoType {
	case pb.ProtocolType_IMAP:
		return types.ProtocolTypeIMAP
	case pb.ProtocolType_GRAPH:
		return types.ProtocolTypeGraph
	default:
		return types.ProtocolTypeIMAP // 默认值
	}
}

// typesToProtoProtocolType 将内部 ProtocolType 转换为 proto ProtocolType
func typesToProtoProtocolType(protoType types.ProtocolType) pb.ProtocolType {
	switch protoType {
	case types.ProtocolTypeIMAP:
		return pb.ProtocolType_IMAP
	case types.ProtocolTypeGraph:
		return pb.ProtocolType_GRAPH
	}
	return pb.ProtocolType_IMAP
}

// protoServiceProviderToTypes 将 proto ServiceProvider 转换为内部 ServiceProvider
func protoServiceProviderToTypes(protoProvider pb.ServiceProvider) types.ServiceProvider {
	switch protoProvider {
	case pb.ServiceProvider_MICROSOFT:
		return types.ServiceProviderMicrosoft
	case pb.ServiceProvider_GOOGLE:
		return types.ServiceProviderGoogle
	default:
		return types.ServiceProviderMicrosoft // 默认值
	}
}

// domainEmailToProto 将 domain.Email 转换为 proto Email
func domainEmailToProto(email *domain.Email) *pb.Email {
	if email == nil {
		return nil
	}

	result := &pb.Email{
		Id:      email.ID,
		Subject: email.Subject,
		Date:    email.Date,
		Text:    email.Text,
		Html:    email.HTML,
	}

	if email.From != nil {
		result.From = &pb.EmailAddress{
			Name:    email.From.Name,
			Address: email.From.Address,
		}
	}

	if email.To != nil {
		result.To = &pb.EmailAddress{
			Name:    email.To.Name,
			Address: email.To.Address,
		}
	}

	return result
}

// sendSubscriptionSuccess 发送订阅成功消息
func (s *MailServer) sendSubscriptionSuccess(stream pb.MailService_SubscribeMailServer, refreshNeeded bool, refreshToken string) error {
	message := "订阅成功"

	event := &pb.MailEvent{
		EventType: "subscription",
		Message:   &message, // 在 .proto 文件中定义的 optional string 字段会变成 *string❗
	}

	if refreshNeeded && refreshToken != "" {
		event.RefreshToken = &refreshToken
	}

	if err := stream.Send(event); err != nil {
		log.Error().Err(err).Str("eventType", "subscription").Msg("发送 gRPC 事件失败")
		return err
	}

	return nil
}

// sendEmailEvent 发送邮件事件
func (s *MailServer) sendEmailEvent(stream pb.MailService_SubscribeMailServer, email *domain.Email) error {
	event := &pb.MailEvent{
		EventType: "email",
		Email:     domainEmailToProto(email),
	}

	if err := stream.Send(event); err != nil {
		log.Error().Err(err).Str("eventType", "email").Msg("发送 gRPC 事件失败")
		return err
	}

	return nil
}

// sendHeartbeatEvent 发送心跳事件
func (s *MailServer) sendHeartbeatEvent(stream pb.MailService_SubscribeMailServer) error {
	message := "heartbeat"
	event := &pb.MailEvent{
		EventType: "heartbeat",
		Message:   &message,
	}

	if err := stream.Send(event); err != nil {
		log.Error().Err(err).Str("eventType", "heartbeat").Msg("发送 gRPC 事件失败")
		return err
	}

	return nil
}

// sendCompleteEvent 发送完成事件
func (s *MailServer) sendCompleteEvent(stream pb.MailService_SubscribeMailServer, message string) error {
	event := &pb.MailEvent{
		EventType: "complete",
		Message:   &message,
	}

	if err := stream.Send(event); err != nil {
		log.Error().Err(err).Str("eventType", "complete").Msg("发送 gRPC 事件失败")
		return err
	}

	return nil
}

// sendErrorEvent 发送错误事件
func (s *MailServer) sendErrorEvent(stream pb.MailService_SubscribeMailServer, errorMsg string) error {
	event := &pb.MailEvent{
		EventType: "error",
		Message:   &errorMsg,
	}

	if err := stream.Send(event); err != nil {
		log.Error().Err(err).Str("eventType", "error").Msg("发送 gRPC 事件失败")
		return err
	}

	return nil
}
