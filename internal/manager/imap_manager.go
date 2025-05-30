package manager

import (
	"context"
	"fmt"
	"gomailapi2/internal/client/imap/outlook"
	"gomailapi2/internal/domain"
	"math/rand"
	"time"

	"github.com/rs/zerolog/log"
)

// generateSubscriptionID 生成唯一的订阅 ID
func generateSubscriptionID() string {
	// 使用时间戳 + 随机数生成唯一 ID
	timestamp := time.Now().UnixNano()
	random := rand.Int63()
	return fmt.Sprintf("imap_%d_%d", timestamp, random)
}

// ImapSubscription IMAP 订阅信息
type ImapSubscription struct {
	ID         string                     // 订阅 ID
	Email      string                     // 邮箱地址
	Client     *outlook.OutlookImapClient // IMAP 客户端
	EmailChan  chan *domain.Email         // 邮件通道
	StopCtx    context.Context            // 停止上下文
	StopCancel context.CancelFunc         // 停止函数
	CreatedAt  time.Time                  // 创建时间
}

// ImapSubscriptionManager IMAP 订阅管理器
type ImapSubscriptionManager struct {
	subscriptions map[string]*ImapSubscription // key: subscriptionID
}

// NewImapSubscriptionManager 创建新的 IMAP 订阅管理器
func NewImapSubscriptionManager() *ImapSubscriptionManager {
	return &ImapSubscriptionManager{
		subscriptions: make(map[string]*ImapSubscription),
	}
}

// CreateSubscription 创建新的 IMAP 订阅
func (m *ImapSubscriptionManager) CreateSubscription(imapClient *outlook.OutlookImapClient, email string) (*ImapSubscription, error) {
	// 生成唯一的订阅 ID
	subscriptionID := generateSubscriptionID()

	// 创建邮件通道
	emailChan := make(chan *domain.Email, 1)

	// 创建停止上下文（3 分钟超时）
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)

	subscription := &ImapSubscription{
		ID:         subscriptionID,
		Email:      email,
		Client:     imapClient,
		EmailChan:  emailChan,
		StopCtx:    ctx,
		StopCancel: cancel,
		CreatedAt:  time.Now(),
	}

	// 存储订阅
	m.subscriptions[subscriptionID] = subscription

	log.Info().
		Str("subscriptionID", subscriptionID).
		Str("email", email).
		Msg("创建 IMAP 订阅")

	return subscription, nil
}

// CancelSubscription 取消指定的订阅
func (m *ImapSubscriptionManager) CancelSubscription(subscriptionID string) error {
	subscription, exists := m.subscriptions[subscriptionID]
	if !exists {
		log.Warn().
			Str("subscriptionID", subscriptionID).
			Msg("尝试取消不存在的 IMAP 订阅")
		return fmt.Errorf("订阅 ID %s 不存在", subscriptionID)
	}

	// 停止订阅
	subscription.StopCancel()

	// 取消 IMAP 订阅
	if subscription.Client.IsSubscribed() {
		if err := subscription.Client.UnsubscribeNewEmails(); err != nil {
			log.Error().
				Err(err).
				Str("subscriptionID", subscriptionID).
				Msg("取消 IMAP 订阅失败")
		}
	}

	// 断开连接
	if subscription.Client.IsConnected() {
		if err := subscription.Client.Disconnect(); err != nil {
			log.Error().
				Err(err).
				Str("subscriptionID", subscriptionID).
				Msg("断开 IMAP 连接失败")
		}
	}

	// 关闭邮件通道
	close(subscription.EmailChan)

	// 从管理器中移除
	delete(m.subscriptions, subscriptionID)

	log.Info().
		Str("subscriptionID", subscriptionID).
		Str("email", subscription.Email).
		Msg("成功取消 IMAP 订阅")

	return nil
}

// StartSubscription 启动订阅监听
func (m *ImapSubscriptionManager) StartSubscription(subscription *ImapSubscription) error {
	// 建立连接
	log.Info().
		Str("subscriptionID", subscription.ID).
		Msg("正在连接到 IMAP 服务器...")

	if err := subscription.Client.Connect(); err != nil {
		log.Error().
			Err(err).
			Str("subscriptionID", subscription.ID).
			Msg("IMAP 连接失败")
		return fmt.Errorf("连接失败: %v", err)
	}

	log.Info().
		Str("subscriptionID", subscription.ID).
		Msg("IMAP 连接成功")

	// 开始订阅新邮件
	log.Info().
		Str("subscriptionID", subscription.ID).
		Msg("正在启动邮件订阅...")

	if err := subscription.Client.SubscribeNewEmails(subscription.StopCtx, subscription.EmailChan); err != nil {
		log.Error().
			Err(err).
			Str("subscriptionID", subscription.ID).
			Msg("订阅新邮件失败")
		return fmt.Errorf("订阅新邮件失败: %v", err)
	}

	log.Info().
		Str("subscriptionID", subscription.ID).
		Msg("开始监听新邮件...")

	return nil
}
