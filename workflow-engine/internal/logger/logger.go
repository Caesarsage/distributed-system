package logger

import (
	"log"
	"os"
)

// Logger provides simple structured logging
type Logger struct {
	*log.Logger
}

// New creates a new logger
func New(prefix string) *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, prefix+" ", log.LstdFlags|log.Lshortfile),
	}
}

// WithContext creates a logger with additional context
func (l *Logger) WithContext(context string) *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, l.Prefix()+" ["+context+"] ", log.LstdFlags|log.Lshortfile),
	}
}

// Prefix returns the logger prefix
func (l *Logger) Prefix() string {
	return l.Logger.Prefix()
}

