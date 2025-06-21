package handler

import (
	"context"
	"gomailapi2/api/common"
	"gomailapi2/api/rest/dto"
	"gomailapi2/internal/client/graph"
	"gomailapi2/internal/client/imap/outlook"
	"gomailapi2/internal/provider/token"
	"gomailapi2/internal/types"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// HandleUnifiedFindMail 统一处理查找邮件的请求，支持 Graph API 和 IMAP 协议
func HandleUnifiedFindMail(tokenProvider *token.TokenProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从路径中获取 emailID
		emailID := c.Param("emailID")
		if emailID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "邮件 ID 不能为空"})
			return
		}

		// 解析请求
		request, err := parseFindMailRequest(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 验证 MailInfo
		if request.MailInfo == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "MailInfo 不能为空"})
			return
		}

		log.Info().
			Str("email", request.MailInfo.Email).
			Str("protocol", string(request.MailInfo.ProtocolType)).
			Str("provider", string(request.MailInfo.ServiceProvider)).
			Str("emailID", emailID).
			Msg("收到查找邮件请求")

		// 根据协议类型处理请求
		switch request.MailInfo.ProtocolType {
		case types.ProtocolTypeGraph:
			handleGraphFindMail(c, request, tokenProvider, emailID)
		case types.ProtocolTypeIMAP:
			handleImapFindMail(c, request, tokenProvider, emailID)
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "不支持的协议类型: " + string(request.MailInfo.ProtocolType),
			})
		}
	}
}

// AQMkADAwATM0MDAAMi04OABjZC00YzE3LTAwAi0wMAoARgAAAx0x4Uhm0AVMo2iHjMDoSEYHAOcTatRO7PhKvvyq-xjvgbAAAAIBDAAAAOcTatRO7PhKvvyq-xjvgbAAAAE7EyAAAAA=
// handleGraphFindMail 处理 Graph API 协议的邮件查找
func handleGraphFindMail(c *gin.Context, request *dto.FindMailRequest, tokenProvider *token.TokenProvider, emailID string) {
	// 获取访问令牌
	accessToken, err := tokenProvider.GetAccessToken(request.MailInfo)
	if err != nil {
		log.Error().Err(err).Str("email", request.MailInfo.Email).Msg("获取 Graph API 访问令牌失败")
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 根据邮件 ID 查找邮件
	email, err := graph.GetEmailByID(context.Background(), accessToken, emailID)
	if err != nil {
		log.Error().Err(err).Str("email", request.MailInfo.Email).Str("emailID", emailID).Msg("通过 Graph API 查找邮件失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().Str("email", request.MailInfo.Email).Str("emailID", emailID).Msg("成功通过 Graph API 查找邮件")
	c.JSON(http.StatusOK, gin.H{"email": email})
}

// CAAct-4UR-Q_MgMpp4t6-M-V_i7++Efn2yKDR1vRc975bcgan-A@mail.gmail.com
// handleImapFindMail 处理 IMAP 协议的邮件查找
func handleImapFindMail(c *gin.Context, request *dto.FindMailRequest, tokenProvider *token.TokenProvider, emailID string) {
	// 获取访问令牌
	accessToken, err := tokenProvider.GetAccessToken(request.MailInfo)
	if err != nil {
		log.Error().Err(err).Str("email", request.MailInfo.Email).Msg("获取 IMAP 访问令牌失败")
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 创建 IMAP 客户端
	imapClient := outlook.NewOutlookImapClient(common.MailInfoToCredentials(request.MailInfo), accessToken)

	// 根据邮件 ID 查找邮件
	email, err := imapClient.FetchEmailByID(emailID)
	if err != nil {
		log.Error().Err(err).Str("email", request.MailInfo.Email).Str("emailID", emailID).Msg("通过 IMAP 查找邮件失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().Str("email", request.MailInfo.Email).Str("emailID", emailID).Msg("成功通过 IMAP 查找邮件")
	c.JSON(http.StatusOK, gin.H{"email": email})
}

// parseFindMailRequest 解析查找邮件请求
func parseFindMailRequest(c *gin.Context) (*dto.FindMailRequest, error) {
	var request dto.FindMailRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Msg("解析查找邮件请求失败")
		return nil, err
	}
	return &request, nil
}
