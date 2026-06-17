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
	mu       sync.Mutex
	level    Level
	jsonMode bool
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

func SetJSONMode(enabled bool) {
	defaultLogger.jsonMode = enabled
}

func output(lvl Level, format string, args ...interface{}) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	if lvl < defaultLogger.level {
		return
	}
	prefix := ""
	switch lvl {
	case DebugLevel:
		prefix = "[DEBUG]"
	case InfoLevel:
		prefix = "[INFO]"
	case WarnLevel:
		prefix = "[WARN]"
	case ErrorLevel:
		prefix = "[ERROR]"
	}
	msg := fmt.Sprintf(format, args...)
	if defaultLogger.jsonMode {
	 lvlStr := "info"
		switch lvl {
		case DebugLevel:
			lvlStr = "debug"
		case WarnLevel:
			lvlStr = "warn"
		case ErrorLevel:
			lvlStr = "error"
	 }
		fmt.Fprintf(os.Stdout, `{"level":"%s","msg":"%s"}`+"\n", lvlStr, msg)
	} else {
		fmt.Fprintf(os.Stdout, "%s %s\n", prefix, msg)
	}
}

func Debug(format string, args ...interface{}) { output(DebugLevel, format, args...) }
func Info(format string, args ...interface{})  { output(InfoLevel, format, args...) }
func Warn(format string, args ...interface{})  { output(WarnLevel, format, args...) }
func Error(format string, args ...interface{}) { output(ErrorLevel, format, args...) }