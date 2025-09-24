package main

import "flag"

var (
	flagServerName     = flag.String("name", "smtp2http", "the server name")
	flagListenAddr     = flag.String("listen", ":smtp", "the smtp address to listen on")
	flagWebhook        = flag.String("webhook", "http://localhost:8080/my/webhook", "the webhook to send the data to")
	flagMaxMessageSize = flag.Int64("msglimit", 1024*1024*2, "maximum incoming message size")
	flagReadTimeout    = flag.Int("timeout.read", 5, "the read timeout in seconds")
	flagWriteTimeout   = flag.Int("timeout.write", 5, "the write timeout in seconds")
	flagAuthUSER       = flag.String("user", "", "user for smtp client")
	flagAuthPASS       = flag.String("pass", "", "pass for smtp client")
	flagDomain         = flag.String("domain", "", "domain for recieving mails")
	flagInboundKey     = flag.String("inbound-key", "", "API key for cloud-mail inbound authentication (X-Inbound-Key header)")

	// Security configuration
	flagAllowedDomains   = flag.String("allowed-domains", "", "comma-separated list of allowed recipient domains (empty = allow all)")
	flagStrictSPF        = flag.Bool("strict-spf", false, "reject emails that fail SPF verification")
	flagMaxEmailsPerMin  = flag.Int("rate-limit", 60, "maximum emails per minute per sender IP")
	flagSpamKeywords     = flag.String("spam-keywords", "", "comma-separated list of spam keywords to block")
	flagForbiddenTypes   = flag.String("forbidden-types", "exe,bat,cmd,com,pif,scr,vbs,js,jar,msi", "comma-separated list of forbidden attachment file extensions")
	flagMaxAttachSize    = flag.Int64("max-attach-size", 10*1024*1024, "maximum attachment size in bytes (default 10MB)")
	flagBlacklistDomains = flag.String("blacklist-domains", "", "comma-separated list of blacklisted sender domains")
)

func init() {
	flag.Parse()
}
