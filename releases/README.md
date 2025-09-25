# SMTP2HTTP 预编译版本 (v2.0)

这个目录包含了 smtp2http 的预编译二进制文件，支持不同的操作系统和架构。

## 🆕 新功能 (v2.0)

### 1. 完善的日志处理系统
- 详细的邮件接收、处理、转发各阶段日志
- 清晰的错误信息和调试信息
- 分阶段日志标记（SMTP、WEBHOOK、DNS TXT等）

### 2. DNS TXT 记录域名验证
- 基于 DNS TXT 记录的高级域名验证
- 支持动态域名授权管理
- 内置 DNS 查询缓存机制（5-15分钟缓存）

## 可用版本

### Windows
- `smtp2http-windows-amd64.exe` - Windows 64位 (AMD64/x86_64)

### Linux
- `smtp2http-linux-amd64` - Linux 64位 (AMD64/x86_64)
- `smtp2http-linux-arm64` - Linux ARM64 (适用于 ARM 服务器，如树莓派4、Apple M1等)

## 使用方法

### Windows
```cmd
smtp2http-windows-amd64.exe --listen=:25 --webhook=https://your-domain.com/api/inbound --inbound-key=your-secret-key
```

### Linux
```bash
# 添加执行权限
chmod +x smtp2http-linux-amd64
# 或
chmod +x smtp2http-linux-arm64

# 运行
./smtp2http-linux-amd64 --listen=:25 --webhook=https://your-domain.com/api/inbound --inbound-key=your-secret-key
```

## 🔐 DNS TXT 记录域名验证

### 启用方式
```bash
./smtp2http-linux-amd64 --rcpt-domain-secret=your-secret-password
```

### DNS TXT 记录格式
为需要接收邮件的域名添加 TXT 记录：

**记录名**: `_smtp2http.yourdomain.com`
**记录值**: `"allow=1;hook=https://your-webhook-url;secret=your-secret-password"`

### 示例配置
```
# DNS 记录
_smtp2http.example.com TXT "allow=1;hook=https://mail.example.com/api/inbound;secret=abc123"

# smtp2http 启动命令
./smtp2http-linux-amd64 \
  --listen=:25 \
  --webhook=https://mail.example.com/api/inbound \
  --rcpt-domain-secret=abc123
```

### 验证流程
1. 收到邮件时，提取收件人域名
2. 查询 `_smtp2http.<域名>` 的 TXT 记录
3. 验证记录中的 `secret` 是否匹配
4. 只有验证通过的域名才允许接收邮件

## 安全功能参数

所有版本都支持以下安全功能：

- `--rate-limit=5` - 每个IP每分钟最大邮件数
- `--max-attach-size=10485760` - 最大附件大小（字节）
- `--forbidden-types=exe,bat,cmd` - 禁止的文件扩展名
- `--blacklist-domains=spam.com,bad.org` - 发送者域名黑名单
- `--allowed-domains=good.com,safe.org` - 收件人域名白名单
- `--strict-spf=true` - 启用严格SPF验证
- `--rcpt-domain-secret=password` - 🆕 启用DNS TXT记录域名验证

## 完整示例

### 基础配置
```bash
./smtp2http-linux-arm64 \
  --listen=:25 \
  --webhook=https://your-domain.com/api/inbound \
  --inbound-key=your-secret-key \
  --rate-limit=5 \
  --max-attach-size=10485760 \
  --forbidden-types=exe,bat,cmd,com,pif,scr,vbs,js,jar,msi \
  --blacklist-domains=spam-domain.com,malicious-site.org \
  --allowed-domains=your-domain.com,trusted-domain.org \
  --strict-spf=true
```

### 启用 DNS TXT 验证
```bash
./smtp2http-linux-amd64 \
  --listen=:25 \
  --webhook=https://your-domain.com/api/inbound \
  --inbound-key=your-secret-key \
  --rcpt-domain-secret=your-dns-secret \
  --rate-limit=5
```

## 📋 日志示例

启用详细日志后，您将看到类似以下的输出：
```
2025/09/25 11:45:00 Starting smtp2http server with enhanced logging and DNS TXT validation
2025/09/25 11:45:00 DNS TXT domain validation enabled with secret: abc123...
2025/09/25 11:45:01 SMTP: New connection from 192.168.1.100, MAIL FROM: sender@example.com, RCPT TO: user@yourdomain.com
2025/09/25 11:45:01 SMTP: Performing DNS TXT validation for recipient domain
2025/09/25 11:45:01 DNS TXT: Querying _smtp2http.yourdomain.com
2025/09/25 11:45:01 DNS TXT: Domain yourdomain.com validation passed
2025/09/25 11:45:01 SMTP: Parsing email message
2025/09/25 11:45:01 SMTP: Security checks passed (Score: 0)
2025/09/25 11:45:01 WEBHOOK: Sending POST request to https://your-domain.com/api/inbound
2025/09/25 11:45:02 WEBHOOK: Email successfully processed and forwarded
```

## 注意事项

1. **端口权限**: 在Linux系统上绑定25端口需要root权限
2. **防火墙**: 确保SMTP端口（通常是25）在防火墙中开放
3. **DNS配置**: 确保MX记录指向运行smtp2http的服务器
4. **SSL/TLS**: 如果需要加密传输，请在前端使用nginx等反向代理
5. **🆕 DNS TXT记录**: 启用DNS验证时，确保为每个接收域名配置正确的TXT记录，格式为 `_smtp2http.yourdomain.com TXT "allow=1;hook=https://inbox.example.com/api/inbound;secret=your-secret-password"`

## 更新日志

### v2.0 (2025-09-25)
- 🆕 **DNS TXT记录域名验证**: 基于DNS TXT记录的高级域名验证功能
- 🆕 **完善的日志系统**: 详细的分阶段日志记录，便于调试和监控
- 🆕 **增强的错误处理**: 更清晰的错误信息和应用层响应检查
- 🆕 **DNS查询缓存**: 内置DNS查询结果缓存，提高性能
- 🔧 **改进的webhook响应处理**: 检查应用层错误码，不仅仅是HTTP状态码

### v1.x
- 增强了邮件安全检查功能
- 添加了SPF验证支持
- 改进了附件过滤机制
- 优化了错误处理和日志记录

更多详细配置和高级功能请参考主项目的 README.md 文件。
