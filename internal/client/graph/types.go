package graph

import (
	"gomailapi2/internal/domain"
	"time"
)

type NewEmailResponse struct {
	Value []EmailData `json:"value"`
}

type FindEmailResponse struct {
	*EmailData
}

// EmailData 表示从 Microsoft Graph API 返回的单个邮件数据
type EmailData struct {
	Subject          string `json:"subject"`
	ReceivedDateTime string `json:"receivedDateTime"`
	BodyPreview      string `json:"bodyPreview"`
	Body             struct {
		Content string `json:"content"`
	} `json:"body"`
	From struct {
		EmailAddress domain.EmailAddress `json:"emailAddress"`
	} `json:"from"`
	ToRecipients []ToRecipient `json:"toRecipients"`
}

type ToRecipient struct {
	EmailAddress domain.EmailAddress `json:"emailAddress"`
}

// Subscription Graph 订阅结构体
type Subscription struct {
	Resource           string    `json:"resource"`
	ChangeType         string    `json:"changeType"`
	NotificationURL    string    `json:"notificationUrl"`
	ExpirationDateTime time.Time `json:"expirationDateTime"`
	// ClientState        string    `json:"clientState,omitempty"`
}

// SubscriptionResponse 订阅响应结构体
type SubscriptionResponse struct {
	ID string `json:"id"`
}
