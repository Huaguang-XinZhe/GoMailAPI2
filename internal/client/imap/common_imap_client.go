package imap

import (
	"context"
	"errors"
	"fmt"
	"gomailapi2/internal/domain"
	"gomailapi2/internal/utils"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

// CommonImapClient 通用 IMAP 客户端
type CommonImapClient struct {
	config       *ImapConfig
	authProvider AuthProvider

	// 连接管理
	client       *client.Client // 来自 go-imap/client
	isConnected  bool
	isSubscribed bool
	stopChan     chan struct{}
	mu           sync.RWMutex

	// 用于等待 goroutine 完成
	listenerWg sync.WaitGroup

	// 记录 IDLE 通知状态，避免重复 fetch
	lastIdleNotificationTime time.Time
	lastMailboxMessages      uint32 // 记录邮箱中的邮件数量
	lastMailboxRecent        uint32 // 记录最近邮件数量
}

// NewCommonImapClient 创建通用 IMAP 客户端
func NewCommonImapClient(config *ImapConfig, authProvider AuthProvider) *CommonImapClient {
	return &CommonImapClient{
		config:       config,
		authProvider: authProvider,
		isConnected:  false,
		isSubscribed: false,
		stopChan:     make(chan struct{}),
		// sync.RWMutex 是零值可用的，不需要显式初始化。Go 的零值机制确保 mutex 可以直接使用。
	}
}

// Connect 建立持久连接
func (c *CommonImapClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isConnected {
		return nil
	}

	// 连接 IMAP 服务器
	var imapClient *client.Client
	var err error

	if c.config.UseTLS {
		imapClient, err = client.DialTLS(c.config.Host, nil)
	} else {
		imapClient, err = client.Dial(c.config.Host)
	}

	if err != nil {
		return fmt.Errorf("连接 IMAP 服务器失败: %v", err)
	}

	// 获取认证客户端
	saslClient, err := c.authProvider.GetSASLClient()
	if err != nil {
		imapClient.Logout()
		return fmt.Errorf("获取认证客户端失败: %v", err)
	}

	// 进行认证
	if err := imapClient.Authenticate(saslClient); err != nil {
		imapClient.Logout()
		return fmt.Errorf("认证失败: %v", err)
	}

	// 选择邮箱（通常是 INBOX）
	_, err = imapClient.Select("INBOX", false)
	if err != nil {
		imapClient.Logout()
		return fmt.Errorf("选择邮箱失败: %v", err)
	}

	c.client = imapClient
	c.isConnected = true
	return nil
}

// Disconnect 断开连接
func (c *CommonImapClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isConnected {
		return nil
	}

	// 如果正在订阅，先停止订阅
	if c.isSubscribed {
		c.stopSubscription()
	}

	// 重置 Updates 通道，防止死锁
	if c.client != nil {
		c.client.Updates = nil
		err := c.client.Logout()
		c.client = nil
		c.isConnected = false
		if err != nil {
			return fmt.Errorf("登出时出错: %v", err)
		}
	}

	return nil
}

// FetchLatestEmail 获取最新邮件【单独建立连接】
func (c *CommonImapClient) FetchLatestEmail() (*domain.Email, error) {
	return c.fetchLatestEmailFromFolder("inbox")
}

func (c *CommonImapClient) FetchLatestJunkEmail() (*domain.Email, error) {
	return c.fetchLatestEmailFromFolder("junk")
}

// FetchEmailByID 根据邮件 ID 获取邮件详情【单独建立连接】
func (c *CommonImapClient) FetchEmailByID(emailID string) (*domain.Email, error) {
	// 检查是否已连接，如果没有连接则自动连接
	if !c.isConnected {
		if err := c.Connect(); err != nil {
			return nil, fmt.Errorf("建立连接失败: %v", err)
		}
	}

	// 选择收件箱（默认在收件箱中搜索）
	_, err := c.client.Select("INBOX", false)
	if err != nil {
		return nil, fmt.Errorf("选择邮箱失败: %v", err)
	}

	// 根据 Message-ID 搜索邮件
	criteria := imap.NewSearchCriteria()
	criteria.Header.Set("Message-ID", emailID)

	uids, err := c.client.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("搜索邮件失败: %v", err)
	}

	if len(uids) == 0 {
		return nil, fmt.Errorf("未找到 ID 为 %s 的邮件", emailID)
	}

	// 获取第一个匹配的邮件
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids[0])

	section := &imap.BodySectionName{
		Partial: []int{0, 50000}, // 获取前 50kb
	}

	items := []imap.FetchItem{section.FetchItem()}

	messages := make(chan *imap.Message, 1)
	if err := c.client.Fetch(seqSet, items, messages); err != nil {
		return nil, fmt.Errorf("获取邮件失败: %v", err)
	}

	message := <-messages
	if message == nil {
		return nil, errors.New("没有收到邮件内容")
	}

	return parseMail(message, section)
}

// SubscribeNewEmails 订阅新邮件通知
func (c *CommonImapClient) SubscribeNewEmails(ctx context.Context, emailChan chan<- *domain.Email) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isConnected {
		return errors.New("客户端未连接")
	}

	if c.isSubscribed {
		return errors.New("已经在订阅中")
	}

	// 创建接收更新的通道
	updates := make(chan client.Update, 1)
	c.client.Updates = updates

	// 重新创建停止通道
	// 关闭的 channel 不能重复关闭（会 panic）
	// 不能重新打开已关闭的 channel
	// 所以需要创建新的 channel 用于下次订阅
	c.stopChan = make(chan struct{})
	c.isSubscribed = true

	// 启动监听 goroutine，并添加到 WaitGroup
	c.listenerWg.Add(1)
	go c.listenForEmails(ctx, updates, emailChan)

	return nil
}

// UnsubscribeNewEmails 取消订阅新邮件通知
func (c *CommonImapClient) UnsubscribeNewEmails() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isSubscribed {
		return nil
	}

	c.stopSubscription()
	return nil
}

// IsConnected 检查是否已连接
func (c *CommonImapClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// IsSubscribed 检查是否正在订阅
func (c *CommonImapClient) IsSubscribed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isSubscribed
}

// stopSubscription 内部方法，停止订阅（调用时需要已持有锁）
func (c *CommonImapClient) stopSubscription() {
	close(c.stopChan)
	c.isSubscribed = false

	// 等待监听 goroutine 完成，而不是硬编码 sleep
	c.listenerWg.Wait()
}

// listenForEmails 监听新邮件的 goroutine
func (c *CommonImapClient) listenForEmails(ctx context.Context, updates <-chan client.Update, emailChan chan<- *domain.Email) {
	defer func() {
		// 标记 goroutine 完成
		c.listenerWg.Done()

		// 当 goroutine 发生 panic 时，recover() 可以捕获并处理
		// 防止整个程序崩溃
		// 只能在 defer 函数中使用
		// 这里用来保护监听邮件的 goroutine，即使出现意外错误也不会影响主程序
		if r := recover(); r != nil {
			log.Printf("监听邮件时发生 panic: %v", r)
		}
		log.Println("邮件监听 goroutine 已退出")
	}()

	for {
		// 创建停止 IDLE 的通道
		stop := make(chan struct{})

		// 启动 IDLE 命令
		idleDone := make(chan error, 1)
		go func() {
			idleDone <- c.client.Idle(stop, nil)
		}()

		log.Println("IDLE 监听已启动，等待新邮件...")

		select {
		case update := <-updates:
			// 处理不同类型的更新
			switch update := update.(type) {
			case *client.MailboxUpdate:
				log.Println("收到新邮件通知！")

				// 检查是否是重复的 IDLE 通知（在 fetch 之前拦截）
				now := time.Now()
				duration := now.Sub(c.lastIdleNotificationTime)
				// todo 可配置
				if duration < 5*time.Second {
					log.Printf("检测到重复的 IDLE 通知（间隔 %.2f 秒），跳过 fetch", duration.Seconds())

					// 仍然需要停止 IDLE 命令
					close(stop)
					if err := <-idleDone; err != nil {
						log.Printf("IDLE 命令结束时出错: %v", err)
					}
					log.Println("继续监听新邮件...")
					continue // 跳过后续的 fetch 操作
				}

				// 检查邮箱状态变化
				if update.Mailbox.Messages == c.lastMailboxMessages && update.Mailbox.Recent == c.lastMailboxRecent {
					log.Printf("邮箱状态未变化（Messages: %d, Recent: %d），跳过 fetch",
						update.Mailbox.Messages, update.Mailbox.Recent)

					// 仍然需要停止 IDLE 命令
					close(stop)
					if err := <-idleDone; err != nil {
						log.Printf("IDLE 命令结束时出错: %v", err)
					}
					log.Println("继续监听新邮件...")
					continue // 跳过后续的 fetch 操作
				}

				// 更新通知时间和邮箱状态
				c.lastIdleNotificationTime = now
				c.lastMailboxMessages = update.Mailbox.Messages
				c.lastMailboxRecent = update.Mailbox.Recent

				log.Printf("邮箱状态变化：Messages: %d, Recent: %d", update.Mailbox.Messages, update.Mailbox.Recent)

				// 停止 IDLE 命令
				close(stop)
				// 等待 IDLE 命令结束
				if err := <-idleDone; err != nil {
					log.Printf("IDLE 命令结束时出错: %v", err)
				}

				// 获取最新邮件
				log.Println("正在获取最新邮件...")
				email, err := c.fetchEmailBySequenceNumber(update.Mailbox.Messages)
				if err != nil {
					log.Printf("获取最新邮件失败: %v", err)
				} else if email != nil {
					log.Printf("成功获取邮件: %s", email.Subject)
					select {
					case emailChan <- email:
						log.Printf("新邮件已发送到通道: %s", email.Subject)
					case <-ctx.Done():
						log.Println("上下文已取消，停止发送邮件到通道")
						return
					case <-c.stopChan:
						log.Println("收到停止信号，停止发送邮件到通道")
						return
					}
				} else {
					log.Println("没有获取到新邮件")
				}

				// 继续下一轮监听
				log.Println("继续监听新邮件...")

			default:
				log.Printf("收到未知类型的更新: %T", update)
				close(stop)
				if err := <-idleDone; err != nil {
					log.Printf("IDLE 命令结束时出错: %v", err)
				}
			}

		case err := <-idleDone:
			// IDLE 命令自行结束
			if err != nil {
				log.Printf("IDLE 命令意外结束: %v", err)
				// 如果是网络错误等，可以考虑重新连接
				return
			}
			log.Println("IDLE 命令正常结束，重新启动...")

		case <-ctx.Done():
			// 上下文取消
			log.Println("收到上下文取消信号，停止 IDLE 监听")
			close(stop)
			<-idleDone
			return

		case <-c.stopChan:
			// 停止订阅
			log.Println("收到停止订阅信号，停止 IDLE 监听")
			close(stop)
			<-idleDone
			return
		}
	}
}

// fetchLatestEmailFromFolder 从指定文件夹获取最新邮件
func (c *CommonImapClient) fetchLatestEmailFromFolder(folderName string) (*domain.Email, error) {
	// 检查是否已连接，如果没有连接则自动连接
	if !c.isConnected {
		if err := c.Connect(); err != nil {
			return nil, fmt.Errorf("建立连接失败: %v", err)
		}
	}

	// 选择指定文件夹
	mbox, err := c.client.Select(folderName, false)
	if err != nil {
		return nil, fmt.Errorf("选择文件夹 %s 失败: %v", folderName, err)
	}

	if mbox.Messages == 0 {
		log.Printf("文件夹 %s 中没有邮件", folderName)
		return nil, nil
	}

	return c.fetchEmailBySequenceNumber(mbox.Messages)
}

// fetchEmailBySequenceNumber 通过序列号获取邮件
func (c *CommonImapClient) fetchEmailBySequenceNumber(sequenceNumber uint32) (*domain.Email, error) {
	// 获取指定序列号的邮件
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(sequenceNumber)

	section := &imap.BodySectionName{
		Partial: []int{0, 50000}, // 获取前 50kb
	}

	items := []imap.FetchItem{section.FetchItem()}

	messages := make(chan *imap.Message, 1)
	// if err := c.client.UidFetch(seqSet, items, messages); err != nil {
	if err := c.client.Fetch(seqSet, items, messages); err != nil {
		return nil, fmt.Errorf("获取邮件失败: %v", err)
	}

	message := <-messages
	if message == nil {
		return nil, errors.New("没有收到邮件内容")
	}

	return parseMail(message, section)
}

// parseMail 解析邮件内容（通用函数）
func parseMail(message *imap.Message, section *imap.BodySectionName) (*domain.Email, error) {
	literal := message.GetBody(section)

	if literal == nil {
		return nil, errors.New("邮件 Literal 为空")
	}

	// 创建邮件阅读器
	mr, err := mail.CreateReader(literal)
	if err != nil {
		return nil, errors.New("创建邮件阅读器失败")
	}

	// 解析邮件基本信息
	header := mr.Header

	var date string
	var from, to *domain.EmailAddress
	var subject string

	// 记录 ID
	messageID := header.Get("Message-ID")
	// 去掉两端的尖括号
	messageID = strings.Trim(messageID, "<>")
	log.Printf("邮件 ID: %s", messageID)

	if d, err := header.Date(); err != nil {
		return nil, errors.New("获取邮件日期失败")
	} else {
		date = d.Format(time.RFC3339)
	}

	if f, err := header.AddressList("From"); err != nil {
		return nil, errors.New("获取邮件发件人失败")
	} else {
		from = utils.CleanEmailAddress(f[0])
	}

	if t, err := header.AddressList("To"); err != nil {
		return nil, errors.New("获取邮件收件人失败")
	} else {
		to = utils.CleanEmailAddress(t[0])
	}

	if s, err := header.Subject(); err != nil {
		return nil, errors.New("获取邮件主题失败")
	} else {
		subject = s
	}

	email := &domain.Email{
		ID:      messageID,
		Date:    date,
		From:    from,
		To:      to,
		Subject: subject,
	}

	// 处理邮件正文
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			return email, errors.New("读取邮件正文失败")
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			// 这是邮件的正文部分
			body, _ := io.ReadAll(p.Body)
			contentType, _, _ := h.ContentType()

			if contentType == "text/plain" {
				email.Text = string(body)
			} else if contentType == "text/html" {
				email.HTML = string(body)
			}
		case *mail.AttachmentHeader:
			// 这是附件部分
			filename, _ := h.Filename()
			log.Printf("发现附件: %s", filename)
		}
	}

	return email, nil
}
