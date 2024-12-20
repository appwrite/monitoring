package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
)

type Logger struct {
	logger *log.Logger
}

func New() *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) formatMessage(level, format string, args ...interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	return fmt.Sprintf("%s [%s] %s", timestamp, level, message)
}

func (l *Logger) Log(format string, args ...interface{}) {
	msg := l.formatMessage("LOG", format, args...)
	l.logger.Printf("%s", msg)
}

func (l *Logger) Success(format string, args ...interface{}) {
	msg := l.formatMessage("SUCCESS", format, args...)
	l.logger.Printf("%s%s%s", colorGreen, msg, colorReset)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	msg := l.formatMessage("WARNING", format, args...)
	l.logger.Printf("%s%s%s", colorYellow, msg, colorReset)
}

func (l *Logger) Error(format string, args ...interface{}) {
	msg := l.formatMessage("ERROR", format, args...)
	l.logger.Printf("%s%s%s", colorRed, msg, colorReset)
}

func (l *Logger) Info(format string, args ...interface{}) {
	msg := l.formatMessage("INFO", format, args...)
	l.logger.Printf("%s%s%s", colorBlue, msg, colorReset)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	msg := l.formatMessage("DEBUG", format, args...)
	l.logger.Printf("%s%s%s", colorCyan, msg, colorReset)
}

func (l *Logger) Fatal(format string, args ...interface{}) {
	msg := l.formatMessage("FATAL", format, args...)
	l.logger.Fatalf("%s%s%s", colorPurple, msg, colorReset)
} 