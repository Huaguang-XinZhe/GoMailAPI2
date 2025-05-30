package handler

import (
	"context"
	"gomailapi2/api/rest/dto"
	"gomailapi2/internal/client/graph"
	"gomailapi2/internal/provider/token"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleNewMail(tokenProvider *token.TokenProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request dto.GetNewMailRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if request.MailInfo == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "MailInfo 不能为空"})
			return
		}

		token, err := tokenProvider.GetAccessToken(request.MailInfo)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		email, err := graph.GetLatestEmail(context.Background(), token)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": email})
	}
}
