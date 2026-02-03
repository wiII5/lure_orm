package logger

// LogLevel controls which types of queries are logged.
type LogLevel int

const (
	// LogLevelNone disables all query logging.
	LogLevelNone LogLevel = iota
	// LogLevelRead logs only read queries.
	LogLevelRead
	// LogLevelWrite logs only write queries.
	LogLevelWrite
	// LogLevelAll logs all queries.
	LogLevelAll
)

// Config holds logger configuration.
type Config struct {
	level  LogLevel
	fields map[string]any
}

// Option is a functional option for configuring the logger.
type Option func(*Config)

func newConfig(opts ...Option) *Config {
	cfg := &Config{
		level:  LogLevelAll,
		fields: make(map[string]any),
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// WithLogLevel sets the log level.
func WithLogLevel(level LogLevel) Option {
	return func(cfg *Config) {
		cfg.level = level
	}
}

// WithFields sets additional fields to include in log output.
func WithFields(fields map[string]any) Option {
	return func(cfg *Config) {
		cfg.fields = fields
	}
}

func (l LogLevel) allowRead() bool {
	return l == LogLevelAll || l == LogLevelRead
}

func (l LogLevel) allowWrite() bool {
	return l == LogLevelAll || l == LogLevelWrite
}
