SMTP2HTTP (email-to-web)
========================
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


