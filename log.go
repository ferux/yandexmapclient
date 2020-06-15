package yandexmapclient

import (
	"io"
	"log"
)

type ModuleLogger interface {
	Logger

	Module(name string) ModuleLogger
}

type moduleLogger struct {
	next *log.Logger

	out io.Writer
}

func NewLogger(out io.Writer) ModuleLogger {
	return moduleLogger{
		out:  out,
		next: log.New(out, "", log.Lshortfile),
	}
}

func (m moduleLogger) Module(name string) ModuleLogger {
	return moduleLogger{
		out:  m.out,
		next: log.New(m.out, name, log.Lshortfile),
	}
}

// Debug implements ModuleLogger interface.
func (m moduleLogger) Debug(msg string) {
	m.next.Print(msg)
}

// Debugf implements ModuleLogger interface.
func (m moduleLogger) Debugf(format string, args ...interface{}) {
	m.next.Printf(format, args...)
}

type loggerWrapper struct{ Logger }

// Module implements ModuleLogger interface by wrapping logger interafce
// with noop method.
func (l loggerWrapper) Module(_ string) ModuleLogger {
	return l
}

// ModuleLoggerWrapper make Logger interface implement ModuleLogger.
func ModuleLoggerWrapper(log Logger) ModuleLogger {
	return loggerWrapper{log}
}
