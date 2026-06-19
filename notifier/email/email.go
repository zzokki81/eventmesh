// Package email delivers notification emails over SMTP. It knows how to send a
// message, not what the message says — callers build the content.
package email

import "context"

// Sender delivers email messages. Consumers depend on this interface;
// *SMTPSender is its go-mail-backed implementation.
type Sender interface {
	// Send delivers msg, returning an error if delivery fails.
	Send(ctx context.Context, msg Message) error
}
