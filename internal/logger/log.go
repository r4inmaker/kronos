package logger

import (
	"fmt"
	"time"
)

// ANSI colors
const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	gray   = "\033[90m"
)

// Logger struct (optional component name)
type Logger struct {
	component string
}

// Create new logger
func NewLogger(component string) *Logger {
	return &Logger{component: component}
}

// internal print
func (l *Logger) log(level string, color string, msg string) {
	timestamp := time.Now().Format("15:04:05")

	if l.component != "" {
		fmt.Printf("%s[%s]%s %s[%s]%s %s%s%s\n",
			gray, timestamp, reset,
			color, level, reset,
			blue+"["+l.component+"] "+reset,
			msg,
			reset,
		)
	} else {
		fmt.Printf("%s[%s]%s %s[%s]%s %s\n",
			gray, timestamp, reset,
			color, level, reset,
			msg,
		)
	}
}

// Public methods
func (l *Logger) Info(msg string) {
	l.log("INFO", green, msg)
}

func (l *Logger) Warn(msg string) {
	l.log("WARN", yellow, msg)
}

func (l *Logger) Error(msg string) {
	l.log("ERROR", red, msg)
}

func (l *Logger) Debug(msg string) {
	l.log("DEBUG", blue, msg)
}