# SMTP2HTTP 预编译版本

这个目录包含了 smtp2http 的预编译二进制文件，支持不同的操作系统和架构。

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

## 安全功能参数

所有版本都支持以下安全功能：

- `--rate-limit=5` - 每个IP每分钟最大邮件数
- `--max-attach-size=10485760` - 最大附件大小（字节）
- `--forbidden-types=exe,bat,cmd` - 禁止的文件扩展名
- `--blacklist-domains=spam.com,bad.org` - 发送者域名黑名单
- `--allowed-domains=good.com,safe.org` - 收件人域名白名单
- `--strict-spf=true` - 启用严格SPF验证

## 完整示例

```bash
./smtp2http-linux-arm64 \
  --listen=:25 \
  --webhook=https://your-domain.com/api/inbound \
  --inbound-key=your-secret-key \
  --rate-limit=5 \
  --max-attach-size=10485760 \
  --forbidden-types=exe,bat,cmd,com,pif,scr,vbs,js,jar,msi \
  --blacklist-domains=spam-domain.com,malicious-site.org
```

## 版本信息

这些二进制文件基于最新的源代码编译，包含所有安全功能和中文文档支持。

如需自定义编译，请参考项目根目录的源代码。
