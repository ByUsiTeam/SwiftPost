package utils

import (
	"fmt"
	"time"
)

// 颜色常量
const (
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"
	ColorReset   = "\033[0m"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

var logLevelNames = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

var logLevelColors = map[LogLevel]string{
	DEBUG: ColorCyan,
	INFO:  ColorGreen,
	WARN:  ColorYellow,
	ERROR: ColorRed,
	FATAL: ColorMagenta,
}

type Logger struct {
	level LogLevel
	name  string
}

func NewLogger(name string) *Logger {
	return &Logger{
		level: INFO,
		name:  name,
	}
}

func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	levelName := logLevelNames[level]
	levelColor := logLevelColors[level]
	
	message := fmt.Sprintf(format, args...)
	
	fmt.Printf("%s[%s]%s %s%s%s [%s] %s\n",
		ColorCyan, timestamp, ColorReset,
		levelColor, levelName, ColorReset,
		l.name, message)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
}

// 全局日志函数
var defaultLogger = NewLogger("SwiftPost")

func SetLogLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}

func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
	panic(fmt.Sprintf(format, args...))
}

// 彩色输出函数
func PrintColored(text string, length int, color string) {
	if length > 0 {
		fmt.Printf("%s%s%s\n", color, text, ColorReset)
	} else {
		fmt.Printf("%s%s%s\n", color, text, ColorReset)
	}
}

func PrintSection(title string) {
	PrintColored("=", 60, ColorCyan)
	PrintColored(title, 0, ColorGreen)
	PrintColored("=", 60, ColorCyan)
}

func PrintSuccess(message string) {
	PrintColored("✅ "+message, 0, ColorGreen)
}

func PrintWarning(message string) {
	PrintColored("⚠️  "+message, 0, ColorYellow)
}

func PrintError(message string) {
	PrintColored("❌ "+message, 0, ColorRed)
}

func PrintInfo(message string) {
	PrintColored("ℹ️  "+message, 0, ColorBlue)
}