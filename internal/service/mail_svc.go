package service

// import (
// 	"fmt"

// 	"gomailapi2/internal/oauth"

// 	"github.com/rs/zerolog/log"
// )

// // MailResponse 邮件响应结构
// type MailResponse struct {
// 	Success         bool           `json:"success"`
// 	Message         string         `json:"message"`
// 	Data            map[string]any `json:"data,omitempty"`
// 	NewRefreshToken string         `json:"newRefreshToken,omitempty"`
// }

// // MailService 邮件服务，协调 IMAP 和 Graph 服务
// type MailService struct {
// 	imapService  *IMAPService
// 	graphService *GraphService
// }

// func NewMailService(imapService *IMAPService, graphService *GraphService) *MailService {
// 	return &MailService{
// 		imapService:  imapService,
// 		graphService: graphService,
// 	}
// }

// // GetNewMails 获取新邮件
// func (s *MailService) GetNewMails(mailInfo *oauth.MailInfo) (*MailResponse, error) {
// 	log.Info().
// 		Str("email", mailInfo.Email).
// 		Str("protocol", string(mailInfo.ProtoType)).
// 		Str("provider", string(mailInfo.ServiceProvider)).
// 		Bool("refreshRequired", mailInfo.RefreshRequired).
// 		Msg("收到获取新邮件请求")

// 	// 验证必填字段
// 	if err := s.validateMailInfo(mailInfo); err != nil {
// 		return &MailResponse{
// 			Success: false,
// 			Message: fmt.Sprintf("请求参数验证失败: %v", err),
// 		}, nil
// 	}

// 	// 根据协议类型选择服务
// 	switch mailInfo.ProtoType {
// 	case oauth.ProtocolTypeIMAP:
// 		return s.imapService.GetNewMails(mailInfo)
// 	case oauth.ProtocolTypeGraph:
// 		return s.graphService.GetNewMails(mailInfo)
// 	default:
// 		return &MailResponse{
// 			Success: false,
// 			Message: fmt.Sprintf("不支持的协议类型: %s", mailInfo.ProtoType),
// 		}, nil
// 	}
// }

// // SubscribeToMails 订阅邮件
// func (s *MailService) SubscribeToMails(mailInfo *oauth.MailInfo) (*MailResponse, error) {
// 	log.Info().
// 		Str("email", mailInfo.Email).
// 		Str("protocol", string(mailInfo.ProtoType)).
// 		Str("provider", string(mailInfo.ServiceProvider)).
// 		Bool("refreshRequired", mailInfo.RefreshRequired).
// 		Msg("收到订阅邮件请求")

// 	// 验证必填字段
// 	if err := s.validateMailInfo(mailInfo); err != nil {
// 		return &MailResponse{
// 			Success: false,
// 			Message: fmt.Sprintf("请求参数验证失败: %v", err),
// 		}, nil
// 	}

// 	// 根据协议类型选择服务
// 	switch mailInfo.ProtoType {
// 	case oauth.ProtocolTypeIMAP:
// 		return s.imapService.SubscribeToMails(mailInfo)
// 	case oauth.ProtocolTypeGraph:
// 		return s.graphService.SubscribeToMails(mailInfo)
// 	default:
// 		return &MailResponse{
// 			Success: false,
// 			Message: fmt.Sprintf("不支持的协议类型: %s", mailInfo.ProtoType),
// 		}, nil
// 	}
// }

// // validateMailInfo 验证邮件信息
// func (s *MailService) validateMailInfo(mailInfo *oauth.MailInfo) error {
// 	if mailInfo.Email == "" {
// 		return fmt.Errorf("email 不能为空")
// 	}
// 	if mailInfo.ClientID == "" {
// 		return fmt.Errorf("clientId 不能为空")
// 	}
// 	if mailInfo.RefreshToken == "" {
// 		return fmt.Errorf("refreshToken 不能为空")
// 	}
// 	if mailInfo.ProtoType == "" {
// 		return fmt.Errorf("protoType 不能为空")
// 	}
// 	if mailInfo.ServiceProvider == "" {
// 		return fmt.Errorf("serviceProvider 不能为空")
// 	}

// 	// 验证协议类型
// 	if mailInfo.ProtoType != oauth.ProtocolTypeIMAP && mailInfo.ProtoType != oauth.ProtocolTypeGraph {
// 		return fmt.Errorf("不支持的协议类型: %s", mailInfo.ProtoType)
// 	}

// 	// 验证服务提供商
// 	if mailInfo.ServiceProvider != oauth.ServiceProviderMicrosoft && mailInfo.ServiceProvider != oauth.ServiceProviderGoogle {
// 		return fmt.Errorf("不支持的服务提供商: %s", mailInfo.ServiceProvider)
// 	}

// 	// Graph 协议目前只支持 Microsoft
// 	if mailInfo.ProtoType == oauth.ProtocolTypeGraph && mailInfo.ServiceProvider != oauth.ServiceProviderMicrosoft {
// 		return fmt.Errorf("Graph 协议目前只支持 Microsoft 服务提供商")
// 	}

// 	return nil
// }
