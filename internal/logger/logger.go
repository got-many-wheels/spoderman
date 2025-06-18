package logger

import (
	"os"

	"github.com/phuslu/log"
)

type Logger struct {
	log log.Logger
}

func New(verbose bool) *Logger {
	l := Logger{}
	level := log.InfoLevel
	if verbose {
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

func (l *Logger) ToVerbose(v bool) {
	newLogger := New(v)
	l.log = newLogger.log
}

func (l *Logger) Debug() *log.Entry {
	return l.log.Debug()
}

func (l *Logger) Info() *log.Entry {
	return l.log.Info()
}

func (l *Logger) Warn() *log.Entry {
	return l.log.Warn()
}

func (l *Logger) Error() *log.Entry {
	return l.log.Error()
}
