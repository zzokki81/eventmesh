package email

import (
	"context"
	"fmt"

	"github.com/wneessen/go-mail"

	"github.com/zzokki81/eventmesh/notifier/config"
)

// SMTPSender delivers messages over SMTP using the go-mail client.
type SMTPSender struct {
	client *mail.Client
	from   string
}

// NewSMTPSender builds an SMTPSender from cfg. It configures authentication
// only when a username is set, and disables TLS (suitable for a local Mailpit;
// a real relay would require a TLS policy and credentials).
func NewSMTPSender(cfg config.SMTPConfig) (*SMTPSender, error) {
	opts := []mail.Option{
		mail.WithPort(cfg.Port),
		mail.WithTLSPortPolicy(mail.NoTLS),
	}

	if cfg.Username != "" {
		opts = append(opts,
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(cfg.Username),
			mail.WithPassword(cfg.Password),
		)
	}

	client, err := mail.NewClient(cfg.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("create mail client: %w", err)
	}

	return &SMTPSender{client: client, from: cfg.From}, nil
}

// Send builds the MIME message and delivers it, opening and closing the SMTP
// connection for this send.
func (s *SMTPSender) Send(ctx context.Context, msg Message) error {
	m := mail.NewMsg()
	if err := m.From(s.from); err != nil {
		return fmt.Errorf("set from: %w", err)
	}
	if err := m.To(msg.To); err != nil {
		return fmt.Errorf("set to: %w", err)
	}
	m.Subject(msg.Subject)

	switch {
	case msg.Plain != "" && msg.HTML != "":
		m.SetBodyString(mail.TypeTextPlain, msg.Plain)
		m.AddAlternativeString(mail.TypeTextHTML, msg.HTML)
	case msg.HTML != "":
		m.SetBodyString(mail.TypeTextHTML, msg.HTML)
	default:
		m.SetBodyString(mail.TypeTextPlain, msg.Plain)
	}

	if err := s.client.DialAndSendWithContext(ctx, m); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

// Compile-time check that *SMTPSender implements Sender.
var _ Sender = (*SMTPSender)(nil)
