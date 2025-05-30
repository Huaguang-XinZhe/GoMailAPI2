package handler

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
)

// sendSSEEvent 发送 SSE 事件（data 为 json 格式）
func sendSSEEvent(c *gin.Context, event string, data any) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, string(jsonData))
	c.Writer.Flush()
}

// sendSSEError 发送 SSE 错误消息
func sendSSEError(c *gin.Context, message string) {
	sendSSEEvent(c, "error", gin.H{
		"message": message,
	})
}
