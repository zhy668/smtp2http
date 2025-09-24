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

Cloud-Mail Integration
=====================
This fork adds support for cloud-mail inbound API authentication:

```bash
# Basic usage with cloud-mail
smtp2http --listen=:25 --webhook=https://your-domain.com/api/inbound --inbound-key=your-secret-api-key

# With domain filtering
smtp2http --listen=:25 --webhook=https://your-domain.com/api/inbound --inbound-key=your-secret-api-key --domain=example.com
```

**New Parameters:**
- `--inbound-key`: API key for cloud-mail authentication (adds X-Inbound-Key header)

**Features:**
- Automatic X-Inbound-Key header injection for cloud-mail compatibility
- Enhanced logging for webhook requests
- Better error reporting with response body details

Contribution
============
Original repo from @alash3al
Thanks to @aranajuan


