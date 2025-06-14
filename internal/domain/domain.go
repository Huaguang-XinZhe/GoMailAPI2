// internal/domain/mail.go - 新增业务领域模型
package domain

// EmailAddress 邮件地址（业务概念）
type EmailAddress struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// Email 邮件实体（核心业务模型）
type Email struct {
	ID      string        `json:"id"`
	Subject string        `json:"subject"`
	From    *EmailAddress `json:"from"`
	To      *EmailAddress `json:"to"`
	Date    string        `json:"date"`
	Text    string        `json:"text"`
	HTML    string        `json:"html"`
}
