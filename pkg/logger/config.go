package logger

// Level defines the severity level used for log filtering.
type Level string

// Format defines the output format used by the logger.
type Format string

// Output defines the output destination for log entries.
type Output string

const (
	// LevelDebug enables verbose diagnostic logging.
	LevelDebug Level = "debug"

	// LevelInfo enables informational application logs.
	LevelInfo Level = "info"

	// LevelWarn enables warning logs for recoverable issues.
	LevelWarn Level = "warn"

	// LevelError enables logging for error conditions only.
	LevelError Level = "error"
)

const (
	// FormatJSON outputs structured JSON logs.
	FormatJSON Format = "json"

	// FormatText outputs human-readable text logs.
	FormatText Format = "text"

	// FormatPretty outputs human-readable text logs with source info and colorization.
	FormatPretty Format = "pretty"
)

const (
	// OutputStdout writes logs to standard output.
	OutputStdout Output = "stdout"

	// OutputStderr writes logs to standard error.
	OutputStderr Output = "stderr"
)

// Config represents application logging configuration.
type Config struct {
	// Level defines the minimum log severity that will be written.
	Level Level `env:"LOG_LEVEL" envDefault:"info" validate:"oneof=debug info warn error"`

	// Format defines the log output serialization format.
	Format Format `env:"LOG_FORMAT" envDefault:"json" validate:"oneof=json text pretty"`

	// Output defines the destination stream used for log output.
	Output Output `env:"LOG_OUTPUT" envDefault:"stdout" validate:"oneof=stdout stderr"`

	// AddSource enables source file and line information in log records.
	AddSource bool `env:"LOG_ADD_SOURCE" envDefault:"false"`
}
