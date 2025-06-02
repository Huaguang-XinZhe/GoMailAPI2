package handler

import (
	"context"
	"gomailapi2/api/common"
	"gomailapi2/api/rest/dto"
	"gomailapi2/internal/client/graph"
	"gomailapi2/internal/client/imap/outlook"
	"gomailapi2/internal/domain"
	"gomailapi2/internal/provider/token"
	"gomailapi2/internal/types"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// HandleUnifiedLatestMail 统一处理获取最新邮件的请求，支持 Graph API 和 IMAP 协议
func HandleUnifiedLatestMail(tokenProvider *token.TokenProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 解析请求
		request, err := parseNewMailRequest(c)
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
			Str("protocol", string(request.MailInfo.ProtoType)).
			Str("provider", string(request.MailInfo.ServiceProvider)).
			Bool("refreshNeeded", request.RefreshNeeded).
			Msg("收到获取最新邮件请求")

		// 根据协议类型处理请求
		switch request.MailInfo.ProtoType {
		case types.ProtocolTypeGraph:
			handleGraphLatestMail(c, request, tokenProvider)
		case types.ProtocolTypeIMAP:
			handleImapLatestMail(c, request, tokenProvider)
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "不支持的协议类型: " + string(request.MailInfo.ProtoType),
			})
		}
	}
}

// handleGraphLatestMail 处理 Graph API 协议的最新邮件获取
func handleGraphLatestMail(c *gin.Context, request *dto.GetNewMailRequest, tokenProvider *token.TokenProvider) {
	// 获取访问令牌
	accessToken, refreshToken, err := common.GetTokens(tokenProvider, request.RefreshNeeded, request.MailInfo)
	if err != nil {
		log.Error().Err(err).Str("email", request.MailInfo.Email).Msg("获取 Graph API 访问令牌失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取最新邮件
	email, err := graph.GetLatestEmail(context.Background(), accessToken)
	if err != nil {
		log.Error().Err(err).Str("email", request.MailInfo.Email).Msg("通过 Graph API 获取最新邮件失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := buildResponse(email, refreshToken)

	log.Info().Str("email", request.MailInfo.Email).Msg("成功通过 Graph API 获取最新邮件")
	c.JSON(http.StatusOK, response)
}

// handleImapLatestMail 处理 IMAP 协议的最新邮件获取
func handleImapLatestMail(c *gin.Context, request *dto.GetNewMailRequest, tokenProvider *token.TokenProvider) {
	// 获取令牌
	accessToken, refreshToken, err := common.GetTokens(tokenProvider, request.RefreshNeeded, request.MailInfo)
	if err != nil {
		log.Error().Err(err).Str("email", request.MailInfo.Email).Msg("获取 IMAP 访问令牌失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 创建 IMAP 客户端
	imapClient := outlook.NewOutlookImapClient(common.MailInfoToCredentials(request.MailInfo), accessToken)

	// 获取最新邮件
	email, err := imapClient.FetchLatestEmail()
	if err != nil {
		log.Error().Err(err).Str("email", request.MailInfo.Email).Msg("通过 IMAP 获取最新邮件失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := buildResponse(email, refreshToken)

	log.Info().Str("email", request.MailInfo.Email).Msg("成功通过 IMAP 获取最新邮件")
	c.JSON(http.StatusOK, response)
}

func buildResponse(email *domain.Email, refreshToken string) gin.H {
	response := gin.H{
		"email": email,
	}

	if refreshToken != "" {
		response["refreshToken"] = refreshToken
	}

	return response
}

// parseNewMailRequest 解析获取最新一封邮件请求
func parseNewMailRequest(c *gin.Context) (*dto.GetNewMailRequest, error) {
	var request dto.GetNewMailRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error().Err(err).Msg("解析获取最新一封邮件请求失败")
		return nil, err
	}
	return &request, nil
}
