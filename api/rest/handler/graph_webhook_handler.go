package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"gomailapi2/internal/manager"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// NotificationData 通知数据结构体 - 基本通知只包含基础信息
type NotificationData struct {
	SubscriptionID string `json:"subscriptionId"`
	ResourceData   struct {
		ID string `json:"id"`
	} `json:"resourceData"`
}

// NotificationCollection 通知集合结构体
type NotificationCollection struct {
	Value []NotificationData `json:"value"`
}

// HandleGraphWebhook 处理 graph webhook 通知
func HandleGraphWebhook(nfManager *manager.NotificationManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Info().
			Str("method", c.Request.Method).
			Str("url", c.Request.URL.String()).
			Msg("收到 webhook 请求")

		// 处理验证请求（订阅创建时）
		validationToken := c.Query("validationToken")
		if validationToken != "" {
			log.Info().
				Str("validationToken", validationToken).
				Msg("收到验证请求，返回验证令牌")
			c.Header("Content-Type", "text/plain")
			c.String(http.StatusOK, validationToken)
			return
		}

		// 读取请求体
		body, err := c.GetRawData()
		if err != nil {
			log.Error().Err(err).Msg("读取通知内容失败")
			c.JSON(http.StatusBadRequest, gin.H{"error": "无法读取请求体"})
			return
		}

		log.Info().Str("body", string(body)).Msg("收到通知内容")

		// 解析通知内容
		var notifications NotificationCollection
		if err := json.Unmarshal(body, &notifications); err != nil {
			log.Error().Err(err).Msg("解析通知内容失败")
			c.JSON(http.StatusBadRequest, gin.H{"error": "无法解析通知内容"})
			return
		}

		// 处理每个通知
		for _, notificationData := range notifications.Value {
			processNotification(notificationData, nfManager)
		}

		log.Info().Msg("Webhook 通知处理完成")
		c.JSON(http.StatusOK, gin.H{"message": "通知处理成功"})
	}
}

// processNotification 处理单个通知
func processNotification(nfData NotificationData, nfManager *manager.NotificationManager) {
	log.Info().
		Str("subscriptionID", nfData.SubscriptionID).
		Str("resourceID", nfData.ResourceData.ID).
		Int64("timestamp", time.Now().Unix()).
		Msg("开始处理邮件通知")

	// 发送新邮件到达通知
	success := nfManager.SendNotification(nfData.SubscriptionID, nfData.ResourceData.ID)

	if !success {
		log.Warn().
			Str("subscriptionID", nfData.SubscriptionID).
			Msg("发送邮件通知失败，可能是订阅已过期或通道不存在")
	}
}
