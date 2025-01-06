package pubsub

import "github.com/hibiken/asynq"

type NilLogger struct{}

func NewNilLogger() asynq.Logger {
	return &NilLogger{}
}

// Debug logs a message at Debug level.
func (l *NilLogger) Debug(args ...interface{}) {
	// do nothing...
}

// Info logs a message at Info level.
func (l *NilLogger) Info(args ...interface{}) {
	// do nothing...
}

// Warn logs a message at Warning level.
func (l *NilLogger) Warn(args ...interface{}) {
	// do nothing...
}

// Error logs a message at Error level.
func (l *NilLogger) Error(args ...interface{}) {
	// do nothing...
}

// Fatal logs a message at Fatal level
// and process will exit with status set to 1.
func (l *NilLogger) Fatal(args ...interface{}) {
	// do nothing...
}
