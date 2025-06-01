package dto

import (
	"gomailapi2/internal/types"
)

// GetNewJunkMailRequest 获取垃圾箱最新一封邮件请求
type GetNewJunkMailRequest struct {
	MailInfo *types.MailInfo `json:"mailInfo"` // 邮箱信息
}

// FindMailRequest 查找邮件请求
type FindMailRequest struct {
	MailInfo *types.MailInfo `json:"mailInfo"` // 邮箱信息
}

// GetNewMailRequest 获取最新一封邮件请求
type GetNewMailRequest struct {
	MailInfo      *types.MailInfo `json:"mailInfo"`                // 邮箱信息
	RefreshNeeded bool            `json:"refreshNeeded,omitempty"` // 是否需要刷新 refreshToken
}

// SubscribeMailRequest 订阅 -> 获取新到的一封邮件
type SubscribeMailRequest struct {
	MailInfo      *types.MailInfo `json:"mailInfo"`                // 新邮箱的信息
	RefreshNeeded bool            `json:"refreshNeeded,omitempty"` // 是否需要刷新 refreshToken
}

// ! 响应结构可能不需要,因为是通过 SSE 返回

// UnsubscribeMailRequest 纯粹取消订阅
type UnsubscribeMailRequest struct {
	SubScribeID string `json:"subScribeID"` // 订阅 ID
}

// RefreshTokenItem 需要刷新的 Token 项目
type RefreshTokenItem struct {
	Email           string                `json:"email,omitempty"`           // 邮箱地址
	RefreshToken    string                `json:"refreshToken"`              // 当前的 refreshToken
	ClientID        string                `json:"clientID,omitempty"`        // 客户端 ID
	ServiceProvider types.ServiceProvider `json:"serviceProvider,omitempty"` // 服务提供商（microsoft/google）
}

// RefreshTokenRequest 纯粹刷新 refreshToken
type RefreshTokenRequest struct {
	*RefreshTokenItem
}

// RefreshTokenData 刷新 Token 的数据部分
type RefreshTokenData struct {
	NewRefreshToken string `json:"newRefreshToken"` // 新的 refreshToken
}

// BatchRefreshTokenRequest 批量刷新 refreshToken
type BatchRefreshTokenRequest struct {
	Tokens []*RefreshTokenItem `json:"tokens"` // 需要刷新的 Token 列表
}

// BatchRefreshTokenData 批量刷新的数据部分
type BatchRefreshTokenData struct {
	SuccessCount int                  `json:"successCount"` // 成功刷新的数量
	FailCount    int                  `json:"failCount"`    // 失败刷新的数量
	Results      []BatchRefreshResult `json:"results"`      // 详细结果列表
}

// BatchRefreshResult 批量刷新结果项
type BatchRefreshResult struct {
	Email           string `json:"email"`                     // 邮箱地址
	NewRefreshToken string `json:"newRefreshToken,omitempty"` // 新的 refreshToken
	Error           string `json:"error,omitempty"`           // 错误信息（失败时）
}
