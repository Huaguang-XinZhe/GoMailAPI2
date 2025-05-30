package handler

import (
	"context"
	"gomailapi2/api/rest/dto"
	"gomailapi2/internal/client/graph"
	"gomailapi2/internal/provider/token"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleFindMail(tokenProvider *token.TokenProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request dto.FindMailRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		token, err := tokenProvider.GetAccessToken(request.MailInfo)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// 从路径中获取 emailID
		emailID := c.Param("emailID")
		if emailID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "邮件 ID 不能为空"})
			return
		}

		email, err := graph.GetEmailByID(context.Background(), token, emailID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": email})
	}
}
