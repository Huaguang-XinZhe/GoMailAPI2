package handler

import (
	"context"
	"gomailapi2/api/common"
	"gomailapi2/internal/client/graph"
	"gomailapi2/internal/client/imap/outlook"
	"gomailapi2/internal/provider/token"

	"gomailapi2/api/rest/dto"
	"gomailapi2/internal/types"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// HandleUnifiedJunkMail 统一处理获取垃圾邮件的请求，支持 Graph API 和 IMAP 协议
func HandleUnifiedJunkMail(tokenProvider *token.TokenProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 解析请求
		request, err := parseJunkMailRequest(c)
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
			Msg("收到获取垃圾邮件请求")

		// 根据协议类型处理请求
		switch request.MailInfo.ProtocolType {
		case types.ProtocolTypeGraph:
			handleGraphJunkMail(c, request, tokenProvider)
		case types.ProtocolTypeIMAP:
			handleImapJunkMail(c, request, tokenProvider)
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "不支持的协议类型: " + string(request.MailInfo.ProtocolType),
			})
		}
	}
}

// handleGraphJunkMail 处理 Graph API 协议的垃圾邮件获取
func handleGraphJunkMail(c *gin.Context, request *dto.GetNewJunkMailRequest, tokenProvider *token.TokenProvider) {
	// 获取访问令牌
	accessToken, err := tokenProvider.GetAccessToken(request.MailInfo)
	if err != nil {
		log.Error().Err(err).Str("email", request.MailInfo.Email).Msg("获取 Graph API 访问令牌失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取垃圾邮件
	email, err := graph.GetLatestEmailFromJunk(context.Background(), accessToken)
	if err != nil {
		log.Error().Err(err).Str("email", request.MailInfo.Email).Msg("通过 Graph API 获取垃圾邮件失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().Str("email", request.MailInfo.Email).Msg("成功通过 Graph API 获取垃圾邮件")
	c.JSON(http.StatusOK, email)
}

// handleImapJunkMail 处理 IMAP 协议的垃圾邮件获取
func handleImapJunkMail(c *gin.Context, request *dto.GetNewJunkMailRequest, tokenProvider *token.TokenProvider) {
	// 获取令牌
	accessToken, err := tokenProvider.GetAccessToken(request.MailInfo)
	if err != nil {
		log.Error().Err(err).Str("email", request.MailInfo.Email).Msg("获取 IMAP 访问令牌失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 创建 IMAP 客户端
	imapClient := outlook.NewOutlookImapClient(common.MailInfoToCredentials(request.MailInfo), accessToken)

	// 获取垃圾邮件
	email, err := imapClient.FetchLatestJunkEmail()
	if err != nil {
		log.Error().Err(err).Str("email", request.MailInfo.Email).Msg("通过 IMAP 获取垃圾邮件失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Info().Str("email", request.MailInfo.Email).Msg("成功通过 IMAP 获取垃圾邮件")
	c.JSON(http.StatusOK, email)
}

// parseJunkMailRequest 解析获取垃圾邮件请求
func parseJunkMailRequest(c *gin.Context) (*dto.GetNewJunkMailRequest, error) {
	var request dto.GetNewJunkMailRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Msg("解析获取垃圾邮件请求失败")
		return nil, err
	}
	return &request, nil
}
