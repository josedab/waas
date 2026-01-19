package utils

import (
	"log"
	"os"
)

// NewTestLogger creates a logger for testing that outputs to stdout
func NewTestLogger() *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "[TEST] ", log.LstdFlags),
	}
}