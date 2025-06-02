package types

// ServiceProvider 服务提供商类型
type ServiceProvider string

const (
	ServiceProviderMicrosoft ServiceProvider = "microsoft"
	ServiceProviderGoogle    ServiceProvider = "google"
)

// ProtocolType 协议类型
type ProtocolType string

const (
	ProtocolTypeIMAP  ProtocolType = "imap"
	ProtocolTypeGraph ProtocolType = "graph"
)

// MailInfo 邮件信息
type MailInfo struct {
	Email           string          `json:"email"`
	ClientID        string          `json:"clientId"`
	RefreshToken    string          `json:"refreshToken"`
	ProtoType       ProtocolType    `json:"protoType"`
	ServiceProvider ServiceProvider `json:"serviceProvider"`
}
