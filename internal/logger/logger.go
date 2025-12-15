// Package logger provides structured logging functionality for the application.
package logger

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus with application-specific functionality.
type Logger struct {
	log       *logrus.Logger
	fields    logrus.Fields
	startTime time.Time
}

// New creates a new Logger instance with the specified log level.
func New(level string) *Logger {
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true,
	})

	// Parse and set log level
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	log.SetLevel(lvl)

	return &Logger{
		log:       log,
		fields:    logrus.Fields{},
		startTime: time.Now(),
	}
}

// WithField returns a new logger with an additional field.
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newFields := make(logrus.Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value
	return &Logger{
		log:       l.log,
		fields:    newFields,
		startTime: l.startTime,
	}
}

// WithFields returns a new logger with additional fields.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newFields := make(logrus.Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	return &Logger{
		log:       l.log,
		fields:    newFields,
		startTime: l.startTime,
	}
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, fields map[string]interface{}) {
	entry := l.log.WithFields(l.fields)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	entry.Debug(msg)
}

// Info logs an info message.
func (l *Logger) Info(msg string, fields map[string]interface{}) {
	entry := l.log.WithFields(l.fields)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	entry.Info(msg)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, fields map[string]interface{}) {
	entry := l.log.WithFields(l.fields)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	entry.Warn(msg)
}

// Error logs an error message.
func (l *Logger) Error(msg string, fields map[string]interface{}) {
	entry := l.log.WithFields(l.fields)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	entry.Error(msg)
}

// Fatal logs a fatal message and exits.
func (l *Logger) Fatal(msg string, fields map[string]interface{}) {
	entry := l.log.WithFields(l.fields)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	entry.Fatal(msg)
}

// StartOperation logs the start of an operation and returns a function to log its completion.
func (l *Logger) StartOperation(name string) func(error) {
	start := time.Now()
	l.Info("Starting operation", map[string]interface{}{
		"operation": name,
	})
	return func(err error) {
		duration := time.Since(start)
		if err != nil {
			l.Error("Operation failed", map[string]interface{}{
				"operation": name,
				"duration":  duration.String(),
				"error":     err.Error(),
			})
		} else {
			l.Info("Operation completed", map[string]interface{}{
				"operation": name,
				"duration":  duration.String(),
			})
		}
	}
}

// GetUptime returns the duration since the logger was created.
func (l *Logger) GetUptime() time.Duration {
	return time.Since(l.startTime)
}

// SetLevel changes the log level.
func (l *Logger) SetLevel(level string) {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return
	}
	l.log.SetLevel(lvl)
}

// SetJSONFormat switches to JSON formatting.
func (l *Logger) SetJSONFormat() {
	l.log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})
}

// SetTextFormat switches to text formatting.
func (l *Logger) SetTextFormat() {
	l.log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true,
	})
}

// SetOutput sets the log output destination.
func (l *Logger) SetOutput(output *os.File) {
	l.log.SetOutput(output)
}
