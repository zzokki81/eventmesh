package email

// Message is an email to be delivered. A message should set at least one of
// Plain or HTML; if both are set, clients show HTML and fall back to Plain.
type Message struct {
	// To is the recipient address.
	To string

	// Subject is the email subject line.
	Subject string

	// Plain is the plain-text body.
	Plain string

	// HTML is the optional HTML body.
	HTML string
}
