// Package email содержит SMTP-клиент для отправки email-уведомлений.
package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/mintfary/aqi-platform/internal/config"
)

// Sender — отправитель email-уведомлений через SMTP.
type Sender struct {
	cfg config.EmailConfig
}

// NewSender создаёт SMTP-отправитель.
// Если SMTP не настроен (SMTPHost пустой), отправка будет no-op.
func NewSender(cfg config.EmailConfig) *Sender {
	return &Sender{cfg: cfg}
}

// IsConfigured возвращает true если SMTP настроен.
func (s *Sender) IsConfigured() bool {
	return s.cfg.SMTPHost != ""
}

// SendPasswordReset отправляет письмо со ссылкой для сброса пароля.
func (s *Sender) SendPasswordReset(toEmail, resetToken, baseURL string) error {
	if !s.IsConfigured() {
		return nil // SMTP не настроен — тихо игнорируем
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", strings.TrimRight(baseURL, "/"), resetToken)

	data := passwordResetData{
		ResetURL:  resetURL,
		ExpiresAt: time.Now().Add(1 * time.Hour).Format("15:04 02.01.2006"),
		AppName:   "AQI Кемерово",
	}

	body, err := renderPasswordResetEmail(data)
	if err != nil {
		return fmt.Errorf("smtp: render template: %w", err)
	}

	subject := "Сброс пароля — AQI Кемерово"
	return s.send(toEmail, subject, body)
}

// send отправляет письмо на указанный адрес.
func (s *Sender) send(toEmail, subject, htmlBody string) error {
	from := s.cfg.FromAddr
	if from == "" {
		from = s.cfg.SMTPUser
	}

	fromAddr := mail.Address{Name: "AQI Кемерово", Address: from}
	toAddr := mail.Address{Address: toEmail}

	// Формируем заголовки письма.
	headers := map[string]string{
		"From":         fromAddr.String(),
		"To":           toAddr.String(),
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=utf-8",
		"Date":         time.Now().Format(time.RFC1123Z),
	}

	var msg bytes.Buffer
	for k, v := range headers {
		fmt.Fprintf(&msg, "%s: %s\r\n", k, v)
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	port := s.cfg.SMTPPort
	if port == 0 {
		port = 587 // STARTTLS default
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, port)

	// Определяем режим TLS по порту.
	if port == 465 {
		// Implicit TLS (SMTPS)
		return s.sendTLS(addr, from, toEmail, msg.Bytes())
	}
	// STARTTLS (порт 587/25)
	return s.sendSTARTTLS(addr, from, toEmail, msg.Bytes())
}

func (s *Sender) sendSTARTTLS(addr, from, to string, msg []byte) error {
	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPass, s.cfg.SMTPHost)

	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

func (s *Sender) sendTLS(addr, from, to string, msg []byte) error {
	host, _, _ := net.SplitHostPort(addr)
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}
	defer conn.Close() //nolint:errcheck

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer c.Quit() //nolint:errcheck

	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPass, host)
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	defer w.Close() //nolint:errcheck

	_, err = w.Write(msg)
	return err
}

// ── HTML-шаблон письма ─────────────────────────────────────────────────────

type passwordResetData struct {
	ResetURL  string
	ExpiresAt string
	AppName   string
}

const passwordResetHTML = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Сброс пароля</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
         background: #f4f6f9; margin: 0; padding: 20px; }
  .card { max-width: 480px; margin: 0 auto; background: #fff;
          border-radius: 10px; padding: 40px; box-shadow: 0 2px 12px rgba(0,0,0,.1); }
  .logo { font-size: 24px; font-weight: 700; color: #5b8dee; margin-bottom: 24px; }
  h2   { font-size: 20px; color: #1a1d27; margin-bottom: 12px; }
  p    { color: #555; line-height: 1.6; margin-bottom: 16px; }
  .btn { display: inline-block; padding: 13px 28px; background: #5b8dee;
         color: #fff !important; text-decoration: none; border-radius: 7px;
         font-weight: 600; font-size: 15px; margin: 8px 0 20px; }
  .note { font-size: 13px; color: #999; border-top: 1px solid #eee; padding-top: 16px; }
  .url  { word-break: break-all; color: #5b8dee; font-size: 12px; }
</style>
</head>
<body>
<div class="card">
  <div class="logo">🌫️ {{.AppName}}</div>
  <h2>Сброс пароля</h2>
  <p>Мы получили запрос на сброс пароля для вашего аккаунта.</p>
  <p>Нажмите кнопку ниже для создания нового пароля:</p>
  <a href="{{.ResetURL}}" class="btn">Сбросить пароль</a>
  <p>Ссылка действительна до: <strong>{{.ExpiresAt}}</strong></p>
  <p class="note">
    Если вы не запрашивали сброс пароля — просто проигнорируйте это письмо.
    Ваш пароль останется прежним.<br><br>
    Если кнопка не работает, скопируйте эту ссылку в браузер:<br>
    <span class="url">{{.ResetURL}}</span>
  </p>
</div>
</body>
</html>`

func renderPasswordResetEmail(data passwordResetData) (string, error) {
	tmpl, err := template.New("reset").Parse(passwordResetHTML)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
