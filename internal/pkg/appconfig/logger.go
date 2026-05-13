package appconfig

// LogLevel defines the severity level used for log filtering.
type LogLevel string

// LogFormat defines the output format used by the logger.
type LogFormat string

// LogOutput defines the output destination for log entries.
type LogOutput string

const (
	// LogLevelDebug enables verbose diagnostic logging.
	LogLevelDebug LogLevel = "debug"

	// LogLevelInfo enables informational application logs.
	LogLevelInfo LogLevel = "info"

	// LogLevelWarn enables warning logs for recoverable issues.
	LogLevelWarn LogLevel = "warn"

	// LogLevelError enables logging for error conditions only.
	LogLevelError LogLevel = "error"
)

const (
	// LogFormatJSON outputs structured JSON logs.
	LogFormatJSON LogFormat = "json"

	// LogFormatText outputs human-readable text logs.
	LogFormatText LogFormat = "text"

	// LogFormatPretty outputs human-readable text logs with source info and colorization.
	LogFormatPretty LogFormat = "pretty"
)

const (
	// LogOutputStdout writes logs to standard output.
	LogOutputStdout LogOutput = "stdout"

	// LogOutputStderr writes logs to standard error.
	LogOutputStderr LogOutput = "stderr"
)

// LoggerConfig represents application logging configuration.
type LoggerConfig struct {
	// Level defines the minimum log severity that will be written.
	Level LogLevel `env:"LOG_LEVEL" envDefault:"info" validate:"oneof=debug info warn error"`

	// Format defines the log output serialization format.
	Format LogFormat `env:"LOG_FORMAT" envDefault:"json" validate:"oneof=json text pretty"`

	// Output defines the destination stream used for log output.
	Output LogOutput `env:"LOG_OUTPUT" envDefault:"stdout" validate:"oneof=stdout stderr"`

	// AddSource enables source file and line information in log records.
	AddSource bool `env:"LOG_ADD_SOURCE" envDefault:"false"`
}
