package main

import (
	"fmt"
	"io"
	"log"
	"os"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

type Logger interface {
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
}

type baseLogger struct {
	logger *log.Logger
	debug  bool
}

func (b *baseLogger) logf(level string, format string, args ...interface{}) {
	prefix := level + ": "
	// Output depth 3 to attribute file:line to the original caller of Infof/Warnf/Errorf/Debugf
	_ = b.logger.Output(3, prefix+fmt.Sprintf(format, args...))
}

func (b *baseLogger) Infof(format string, args ...interface{})  { b.logf("INFO", format, args...) }
func (b *baseLogger) Warnf(format string, args ...interface{})  { b.logf("WARN", format, args...) }
func (b *baseLogger) Errorf(format string, args ...interface{}) { b.logf("ERROR", format, args...) }
func (b *baseLogger) Debugf(format string, args ...interface{}) {
	if b.debug {
		b.logf("DEBUG", format, args...)
	}
}

// NewStdoutLogger returns a logger that writes to stdout (timestamp + microseconds + shortfile)
func NewStdoutLogger() Logger {
	l := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	return &baseLogger{logger: l}
}

// NewRotatingFileLogger returns a logger that writes to a rotating file using lumberjack
func NewRotatingFileLogger(maxSizeMB, maxBackups, maxAgeDays int, compress bool) Logger {
	w := &lumberjack.Logger{
		Filename:   "/var/log/ebpf-game/ebpf-game.log",
		MaxSize:    maxSizeMB,
		MaxBackups: maxBackups,
		MaxAge:     maxAgeDays,
		Compress:   compress,
	}
	l := log.New(w, "", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	return &baseLogger{logger: l}
}

// NewStdoutAndFileLogger writes to both stdout and a rotating file
func NewStdoutAndFileLogger(maxSizeMB, maxBackups, maxAgeDays int, compress bool) Logger {
	fileW := &lumberjack.Logger{
		Filename:   "/var/log/ebpf-game/ebpf-game.log",
		MaxSize:    maxSizeMB,
		MaxBackups: maxBackups,
		MaxAge:     maxAgeDays,
		Compress:   compress,
	}
	mw := io.MultiWriter(os.Stdout, fileW)
	l := log.New(mw, "", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	return &baseLogger{logger: l}
}

// EnableDebug wraps a logger enabling debug prints
func EnableDebug(l Logger) Logger {
	if bl, ok := l.(*baseLogger); ok {
		bl.debug = true
	}
	return l
}