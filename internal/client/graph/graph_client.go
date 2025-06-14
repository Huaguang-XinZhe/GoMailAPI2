package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gomailapi2/internal/domain"
	"gomailapi2/internal/utils"
	"io"
	"net/http"
	"time"
)

const (
	// API 端点
	graphBaseURL = "https://graph.microsoft.com/v1.0"
	// // 通用消息端点（所有邮件）
	// messagesEndpoint = graphBaseURL + "/me/messages"
	// 收件箱端点
	inboxEndpoint = graphBaseURL + "/me/mailFolders/Inbox/messages"
	// 订阅端点
	subscriptionsEndpoint = graphBaseURL + "/subscriptions"
	// 选择字段
	selectFields = "subject,from,toRecipients,receivedDateTime,bodyPreview,body"
	// 订阅过期时间（分钟）
	SubscriptionTimeoutMinutes = 5
)

// CreateSubscription 创建 Graph 订阅
func CreateSubscription(ctx context.Context, accessToken string, notificationURL string) (*SubscriptionResponse, error) {
	if accessToken == "" {
		return nil, errors.New("访问令牌不能为空")
	}
	if notificationURL == "" {
		return nil, errors.New("通知 URL 不能为空")
	}

	// 创建订阅对象
	subscription := Subscription{
		Resource:        "me/mailFolders('Inbox')/messages",
		ChangeType:      "created",
		NotificationURL: notificationURL,
		// 订阅过期时间比 SSE 超时时间长，给通知留出缓冲时间
		ExpirationDateTime: time.Now().Add(SubscriptionTimeoutMinutes * time.Minute),
		// ClientState:        "GoMailAPI-Secret-State",
	}

	// 序列化为 JSON
	jsonData, err := json.Marshal(subscription)
	if err != nil {
		return nil, fmt.Errorf("序列化订阅数据失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, subscriptionsEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建订阅请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送订阅请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取订阅响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("创建订阅失败 (状态码: %d): %s", resp.StatusCode, string(body))
	}

	var subscriptionResp SubscriptionResponse
	if err := json.Unmarshal(body, &subscriptionResp); err != nil {
		return nil, fmt.Errorf("解析订阅响应失败: %w", err)
	}

	return &SubscriptionResponse{
		ID: subscriptionResp.ID,
	}, nil
}

// DeleteSubscription 删除 Graph 订阅
func DeleteSubscription(ctx context.Context, accessToken string, subscriptionID string) error {
	if accessToken == "" {
		return errors.New("访问令牌不能为空")
	}
	if subscriptionID == "" {
		return errors.New("订阅 ID 不能为空")
	}

	// 构建删除 URL
	deleteURL := fmt.Sprintf("%s/%s", subscriptionsEndpoint, subscriptionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return fmt.Errorf("创建删除请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送删除请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("删除订阅失败 (状态码: %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetLatestEmail 获取最新的一封邮件
func GetLatestEmail(ctx context.Context, accessToken string) (*domain.Email, error) {
	if accessToken == "" {
		return nil, errors.New("访问令牌不能为空")
	}

	// 使用收件箱端点构建请求 URL，确保只获取收件箱的邮件
	requestURL := buildEmailRequestURL(inboxEndpoint, 1)

	return getEmailFromURL(ctx, accessToken, requestURL)
}

// GetEmailByID 根据邮件 ID 获取邮件详情
func GetEmailByID(ctx context.Context, accessToken string, emailID string) (*domain.Email, error) {
	if accessToken == "" {
		return nil, errors.New("访问令牌不能为空")
	}
	if emailID == "" {
		return nil, errors.New("邮件 ID 不能为空")
	}

	// 构建请求 URL
	requestURL := fmt.Sprintf("%s/me/messages/%s?$select=%s", graphBaseURL, emailID, selectFields)

	return getEmailFromURL(ctx, accessToken, requestURL)
}

// GetLatestEmailFromJunk 从垃圾箱获取最新的一封邮件
func GetLatestEmailFromJunk(ctx context.Context, accessToken string) (*domain.Email, error) {
	if accessToken == "" {
		return nil, errors.New("访问令牌不能为空")
	}

	// 使用通用方法构建请求 URL
	folderEndpoint := fmt.Sprintf("%s/me/mailFolders/%s/messages", graphBaseURL, "junkemail")
	requestURL := buildEmailRequestURL(folderEndpoint, 1)

	return getEmailFromURL(ctx, accessToken, requestURL)
}

// buildEmailRequestURL 构建邮件请求 URL 的通用方法
// 注意：不使用 $orderby 排序，因为 Microsoft Graph API 的排序功能在某些情况下会返回错误的结果
// API 默认会按最新的邮件在前的顺序返回，这正是我们需要的
func buildEmailRequestURL(endpoint string, count int) string {
	if count == 1 {
		return fmt.Sprintf("%s?$top=1&$select=%s", endpoint, selectFields)
	}
	return fmt.Sprintf("%s?$top=%d&$select=%s", endpoint, count, selectFields)
}

// getEmailFromURL 从指定 URL 获取单封邮件的通用方法
func getEmailFromURL(ctx context.Context, accessToken, requestURL string) (*domain.Email, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("获取邮件失败 (状态码: %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 判断响应类型：检查是否包含 "value" 字段
	if bytes.Contains(body, []byte(`"value"`)) {
		// 这是邮件列表响应 (GetLatestEmail)
		var response NewEmailResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("解析邮件列表响应失败: %w", err)
		}

		if len(response.Value) == 0 {
			return nil, nil // 没有找到邮件时返回 nil，不是错误
		}

		return convertToEmail(response.Value[0]), nil
	} else {
		// 这是单个邮件响应 (GetEmailByID)
		var response FindEmailResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("解析单个邮件响应失败: %w", err)
		}

		if response.EmailData == nil {
			return nil, nil // 没有找到邮件时返回 nil，不是错误
		}

		return convertToEmail(*response.EmailData), nil
	}
}

// convertToEmail 将 API 响应中的邮件数据转换为 Email 结构体
func convertToEmail(emailData EmailData) *domain.Email {
	var toRecipient *domain.EmailAddress
	if len(emailData.ToRecipients) > 0 {
		toRecipient = &emailData.ToRecipients[0].EmailAddress
	}

	return &domain.Email{
		ID:      emailData.ID,
		Subject: emailData.Subject,
		From:    utils.CleanEmailAddress(&emailData.From.EmailAddress),
		To:      utils.CleanEmailAddress(toRecipient),
		Date:    emailData.ReceivedDateTime,
		Text:    emailData.BodyPreview,
		HTML:    emailData.Body.Content,
	}
}
