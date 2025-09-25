package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// RateLimiter 速率限制器
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter 创建新的速率限制器
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// 清理过期的请求记录
	if requests, exists := rl.requests[key]; exists {
		validRequests := make([]time.Time, 0, len(requests))
		for _, req := range requests {
			if req.After(cutoff) {
				validRequests = append(validRequests, req)
			}
		}
		rl.requests[key] = validRequests
	}

	// 检查是否超过限制
	if len(rl.requests[key]) >= rl.limit {
		return false
	}

	// 添加当前请求
	rl.requests[key] = append(rl.requests[key], now)
	return true
}

// 全局速率限制器
var rateLimiter *RateLimiter

func init() {
	rateLimiter = NewRateLimiter(*flagMaxEmailsPerMin, time.Minute)
}

// SecurityCheck 安全检查结果
type SecurityCheck struct {
	Allowed bool
	Reason  string
	Score   int
}

// ValidateRecipientDomain 验证收件人域名
func ValidateRecipientDomain(recipientEmail string) SecurityCheck {
	if *flagAllowedDomains == "" {
		return SecurityCheck{Allowed: true, Reason: "No domain restrictions"}
	}

	allowedDomains := strings.Split(*flagAllowedDomains, ",")
	parts := strings.Split(recipientEmail, "@")
	if len(parts) != 2 {
		return SecurityCheck{Allowed: false, Reason: "Invalid email format"}
	}

	recipientDomain := strings.ToLower(strings.TrimSpace(parts[1]))

	for _, domain := range allowedDomains {
		if strings.ToLower(strings.TrimSpace(domain)) == recipientDomain {
			return SecurityCheck{Allowed: true, Reason: "Domain allowed"}
		}
	}

	return SecurityCheck{Allowed: false, Reason: fmt.Sprintf("Domain %s not in allowed list", recipientDomain)}
}

// ValidateSenderDomain 验证发送者域名
func ValidateSenderDomain(senderEmail string) SecurityCheck {
	if *flagBlacklistDomains == "" {
		return SecurityCheck{Allowed: true, Reason: "No sender domain restrictions"}
	}

	blacklistDomains := strings.Split(*flagBlacklistDomains, ",")
	parts := strings.Split(senderEmail, "@")
	if len(parts) != 2 {
		return SecurityCheck{Allowed: true, Reason: "Invalid sender email format, allowing"}
	}

	senderDomain := strings.ToLower(strings.TrimSpace(parts[1]))

	for _, domain := range blacklistDomains {
		if strings.ToLower(strings.TrimSpace(domain)) == senderDomain {
			return SecurityCheck{Allowed: false, Reason: fmt.Sprintf("Sender domain %s is blacklisted", senderDomain)}
		}
	}

	return SecurityCheck{Allowed: true, Reason: "Sender domain not blacklisted"}
}

// CheckRateLimit 检查速率限制
func CheckRateLimit(clientIP string) SecurityCheck {
	if rateLimiter.Allow(clientIP) {
		return SecurityCheck{Allowed: true, Reason: "Rate limit OK"}
	}
	return SecurityCheck{Allowed: false, Reason: fmt.Sprintf("Rate limit exceeded for IP %s", clientIP)}
}

// CheckSpamKeywords 检查垃圾邮件关键词
func CheckSpamKeywords(subject, body string) SecurityCheck {
	if *flagSpamKeywords == "" {
		return SecurityCheck{Allowed: true, Reason: "No spam keyword filtering"}
	}

	keywords := strings.Split(*flagSpamKeywords, ",")
	content := strings.ToLower(subject + " " + body)

	for _, keyword := range keywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword != "" && strings.Contains(content, keyword) {
			return SecurityCheck{
				Allowed: false,
				Reason:  fmt.Sprintf("Contains spam keyword: %s", keyword),
				Score:   50,
			}
		}
	}

	return SecurityCheck{Allowed: true, Reason: "No spam keywords detected"}
}

// CheckAttachments 检查附件安全性
func CheckAttachments(attachments []*EmailAttachment) SecurityCheck {
	if len(attachments) == 0 {
		return SecurityCheck{Allowed: true, Reason: "No attachments"}
	}

	forbiddenTypes := strings.Split(*flagForbiddenTypes, ",")

	for _, attachment := range attachments {
		// 检查文件扩展名
		if attachment.Filename != "" {
			parts := strings.Split(attachment.Filename, ".")
			if len(parts) > 1 {
				ext := strings.ToLower(strings.TrimSpace(parts[len(parts)-1]))
				for _, forbiddenExt := range forbiddenTypes {
					if strings.ToLower(strings.TrimSpace(forbiddenExt)) == ext {
						return SecurityCheck{
							Allowed: false,
							Reason:  fmt.Sprintf("Forbidden file type: %s (%s)", ext, attachment.Filename),
							Score:   80,
						}
					}
				}
			}
		}

		// 检查文件大小
		if len(attachment.Content) > int(*flagMaxAttachSize) {
			return SecurityCheck{
				Allowed: false,
				Reason:  fmt.Sprintf("Attachment too large: %s (%d bytes)", attachment.Filename, len(attachment.Content)),
				Score:   30,
			}
		}
	}

	return SecurityCheck{Allowed: true, Reason: "Attachments OK"}
}

// ValidateSPF 验证 SPF 记录
func ValidateSPF(spfResult string) SecurityCheck {
	if !*flagStrictSPF {
		return SecurityCheck{Allowed: true, Reason: "SPF check disabled"}
	}

	switch strings.ToLower(spfResult) {
	case "pass":
		return SecurityCheck{Allowed: true, Reason: "SPF verification passed"}
	case "fail":
		return SecurityCheck{Allowed: false, Reason: "SPF verification failed", Score: 60}
	case "softfail":
		return SecurityCheck{Allowed: true, Reason: "SPF soft fail, allowing", Score: 20}
	case "neutral", "none":
		return SecurityCheck{Allowed: true, Reason: "No SPF record or neutral", Score: 10}
	default:
		return SecurityCheck{Allowed: true, Reason: "SPF result unknown, allowing", Score: 5}
	}
}

// PerformSecurityChecks 执行所有安全检查
func PerformSecurityChecks(clientIP, senderEmail, recipientEmail, subject, body, spfResult string, attachments []*EmailAttachment) (bool, string, int) {
	var totalScore int
	var reasons []string

	// 1. 速率限制检查
	if check := CheckRateLimit(clientIP); !check.Allowed {
		return false, check.Reason, 100
	}

	// 2. 收件人域名验证
	if check := ValidateRecipientDomain(recipientEmail); !check.Allowed {
		return false, check.Reason, 100
	}

	// 3. 发送者域名验证
	if check := ValidateSenderDomain(senderEmail); !check.Allowed {
		return false, check.Reason, 100
	}

	// 4. SPF 验证
	if check := ValidateSPF(spfResult); !check.Allowed {
		return false, check.Reason, check.Score
	} else if check.Score > 0 {
		totalScore += check.Score
		reasons = append(reasons, check.Reason)
	}

	// 5. 垃圾邮件关键词检查
	if check := CheckSpamKeywords(subject, body); !check.Allowed {
		return false, check.Reason, check.Score
	} else if check.Score > 0 {
		totalScore += check.Score
		reasons = append(reasons, check.Reason)
	}

	// 6. 附件安全检查
	if check := CheckAttachments(attachments); !check.Allowed {
		return false, check.Reason, check.Score
	} else if check.Score > 0 {
		totalScore += check.Score
		reasons = append(reasons, check.Reason)
	}

	// 综合评分判断
	if totalScore >= 70 {
		reasonStr := strings.Join(reasons, "; ")
		return false, fmt.Sprintf("High security risk score: %d (%s)", totalScore, reasonStr), totalScore
	}

	if len(reasons) > 0 {
		log.Printf("Email flagged with security score %d: %s", totalScore, strings.Join(reasons, "; "))
	}

	return true, "Security checks passed", totalScore
}

// GetClientIP 获取客户端 IP 地址
func GetClientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

// DNSTXTRecord DNS TXT 记录结构
type DNSTXTRecord struct {
	Allow  bool   `json:"allow"`
	Hook   string `json:"hook"`
	Secret string `json:"secret"`
}

// DNSCache DNS 查询缓存
type DNSCache struct {
	mu      sync.RWMutex
	records map[string]*CacheEntry
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Record    *DNSTXTRecord
	ExpiresAt time.Time
	Error     error
}

// NewDNSCache 创建新的 DNS 缓存
func NewDNSCache() *DNSCache {
	return &DNSCache{
		records: make(map[string]*CacheEntry),
	}
}

// Get 从缓存获取记录
func (dc *DNSCache) Get(domain string) (*DNSTXTRecord, error, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	entry, exists := dc.records[domain]
	if !exists {
		return nil, nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		return nil, nil, false
	}

	return entry.Record, entry.Error, true
}

// Set 设置缓存记录
func (dc *DNSCache) Set(domain string, record *DNSTXTRecord, err error, ttl time.Duration) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.records[domain] = &CacheEntry{
		Record:    record,
		ExpiresAt: time.Now().Add(ttl),
		Error:     err,
	}
}

// 全局 DNS 缓存实例
var dnsCache = NewDNSCache()

// parseTXTRecord 解析 TXT 记录内容
func parseTXTRecord(txtRecord string) (*DNSTXTRecord, error) {
	record := &DNSTXTRecord{}

	// 移除引号
	txtRecord = strings.Trim(txtRecord, "\"")

	// 按分号分割字段
	fields := strings.Split(txtRecord, ";")

	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		// 按等号分割键值对
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "allow":
			record.Allow = value == "1" || strings.ToLower(value) == "true"
		case "hook":
			// 验证 URL 格式
			if _, err := url.Parse(value); err != nil {
				return nil, fmt.Errorf("invalid hook URL: %v", err)
			}
			record.Hook = value
		case "secret":
			record.Secret = value
		}
	}

	return record, nil
}

// queryDNSTXTRecord 查询 DNS TXT 记录
func queryDNSTXTRecord(domain string) (*DNSTXTRecord, error) {
	// 构造查询域名
	queryDomain := "_smtp2http." + domain

	log.Printf("DNS TXT: Querying %s", queryDomain)

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 查询 TXT 记录
	txtRecords, err := net.DefaultResolver.LookupTXT(ctx, queryDomain)
	if err != nil {
		log.Printf("DNS TXT: Query failed for %s: %v", queryDomain, err)
		return nil, fmt.Errorf("DNS query failed: %v", err)
	}

	if len(txtRecords) == 0 {
		log.Printf("DNS TXT: No records found for %s", queryDomain)
		return nil, fmt.Errorf("no TXT records found")
	}

	// 查找包含 smtp2http 配置的记录
	for _, record := range txtRecords {
		log.Printf("DNS TXT: Found record for %s: %s", queryDomain, record)

		// 检查是否包含 allow 字段
		if strings.Contains(record, "allow=") {
			parsed, err := parseTXTRecord(record)
			if err != nil {
				log.Printf("DNS TXT: Failed to parse record: %v", err)
				continue
			}

			log.Printf("DNS TXT: Parsed record for %s: allow=%t, secret=%s, hook=%s",
				domain, parsed.Allow,
				func() string {
					if len(parsed.Secret) > 8 {
						return parsed.Secret[:8] + "..."
					}
					return parsed.Secret
				}(), parsed.Hook)

			return parsed, nil
		}
	}

	return nil, fmt.Errorf("no valid smtp2http TXT record found")
}

// ValidateRecipientDomainDNS 基于 DNS TXT 记录验证收件人域名
func ValidateRecipientDomainDNS(recipientEmail, requiredSecret string) SecurityCheck {
	// 如果未设置密码，跳过 DNS 验证
	if requiredSecret == "" {
		return SecurityCheck{Allowed: true, Reason: "DNS TXT validation disabled"}
	}

	// 提取域名
	parts := strings.Split(recipientEmail, "@")
	if len(parts) != 2 {
		log.Printf("DNS TXT: Invalid email format: %s", recipientEmail)
		return SecurityCheck{Allowed: false, Reason: "Invalid email format"}
	}

	domain := strings.ToLower(strings.TrimSpace(parts[1]))

	// 检查缓存
	if record, err, found := dnsCache.Get(domain); found {
		if err != nil {
			log.Printf("DNS TXT: Cached error for %s: %v", domain, err)
			return SecurityCheck{Allowed: false, Reason: fmt.Sprintf("DNS validation failed: %v", err)}
		}

		log.Printf("DNS TXT: Using cached record for %s", domain)
		return validateDNSRecord(record, requiredSecret, domain)
	}

	// 查询 DNS TXT 记录
	record, err := queryDNSTXTRecord(domain)

	// 缓存结果（包括错误）
	cacheTTL := 10 * time.Minute // 默认缓存 10 分钟
	if err != nil {
		cacheTTL = 2 * time.Minute // 错误结果缓存时间较短
	}
	dnsCache.Set(domain, record, err, cacheTTL)

	if err != nil {
		log.Printf("DNS TXT: Validation failed for %s: %v", domain, err)
		return SecurityCheck{Allowed: false, Reason: fmt.Sprintf("DNS validation failed: %v", err)}
	}

	return validateDNSRecord(record, requiredSecret, domain)
}

// validateDNSRecord 验证 DNS 记录
func validateDNSRecord(record *DNSTXTRecord, requiredSecret, domain string) SecurityCheck {
	// 检查是否允许
	if !record.Allow {
		log.Printf("DNS TXT: Domain %s is not allowed (allow=false)", domain)
		return SecurityCheck{Allowed: false, Reason: fmt.Sprintf("Domain %s not allowed by DNS TXT record", domain)}
	}

	// 验证密码
	if record.Secret != requiredSecret {
		log.Printf("DNS TXT: Secret mismatch for domain %s", domain)
		return SecurityCheck{Allowed: false, Reason: fmt.Sprintf("Secret mismatch for domain %s", domain)}
	}

	log.Printf("DNS TXT: Domain %s validation passed", domain)
	return SecurityCheck{Allowed: true, Reason: fmt.Sprintf("Domain %s validated via DNS TXT", domain)}
}
