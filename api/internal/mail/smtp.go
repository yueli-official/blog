package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
)

// SMTPSender delivers over implicit TLS on :465 (tls.Dial -> smtp.NewClient ->
// PlainAuth). net/smtp.SendMail's STARTTLS path does not fit port 465.
type SMTPSender struct {
	host, port, username, password, from, fromName string
	tlsConfig                                      *tls.Config
}

func NewSMTP(host, port, username, password, from, fromName string) *SMTPSender {
	return &SMTPSender{host: host, port: port, username: username, password: password, from: from, fromName: fromName, tlsConfig: &tls.Config{ServerName: host}}
}

// Send validates the recipient, then delivers asynchronously and returns
// immediately. Returning before the SMTP round-trip keeps response latency
// identical whether or not a mail is sent. Delivery is best-effort: errors are
// logged, not surfaced (the caller must not learn whether an address
// exists/delivered).
func (m *SMTPSender) Send(_ context.Context, to, subject, htmlBody string) error {
	if strings.ContainsAny(to, "\r\n") { // header-injection guard (defense-in-depth)
		g.Log().Errorf(context.Background(), "[blog:mail] rejecting recipient with CR/LF: %q", to)
		return nil
	}
	msg := m.buildMessage(to, subject, htmlBody)
	go func() {
		if err := m.send(to, msg); err != nil {
			g.Log().Errorf(context.Background(), "[blog:mail] send to=%s failed: %v", to, err)
		}
	}()
	return nil
}

// CheckHealth verifies TCP/TLS and SMTP authentication without selecting a
// sender/recipient or opening DATA, so it cannot deliver a real message.
func (m *SMTPSender) CheckHealth(ctx context.Context) error {
	client, err := m.authenticatedClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Quit()
}

// buildMessage is pure (unit-tested); send dials and delivers.
func (m *SMTPSender) buildMessage(to, subject, htmlBody string) []byte {
	from := m.from
	if m.fromName != "" {
		from = fmt.Sprintf("%s <%s>", m.fromName, m.from)
	}
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(htmlBody)
	return []byte(b.String())
}

func (m *SMTPSender) send(to string, msg []byte) error {
	c, err := m.authenticatedClient(context.Background())
	if err != nil {
		return err
	}
	defer c.Close()
	if err := c.Mail(m.from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

func (m *SMTPSender) authenticatedClient(ctx context.Context) (*smtp.Client, error) {
	address := net.JoinHostPort(m.host, m.port)
	tlsConfig := m.tlsConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{ServerName: m.host}
	}
	connection, err := (&tls.Dialer{Config: tlsConfig.Clone()}).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("smtp dial: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		if err := connection.SetDeadline(deadline); err != nil {
			_ = connection.Close()
			return nil, fmt.Errorf("smtp deadline: %w", err)
		}
	}
	client, err := smtp.NewClient(connection, m.host)
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("smtp client: %w", err)
	}
	if err := client.Auth(smtp.PlainAuth("", m.username, m.password, m.host)); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("smtp auth: %w", err)
	}
	return client, nil
}

var (
	_ Sender        = (*SMTPSender)(nil)
	_ Sender        = (*DevSender)(nil)
	_ HealthChecker = (*SMTPSender)(nil)
	_ HealthChecker = (*DevSender)(nil)
)
