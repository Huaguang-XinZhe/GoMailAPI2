package types

// ServiceProvider 服务提供商类型
type ServiceProvider string

const (
	ServiceProviderMicrosoft ServiceProvider = "MICROSOFT"
	ServiceProviderGoogle    ServiceProvider = "GOOGLE"
)

// ProtocolType 协议类型
type ProtocolType string

const (
	ProtocolTypeIMAP  ProtocolType = "IMAP"
	ProtocolTypeGraph ProtocolType = "GRAPH"
)

// MailInfo 邮件信息
type MailInfo struct {
	Email           string          `json:"email"`
	ClientID        string          `json:"clientId"`
	RefreshToken    string          `json:"refreshToken"`
	ProtocolType    ProtocolType    `json:"protocolType"`
	ServiceProvider ServiceProvider `json:"serviceProvider"`
}
