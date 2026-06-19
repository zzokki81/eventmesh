package config

// SMTPConfig holds SMTP connection settings. Defaults target a local Mailpit
// instance, which accepts mail without auth or TLS and shows it in a web UI.
type SMTPConfig struct {
	// Host is the SMTP server hostname.
	Host string `env:"SMTP_HOST" envDefault:"127.0.0.1" validate:"required"`

	// Port is the SMTP server port (1025 is Mailpit's default).
	Port int `env:"SMTP_PORT" envDefault:"1025" validate:"required,min=1,max=65535"`

	// From is the sender address used on every message.
	From string `env:"SMTP_FROM" envDefault:"no-reply@eventmesh.local" validate:"required,email"`

	// Username is the optional SMTP account. When empty (e.g. Mailpit), the
	// client connects without authentication.
	Username string `env:"SMTP_USERNAME"`

	// Password is the optional SMTP password, used only when Username is set.
	Password string `env:"SMTP_PASSWORD"`
}
