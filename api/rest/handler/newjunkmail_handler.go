package handler

import (
	"context"
	"gomailapi2/internal/client/graph"
	"gomailapi2/internal/provider/token"

	"gomailapi2/api/rest/dto"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleNewJunkMail(tokenProvider *token.TokenProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request dto.GetNewJunkMailRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		token, err := tokenProvider.GetAccessToken(request.MailInfo)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		email, err := graph.GetLatestEmailFromJunk(context.Background(), token)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": email})
	}
}
