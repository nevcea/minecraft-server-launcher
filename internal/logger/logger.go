package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

type Level int

const (
	LevelTrace Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelNone
)

var (
	currentLevel = LevelInfo
	mu           sync.Mutex
	logFile      *os.File
)

func ParseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "trace":
		return LevelTrace
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "none", "off":
		return LevelNone
	default:
		return LevelInfo
	}
}

func SetLevel(level Level) {
	mu.Lock()
	defer mu.Unlock()
	currentLevel = level
}

func SetLogFile(path string) error {
	mu.Lock()
	defer mu.Unlock()

	if logFile != nil {
		logFile.Close()
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	logFile = f
	log.SetOutput(f)
	return nil
}

func Close() {
	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

func logMsg(level Level, format string, args ...interface{}) {
	if level < currentLevel {
		return
	}

	prefix := ""
	switch level {
	case LevelTrace:
		prefix = "[TRACE]"
	case LevelDebug:
		prefix = "[DEBUG]"
	case LevelInfo:
		prefix = "[INFO]"
	case LevelWarn:
		prefix = "[WARN]"
	case LevelError:
		prefix = "[ERROR]"
	}

	msg := fmt.Sprintf(format, args...)

	// Console output
	fmt.Printf("%s %s\n", prefix, msg)

	// File output (via standard log package)
	if logFile != nil {
		log.Printf("%s %s", prefix, msg)
	}
}

func Trace(format string, args ...interface{}) {
	logMsg(LevelTrace, format, args...)
}

func Debug(format string, args ...interface{}) {
	logMsg(LevelDebug, format, args...)
}

func Info(format string, args ...interface{}) {
	logMsg(LevelInfo, format, args...)
}

func Warn(format string, args ...interface{}) {
	logMsg(LevelWarn, format, args...)
}

func Error(format string, args ...interface{}) {
	logMsg(LevelError, format, args...)
}

// Fatal logs error and exits
func Fatal(format string, args ...interface{}) {
	logMsg(LevelError, format, args...)
	Close()
	os.Exit(1)
}
