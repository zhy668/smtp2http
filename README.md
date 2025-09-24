SMTP2HTTP (email-to-web)
========================

[English](#english) | [中文](#中文)

## English

smtp2http is a simple smtp server that resends the incoming email to the configured web endpoint (webhook) as a basic http post request.

Dev 
===
- `go mod vendor`
- `go build`

Dev with Docker
==============
Locally :
- `go mod vendor`
- `docker build -f Dockerfile.dev -t smtp2http-dev .`
- `docker run -p 25:25 smtp2http-dev --timeout.read=50 --timeout.write=50 --webhook=http://some.hook/api`

Or build it as it comes from the repo :
- `docker build -t smtp2http .`
- `docker run -p 25:25 smtp2http --timeout.read=50 --timeout.write=50 --webhook=http://some.hook/api`

The `timeout` options are of course optional but make it easier to test in local with `telnet localhost 25`
Here is a telnet example payload : 
```
HELO zeus
# smtp answer

MAIL FROM:<email@from.com>
# smtp answer

RCPT TO:<youremail@example.com>
# smtp answer

DATA
your mail content
.

```

Docker (production)
=====
**Docker images arn't available online for now**
**See "Dev with Docker" above**
- `docker run -p 25:25 smtp2http --webhook=http://some.hook/api`

Native usage
=====
`smtp2http --listen=:25 --webhook=http://localhost:8080/api/smtp-hook`
`smtp2http --help`

Cloud-Mail Integration with Security Features
============================================
This fork adds comprehensive security features and cloud-mail inbound API authentication:

## Basic Usage
```bash
# Simple usage with cloud-mail
smtp2http --listen=:25 --webhook=https://your-domain.com/api/inbound --inbound-key=your-secret-api-key
```

## Advanced Security Configuration
```bash
# Production-ready configuration with security features
smtp2http \
  --listen=:25 \
  --webhook=https://your-domain.com/api/inbound \
  --inbound-key=your-secret-api-key \
  --allowed-domains=yourdomain.com,anotherdomain.com \
  --strict-spf=true \
  --rate-limit=10 \
  --spam-keywords="viagra,lottery,urgent,winner,free money" \
  --forbidden-types=exe,bat,cmd,com,pif,scr,vbs,js,jar,msi \
  --max-attach-size=10485760 \
  --blacklist-domains=spam-domain.com,malicious-site.org
```

## Security Parameters

### Authentication & Access Control
- `--inbound-key`: API key for cloud-mail authentication (adds X-Inbound-Key header)
- `--allowed-domains`: Comma-separated list of allowed recipient domains (empty = allow all)
- `--blacklist-domains`: Comma-separated list of blacklisted sender domains

### SPF Verification
- `--strict-spf`: Reject emails that fail SPF verification (default: false)

### Rate Limiting
- `--rate-limit`: Maximum emails per minute per sender IP (default: 60)

### Spam Protection
- `--spam-keywords`: Comma-separated list of spam keywords to block
- `--forbidden-types`: Comma-separated list of forbidden attachment file extensions
- `--max-attach-size`: Maximum attachment size in bytes (default: 10MB)

## Security Features

### Multi-Layer Protection
1. **Rate Limiting**: Prevents email bombing attacks
2. **Domain Validation**: Controls which domains can receive emails
3. **SPF Verification**: Validates sender authorization
4. **Content Filtering**: Blocks emails with spam keywords
5. **Attachment Security**: Restricts dangerous file types and sizes
6. **Sender Blacklisting**: Blocks emails from known malicious domains

### Logging & Monitoring
- Detailed security event logging
- Email acceptance/rejection reasons
- Spam score calculation
- Client IP tracking
- SPF verification results

### Example Security Scenarios

**High Security Mode** (Recommended for production):
```bash
smtp2http \
  --listen=:25 \
  --webhook=https://your-domain.com/api/inbound \
  --inbound-key=your-secret-key \
  --allowed-domains=yourdomain.com \
  --strict-spf=true \
  --rate-limit=5 \
  --spam-keywords="viagra,lottery,urgent,winner" \
  --max-attach-size=5242880
```

**Medium Security Mode** (Balanced):
```bash
smtp2http \
  --listen=:25 \
  --webhook=https://your-domain.com/api/inbound \
  --inbound-key=your-secret-key \
  --allowed-domains=yourdomain.com \
  --rate-limit=10 \
  --spam-keywords="viagra,lottery"
```

**Development Mode** (Minimal restrictions):
```bash
smtp2http \
  --listen=:25 \
  --webhook=https://your-domain.com/api/inbound \
  --inbound-key=your-secret-key \
  --rate-limit=100
```

Contribution
============
Original repo from @alash3al
Thanks to @aranajuan

---

## 中文

SMTP2HTTP 是一个简单的 SMTP 服务器，它将接收到的邮件转发到配置的 Web 端点（webhook）作为基本的 HTTP POST 请求。

## 开发环境
- `go mod vendor`
- `go build`

## Docker 开发环境
本地开发：
- `go mod vendor`
- `docker build -f Dockerfile.dev -t smtp2http-dev .`
- `docker run -p 25:25 smtp2http-dev --timeout.read=50 --timeout.write=50 --webhook=http://some.hook/api`

或者直接从仓库构建：
- `docker build -t smtp2http .`
- `docker run -p 25:25 smtp2http --timeout.read=50 --timeout.write=50 --webhook=http://some.hook/api`

`timeout` 选项是可选的，但在本地使用 `telnet localhost 25` 测试时很有用。

## 本地使用
```bash
smtp2http --listen=:25 --webhook=http://localhost:8080/api/smtp-hook
smtp2http --help
```

## Cloud-Mail 集成与安全功能
此分支添加了全面的安全功能和 cloud-mail 入站 API 认证：

### 基础用法
```bash
# 与 cloud-mail 的简单用法
smtp2http --listen=:25 --webhook=https://your-domain.com/api/inbound --inbound-key=your-secret-api-key
```

### 高级安全配置
```bash
# 生产环境推荐的安全配置
smtp2http \
  --listen=:25 \
  --webhook=https://your-domain.com/api/inbound \
  --inbound-key=your-secret-api-key \
  --allowed-domains=yourdomain.com,anotherdomain.com \
  --strict-spf=true \
  --rate-limit=10 \
  --spam-keywords="viagra,lottery,urgent,winner,free money" \
  --forbidden-types=exe,bat,cmd,com,pif,scr,vbs,js,jar,msi \
  --max-attach-size=10485760 \
  --blacklist-domains=spam-domain.com,malicious-site.org
```

### 安全参数说明

#### 认证与访问控制
- `--inbound-key`: cloud-mail 认证的 API 密钥（添加 X-Inbound-Key 头部）
- `--allowed-domains`: 允许的收件人域名列表，逗号分隔（空值 = 允许所有）
- `--blacklist-domains`: 黑名单发送者域名列表，逗号分隔

#### SPF 验证
- `--strict-spf`: 拒绝 SPF 验证失败的邮件（默认：false）

#### 速率限制
- `--rate-limit`: 每个发送者 IP 每分钟最大邮件数（默认：60）

#### 垃圾邮件防护
- `--spam-keywords`: 要阻止的垃圾邮件关键词，逗号分隔
- `--forbidden-types`: 禁止的附件文件扩展名，逗号分隔
- `--max-attach-size`: 最大附件大小（字节，默认：10MB）

### 安全功能

#### 多层防护
1. **速率限制**：防止邮件轰炸攻击
2. **域名验证**：控制哪些域名可以接收邮件
3. **SPF 验证**：验证发送者授权
4. **内容过滤**：阻止包含垃圾邮件关键词的邮件
5. **附件安全**：限制危险文件类型和大小
6. **发送者黑名单**：阻止来自已知恶意域名的邮件

#### 日志记录与监控
- 详细的安全事件日志
- 邮件接受/拒绝原因
- 垃圾邮件评分计算
- 客户端 IP 跟踪
- SPF 验证结果

### 安全配置示例

**高安全模式**（推荐生产环境）：
```bash
smtp2http \
  --listen=:25 \
  --webhook=https://your-domain.com/api/inbound \
  --inbound-key=your-secret-key \
  --allowed-domains=yourdomain.com \
  --strict-spf=true \
  --rate-limit=5 \
  --spam-keywords="viagra,lottery,urgent,winner" \
  --max-attach-size=5242880
```

**中等安全模式**（平衡）：
```bash
smtp2http \
  --listen=:25 \
  --webhook=https://your-domain.com/api/inbound \
  --inbound-key=your-secret-key \
  --allowed-domains=yourdomain.com \
  --rate-limit=10 \
  --spam-keywords="viagra,lottery"
```

**开发模式**（最小限制）：
```bash
smtp2http \
  --listen=:25 \
  --webhook=https://your-domain.com/api/inbound \
  --inbound-key=your-secret-key \
  --rate-limit=100
```

## 贡献
原始仓库来自 @alash3al
感谢 @aranajuan


