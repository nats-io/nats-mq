package logging

import (
	"log"
)

// NewNilLogger creates a nil logger for testing
func NewNilLogger() Logger {
	return &NilLogger{}
}

// NilLogger - only implements Fatal
type NilLogger struct {
}

// Close no-op
func (NilLogger) Close() error { return nil }

// Debugf no-op
func (NilLogger) Debugf(format string, v ...interface{}) {}

// Errorf no-op
func (NilLogger) Errorf(format string, v ...interface{}) {}

// Fatalf calls log.Fatalf
func (NilLogger) Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

// Noticef no-op
func (NilLogger) Noticef(format string, v ...interface{}) {}

// Tracef no-op
func (NilLogger) Tracef(format string, v ...interface{}) {}

// Warnf no-op
func (NilLogger) Warnf(format string, v ...interface{}) {}
