package manager

import (
	"github.com/rs/zerolog/log"
)

// NotificationManager 通知管理器
type NotificationManager struct {
	channels map[string]chan string // key: subscriptionID（emailID）
}

// NewNotificationManager 创建新的通知管理器
func NewNotificationManager() *NotificationManager {
	return &NotificationManager{
		channels: make(map[string]chan string),
	}
}

// RegisterChannel 注册一个新的邮件通知通道
func (nm *NotificationManager) RegisterChannel(subscriptionID string) chan string {
	notifyChan := make(chan string, 1)

	nm.channels[subscriptionID] = notifyChan

	log.Info().
		Str("subscriptionID", subscriptionID).
		Msg("注册新的邮件通知通道")

	return notifyChan
}

// SendNotification 发送邮件通知到指定的订阅通道
func (nm *NotificationManager) SendNotification(subscriptionID, emailID string) bool {
	channel, exists := nm.channels[subscriptionID]
	if !exists {
		log.Warn().
			Str("subscriptionID", subscriptionID).
			Msg("未找到对应的通知通道")
		return false
	}

	select {
	case channel <- emailID:
		log.Info().
			Str("subscriptionID", subscriptionID).
			Msg("成功发送邮件通知")
		return true
	default:
		log.Warn().
			Str("subscriptionID", subscriptionID).
			Msg("通知通道已满，丢弃邮件通知")
		return false
	}
}

// RemoveChannel 移除指定的通知通道
func (nm *NotificationManager) RemoveChannel(subscriptionID string) {
	if channel, exists := nm.channels[subscriptionID]; exists {
		close(channel)
		delete(nm.channels, subscriptionID)

		log.Info().
			Str("subscriptionID", subscriptionID).
			Msg("移除邮件通知通道")
	}
}
