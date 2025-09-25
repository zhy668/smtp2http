package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/mail"
	"strings"
	"time"

	"github.com/alash3al/go-smtpsrv"
	"github.com/go-resty/resty/v2"
)

// min returns the smaller of two integers (for Go versions < 1.21)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	flag.Parse()

	log.Printf("Starting smtp2http server with enhanced logging and DNS TXT validation")
	if *flagRcptDomainSecret != "" {
		log.Printf("DNS TXT domain validation enabled with secret: %s...",
			func() string {
				if len(*flagRcptDomainSecret) > 8 {
					return (*flagRcptDomainSecret)[:8]
				}
				return *flagRcptDomainSecret
			}())
	} else {
		log.Printf("DNS TXT domain validation disabled")
	}

	cfg := smtpsrv.ServerConfig{
		ReadTimeout:     time.Duration(*flagReadTimeout) * time.Second,
		WriteTimeout:    time.Duration(*flagWriteTimeout) * time.Second,
		ListenAddr:      *flagListenAddr,
		MaxMessageBytes: int(*flagMaxMessageSize),
		BannerDomain:    *flagServerName,
		Handler: smtpsrv.HandlerFunc(func(c *smtpsrv.Context) error {
			// 获取客户端信息
			clientIP := GetClientIP(c.RemoteAddr().String())
			recipientEmail := c.To().Address
			senderEmail := c.From().Address

			log.Printf("SMTP: New connection from %s, MAIL FROM: %s, RCPT TO: %s", clientIP, senderEmail, recipientEmail)

			// DNS TXT 记录域名验证（在解析邮件内容之前进行）
			if *flagRcptDomainSecret != "" {
				log.Printf("SMTP: Performing DNS TXT validation for recipient domain")
				dnsCheck := ValidateRecipientDomainDNS(recipientEmail, *flagRcptDomainSecret)
				if !dnsCheck.Allowed {
					log.Printf("SMTP: DNS TXT validation failed: %s (From: %s, To: %s, IP: %s)",
						dnsCheck.Reason, senderEmail, recipientEmail, clientIP)
					return errors.New("Domain not authorized: " + dnsCheck.Reason)
				}
				log.Printf("SMTP: DNS TXT validation passed: %s", dnsCheck.Reason)
			}

			log.Printf("SMTP: Parsing email message")
			msg, err := c.Parse()
			if err != nil {
				log.Printf("SMTP: Failed to parse message: %v (From: %s, To: %s, IP: %s)",
					err, senderEmail, recipientEmail, clientIP)
				return errors.New("Cannot read your message: " + err.Error())
			}

			// 优先使用邮件头中的 From 字段作为发件人地址
			if len(msg.From) > 0 {
				senderEmail = msg.From[0].Address
				log.Printf("SMTP: Updated sender from email header: %s", senderEmail)
			}

			spfResult, _, _ := c.SPF()
			log.Printf("SMTP: SPF check result: %s", spfResult.String())

			// 执行安全检查（在解析附件之前进行基础检查）
			log.Printf("SMTP: Performing security checks")
			allowed, reason, score := PerformSecurityChecks(
				clientIP,
				senderEmail,
				recipientEmail,
				msg.Subject,
				string(msg.TextBody),
				spfResult.String(),
				nil, // 附件稍后检查
			)

			if !allowed {
				log.Printf("SMTP: Security check failed - %s (Score: %d, From: %s, To: %s, IP: %s)",
					reason, score, senderEmail, recipientEmail, clientIP)
				return errors.New("Email rejected: " + reason)
			}
			log.Printf("SMTP: Security checks passed (Score: %d)", score)

			log.Printf("SMTP: Building email message structure")
			jsonData := EmailMessage{
				ID:            msg.MessageID,
				Date:          msg.Date.String(),
				References:    msg.References,
				SPFResult:     spfResult.String(),
				ResentDate:    msg.ResentDate.String(),
				ResentID:      msg.ResentMessageID,
				Subject:       msg.Subject,
				Attachments:   []*EmailAttachment{},
				EmbeddedFiles: []*EmailEmbeddedFile{},
			}

			jsonData.Body.HTML = string(msg.HTMLBody)
			jsonData.Body.Text = string(msg.TextBody)

			log.Printf("SMTP: Email content - Subject: %s, HTML size: %d bytes, Text size: %d bytes",
				msg.Subject, len(jsonData.Body.HTML), len(jsonData.Body.Text))

			// 优先使用邮件头中的 From 字段，如果不存在则使用 SMTP 信封地址
			log.Printf("SMTP: Processing email addresses")
			if len(msg.From) > 0 {
				jsonData.Addresses.From = transformStdAddressToEmailAddress(msg.From)[0]
				log.Printf("SMTP: Using header From address: %s", jsonData.Addresses.From.Address)
			} else {
				jsonData.Addresses.From = transformStdAddressToEmailAddress([]*mail.Address{c.From()})[0]
				log.Printf("SMTP: Using envelope From address: %s", jsonData.Addresses.From.Address)
			}
			jsonData.Addresses.To = transformStdAddressToEmailAddress([]*mail.Address{c.To()})[0]
			log.Printf("SMTP: To address: %s", jsonData.Addresses.To.Address)

			// 检查传统域名限制（向后兼容）
			toSplited := strings.Split(jsonData.Addresses.To.Address, "@")
			if len(*flagDomain) > 0 && (len(toSplited) < 2 || toSplited[1] != *flagDomain) {
				log.Printf("SMTP: Domain restriction failed - expected: %s, got: %s",
					*flagDomain, func() string {
						if len(toSplited) >= 2 {
							return toSplited[1]
						}
						return "invalid"
					}())
				return errors.New("Unauthorized TO domain")
			}

			jsonData.Addresses.Cc = transformStdAddressToEmailAddress(msg.Cc)
			jsonData.Addresses.Bcc = transformStdAddressToEmailAddress(msg.Bcc)
			jsonData.Addresses.ReplyTo = transformStdAddressToEmailAddress(msg.ReplyTo)
			jsonData.Addresses.InReplyTo = msg.InReplyTo

			if resentFrom := transformStdAddressToEmailAddress(msg.ResentFrom); len(resentFrom) > 0 {
				jsonData.Addresses.ResentFrom = resentFrom[0]
			}

			jsonData.Addresses.ResentTo = transformStdAddressToEmailAddress(msg.ResentTo)
			jsonData.Addresses.ResentCc = transformStdAddressToEmailAddress(msg.ResentCc)
			jsonData.Addresses.ResentBcc = transformStdAddressToEmailAddress(msg.ResentBcc)

			// 处理附件
			log.Printf("SMTP: Processing %d attachments", len(msg.Attachments))
			for i, a := range msg.Attachments {
				data, err := ioutil.ReadAll(a.Data)
				if err != nil {
					log.Printf("SMTP: Failed to read attachment %d (%s): %v", i+1, a.Filename, err)
					return errors.New("Failed to process attachment: " + a.Filename)
				}

				attachment := &EmailAttachment{
					Filename:    a.Filename,
					ContentType: a.ContentType,
					Data:        base64.StdEncoding.EncodeToString(data),
					Content:     data, // 用于安全检查
				}
				jsonData.Attachments = append(jsonData.Attachments, attachment)
				log.Printf("SMTP: Processed attachment %d: %s (%s, %d bytes)",
					i+1, a.Filename, a.ContentType, len(data))
			}

			// 执行附件安全检查
			if len(jsonData.Attachments) > 0 {
				log.Printf("SMTP: Performing attachment security checks")
				if attachCheck := CheckAttachments(jsonData.Attachments); !attachCheck.Allowed {
					log.Printf("SMTP: Attachment security check failed: %s (From: %s, To: %s)",
						attachCheck.Reason, senderEmail, recipientEmail)
					return errors.New("Email rejected: " + attachCheck.Reason)
				}
				log.Printf("SMTP: Attachment security checks passed")
			}

			// 处理嵌入文件
			log.Printf("SMTP: Processing %d embedded files", len(msg.EmbeddedFiles))
			for i, a := range msg.EmbeddedFiles {
				data, err := ioutil.ReadAll(a.Data)
				if err != nil {
					log.Printf("SMTP: Failed to read embedded file %d (CID: %s): %v", i+1, a.CID, err)
					return errors.New("Failed to process embedded file: " + a.CID)
				}

				jsonData.EmbeddedFiles = append(jsonData.EmbeddedFiles, &EmailEmbeddedFile{
					CID:         a.CID,
					ContentType: a.ContentType,
					Data:        base64.StdEncoding.EncodeToString(data),
				})
				log.Printf("SMTP: Processed embedded file %d: CID=%s (%s, %d bytes)",
					i+1, a.CID, a.ContentType, len(data))
			}

			// 准备 webhook 请求
			log.Printf("SMTP: Preparing webhook request to %s", *flagWebhook)
			req := resty.New().R().SetHeader("Content-Type", "application/json").SetBody(jsonData)

			// Add API key header if provided (for cloud-mail inbound authentication)
			if *flagInboundKey != "" {
				req.SetHeader("X-Inbound-Key", *flagInboundKey)
				log.Printf("SMTP: Adding API key header: %s...", (*flagInboundKey)[:min(8, len(*flagInboundKey))])
			}

			// 记录邮件接受信息
			log.Printf("SMTP: Email accepted for processing - From=%s, To=%s, Subject=%s, IP=%s, SPF=%s, Score=%d",
				senderEmail, recipientEmail, msg.Subject, clientIP, spfResult.String(), score)

			// 发送 webhook 请求
			log.Printf("WEBHOOK: Sending POST request to %s", *flagWebhook)
			resp, err := req.Post(*flagWebhook)
			if err != nil {
				log.Printf("WEBHOOK: Request failed - %v (From: %s, To: %s)", err, senderEmail, recipientEmail)
				return errors.New("E1: Cannot accept your message due to internal error, please report that to our engineers")
			}

			log.Printf("WEBHOOK: Received response - Status: %d, Size: %d bytes",
				resp.StatusCode(), len(resp.Body()))

			if resp.StatusCode() != 200 {
				log.Printf("WEBHOOK: Non-200 status received - %s, Body: %s (From: %s, To: %s)",
					resp.Status(), string(resp.Body()), senderEmail, recipientEmail)
				return errors.New("E2: Cannot accept your message due to internal error, please report that to our engineers")
			}

			// 检查响应内容是否包含错误信息
			responseBody := string(resp.Body())
			if strings.Contains(responseBody, `"code":403`) || strings.Contains(responseBody, `"code":400`) {
				log.Printf("WEBHOOK: Application-level rejection - %s (From: %s, To: %s)",
					responseBody, senderEmail, recipientEmail)
				return errors.New("Email rejected by destination server")
			}

			log.Printf("WEBHOOK: Email successfully processed and forwarded to %s (From: %s, To: %s)",
				*flagWebhook, senderEmail, recipientEmail)

			return nil
		}),
	}

	fmt.Println(smtpsrv.ListenAndServe(&cfg))
}
