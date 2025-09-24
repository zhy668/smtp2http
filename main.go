package main

import (
	"encoding/base64"
	"errors"
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
	cfg := smtpsrv.ServerConfig{
		ReadTimeout:     time.Duration(*flagReadTimeout) * time.Second,
		WriteTimeout:    time.Duration(*flagWriteTimeout) * time.Second,
		ListenAddr:      *flagListenAddr,
		MaxMessageBytes: int(*flagMaxMessageSize),
		BannerDomain:    *flagServerName,
		Handler: smtpsrv.HandlerFunc(func(c *smtpsrv.Context) error {
			msg, err := c.Parse()
			if err != nil {
				return errors.New("Cannot read your message: " + err.Error())
			}

			// 获取客户端信息
			clientIP := GetClientIP(c.RemoteAddr().String())
			senderEmail := c.From().Address
			recipientEmail := c.To().Address

			spfResult, _, _ := c.SPF()

			// 执行安全检查（在解析附件之前进行基础检查）
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
				log.Printf("Email rejected: %s (Score: %d, From: %s, To: %s, IP: %s)",
					reason, score, senderEmail, recipientEmail, clientIP)
				return errors.New("Email rejected: " + reason)
			}

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

			jsonData.Addresses.From = transformStdAddressToEmailAddress([]*mail.Address{c.From()})[0]
			jsonData.Addresses.To = transformStdAddressToEmailAddress([]*mail.Address{c.To()})[0]

			toSplited := strings.Split(jsonData.Addresses.To.Address, "@")
			if len(*flagDomain) > 0 && (len(toSplited) < 2 || toSplited[1] != *flagDomain) {
				log.Println("domain not allowed")
				log.Println(*flagDomain)
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

			for _, a := range msg.Attachments {
				data, _ := ioutil.ReadAll(a.Data)
				attachment := &EmailAttachment{
					Filename:    a.Filename,
					ContentType: a.ContentType,
					Data:        base64.StdEncoding.EncodeToString(data),
					Content:     data, // 用于安全检查
				}
				jsonData.Attachments = append(jsonData.Attachments, attachment)
			}

			// 执行附件安全检查
			if attachCheck := CheckAttachments(jsonData.Attachments); !attachCheck.Allowed {
				log.Printf("Email rejected due to attachment: %s (From: %s, To: %s)",
					attachCheck.Reason, senderEmail, recipientEmail)
				return errors.New("Email rejected: " + attachCheck.Reason)
			}

			for _, a := range msg.EmbeddedFiles {
				data, _ := ioutil.ReadAll(a.Data)
				jsonData.EmbeddedFiles = append(jsonData.EmbeddedFiles, &EmailEmbeddedFile{
					CID:         a.CID,
					ContentType: a.ContentType,
					Data:        base64.StdEncoding.EncodeToString(data),
				})
			}

			// Create HTTP request with required headers
			req := resty.New().R().SetHeader("Content-Type", "application/json").SetBody(jsonData)

			// Add API key header if provided (for cloud-mail inbound authentication)
			if *flagInboundKey != "" {
				req.SetHeader("X-Inbound-Key", *flagInboundKey)
				log.Printf("Sending email to webhook with API key: %s...", (*flagInboundKey)[:min(8, len(*flagInboundKey))])
			}

			// 记录安全检查通过的邮件
			log.Printf("Email accepted: From=%s, To=%s, Subject=%s, IP=%s, SPF=%s, Score=%d",
				senderEmail, recipientEmail, msg.Subject, clientIP, spfResult.String(), score)

			resp, err := req.Post(*flagWebhook)
			if err != nil {
				log.Println("Webhook request failed:", err)
				return errors.New("E1: Cannot accept your message due to internal error, please report that to our engineers")
			} else if resp.StatusCode() != 200 {
				log.Printf("Webhook returned non-200 status: %s, body: %s", resp.Status(), string(resp.Body()))
				return errors.New("E2: Cannot accept your message due to internal error, please report that to our engineers")
			}

			log.Printf("Email successfully forwarded to webhook: %s", *flagWebhook)

			return nil
		}),
	}

	fmt.Println(smtpsrv.ListenAndServe(&cfg))
}
