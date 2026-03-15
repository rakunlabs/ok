package ok

// Logger is the interface for structured logging used throughout ok.
// It follows the slog-style pattern with key-value pairs.
type Logger interface {
	Error(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Debug(msg string, keysAndValues ...any)
}

// NoopLogger is a logger that discards all messages.
// Use it to disable logging entirely.
//
//	client, err := ok.New(ok.WithLogger(ok.NoopLogger{}))
type NoopLogger struct{}

func (NoopLogger) Error(string, ...any) {}
func (NoopLogger) Warn(string, ...any)  {}
func (NoopLogger) Info(string, ...any)  {}
func (NoopLogger) Debug(string, ...any) {}
