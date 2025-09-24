package main

import (
	"fmt"
	"log"
	"net"
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
