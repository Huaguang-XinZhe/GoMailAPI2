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

// UnsubscribeMailRequest 纯粹取消订阅
type UnsubscribeMailRequest struct {
	SubScribeID string `json:"subScribeID"` // 订阅 ID
}

// RefreshTokenRequest 纯粹刷新 refreshToken
type RefreshTokenRequest struct {
	MailInfo *types.MailInfo `json:"mailInfo"` // 邮箱信息
}

// BatchRefreshTokenRequest 批量刷新 refreshToken
type BatchRefreshTokenRequest struct {
	MailInfos []*types.MailInfo `json:"mailInfos"` // 需要刷新的 Token 列表
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

// DetectProtocolTypeRequest 检测协议类型请求
type DetectProtocolTypeRequest struct {
	MailInfo *types.MailInfo `json:"mailInfo"` // 邮箱信息（不包含 protocolType）
}

// DetectProtocolTypeResponse 检测协议类型响应
type DetectProtocolTypeResponse struct {
	ProtocolType types.ProtocolType `json:"protocolType"` // 检测到的协议类型
}

// BatchDetectProtocolTypeRequest 批量检测协议类型请求
type BatchDetectProtocolTypeRequest struct {
	MailInfos []*types.MailInfo `json:"mailInfos"` // 需要检测的邮箱信息列表
}

// BatchDetectProtocolTypeResult 批量检测协议类型结果项
type BatchDetectProtocolTypeResult struct {
	Email        string             `json:"email"`                  // 邮箱地址
	ProtocolType types.ProtocolType `json:"protocolType,omitempty"` // 检测到的协议类型（成功时）
	Error        string             `json:"error,omitempty"`        // 错误信息（失败时）
}

// BatchDetectProtocolTypeResponse 批量检测协议类型的数据部分
type BatchDetectProtocolTypeResponse struct {
	SuccessCount int                             `json:"successCount"` // 成功检测的数量
	FailCount    int                             `json:"failCount"`    // 失败检测的数量
	Results      []BatchDetectProtocolTypeResult `json:"results"`      // 详细结果列表
}
