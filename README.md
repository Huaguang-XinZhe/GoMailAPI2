# GoMailAPI2

基于 Go 的邮件 API 服务器，支持 IMAP 和 Microsoft Graph 协议的邮件获取和订阅功能。

## 功能特性

- 🔐 **OAuth2 Token 管理**: 自动刷新 access token 和 refresh token
- 📧 **多协议支持**: 支持 IMAP 和 Microsoft Graph API
- 🏪 **多服务商支持**: 支持 Microsoft 和 Google
- 💾 **智能缓存**: Redis 缓存 access token，减少 API 调用
- 🗄️ **数据持久化**: PostgreSQL 存储 refresh token
- 📝 **结构化日志**: 使用 zerolog 记录详细日志
- ⚙️ **配置管理**: 使用 viper 管理配置

## 技术架构

```
cmd/
├── mailserver/           # 程序入口
│   ├── main.go          # 初始化各组件
│   └── config.go        # 读取端口、数据库连接等配置
internal/
├── api/                 # HTTP接口
├── service/             # 核心业务逻辑
│   ├── token.go        # Token服务
│   ├── imap.go         # IMAP服务
│   ├── graph.go        # Graph服务
│   └── mail.go         # 邮件服务协调器
└── infra/               # 技术基础设施
    ├── cache/           # Redis操作
    ├── db/              # 数据库操作
    └── oauth/           # 调用第三方OAuth的代码
pkg/
└── utils/               # 通用工具函数
```

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 配置环境

复制 `config.yaml` 文件并修改配置：

```yaml
server:
  host: "localhost"
  port: "8080"

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "your_password"
  dbname: "gomailapi"
  sslmode: "disable"

redis:
  host: "localhost"
  port: "6379"
  password: ""
  db: 0

oauth:
  microsoft:
    client_id: "your_microsoft_client_id"
    client_secret: "your_microsoft_client_secret"
  google:
    client_id: "your_google_client_id"
    client_secret: "your_google_client_secret"
```

### 3. 启动服务

```bash
go run cmd/mailserver/main.go cmd/mailserver/config.go
```

## API 接口

### 获取新邮件

**POST** `/api/v1/mail/new`

```json
{
  "email": "user@example.com",
  "clientId": "your_client_id",
  "refreshToken": "your_refresh_token",
  "protoType": "imap",
  "serviceProvider": "microsoft",
  "refreshRequired": false
}
```

### 订阅邮件

**POST** `/api/v1/mail/subscribe`

```json
{
  "email": "user@example.com",
  "clientId": "your_client_id",
  "refreshToken": "your_refresh_token",
  "protoType": "graph",
  "serviceProvider": "microsoft",
  "refreshRequired": true
}
```

## Token 管理机制

### IMAP 协议

1. 如果 `refreshRequired` 为 true，调用 `getToken(includeScope=false)` 获取新 token
2. 新的 refresh token 会立即发送给客户端并更新数据库
3. Access token 缓存到 Redis
4. 后续请求优先从缓存获取，过期则重新获取

### Graph 协议

1. 如果 `refreshRequired` 为 true，同时调用：
   - `getToken(includeScope=false)` 获取 refresh token
   - `getToken(includeScope=true)` 获取 access token
2. 缓存和数据库更新逻辑与 IMAP 相同

## 支持的协议和服务商

| 协议  | Microsoft | Google |
| ----- | --------- | ------ |
| IMAP  | ✅        | ✅     |
| Graph | ✅        | ❌     |

## 环境要求

- Go 1.21+
- PostgreSQL 12+
- Redis 6+

## 开发说明

项目使用了以下技术栈：

- **Web 框架**: Gin
- **配置管理**: Viper
- **日志**: Zerolog
- **数据库 ORM**: GORM
- **缓存**: Redis
- **OAuth2**: 原生 HTTP 客户端

## 许可证

MIT License
