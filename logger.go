package main

import (
	"os"

	"github.com/phuslu/log"
)

type logger struct {
	log     log.Logger
	verbose bool
}

func newLogger(v bool) *logger {
	l := logger{verbose: v}
	level := log.InfoLevel
	if v {
		level = log.TraceLevel
	}
	if log.IsTerminal(os.Stderr.Fd()) {
		l.log = log.Logger{
			Level:      level,
			TimeFormat: "15:04:05",
			Caller:     1,
			Writer: &log.ConsoleWriter{
				ColorOutput:    true,
				QuoteString:    true,
				EndWithMessage: true,
			},
		}
	} else {
		l.log = log.Logger{
			Level: level,
		}
	}
	return &l
}

func (l *logger) Debug() *log.Entry {
	return l.log.Debug()
}

func (l *logger) Info() *log.Entry {
	return l.log.Info()
}

func (l *logger) Warn() *log.Entry {
	return l.log.Warn()
}

func (l *logger) Error() *log.Entry {
	return l.log.Error()
}
