package log

import (
	"fmt"
	"os"
	"sync"
)

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

type Logger struct {
	mu    sync.Mutex
	level Level
}

var defaultLogger = &Logger{level: InfoLevel}

func SetLevel(lvl string) {
	switch lvl {
	case "debug":
		defaultLogger.level = DebugLevel
	case "info":
		defaultLogger.level = InfoLevel
	case "warn":
		defaultLogger.level = WarnLevel
	case "error":
		defaultLogger.level = ErrorLevel
	default:
		defaultLogger.level = InfoLevel
	}
}

func output(lvl Level, format string, args ...interface{}) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	if lvl < defaultLogger.level {
		return
	}
	fmt.Fprintf(os.Stdout, "%s %s\n", levelPrefix(lvl), fmt.Sprintf(format, args...))
}

func levelPrefix(lvl Level) string {
	switch lvl {
	case DebugLevel:
		return "[DEBUG]"
	case InfoLevel:
		return "[INFO]"
	case WarnLevel:
		return "[WARN]"
	case ErrorLevel:
		return "[ERROR]"
	}
	return "[INFO]"
}

func Debug(format string, args ...interface{}) { output(DebugLevel, format, args...) }
func Info(format string, args ...interface{})  { output(InfoLevel, format, args...) }
func Warn(format string, args ...interface{})  { output(WarnLevel, format, args...) }
func Error(format string, args ...interface{}) { output(ErrorLevel, format, args...) }
