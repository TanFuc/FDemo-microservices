package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

// Init initializes the logger with the given environment.
func Init(env string) {
	zerolog.TimeFieldFormat = time.RFC3339

	if env == "development" {
		log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
			With().
			Timestamp().
			Caller().
			Logger()
	} else {
		log = zerolog.New(os.Stdout).
			With().
			Timestamp().
			Logger()
	}
}

// Debug returns a debug event.
func Debug() *zerolog.Event {
	return log.Debug()
}

// Info returns an info event.
func Info() *zerolog.Event {
	return log.Info()
}

// Warn returns a warn event.
func Warn() *zerolog.Event {
	return log.Warn()
}

// Error returns an error event.
func Error() *zerolog.Event {
	return log.Error()
}

// Fatal returns a fatal event.
func Fatal() *zerolog.Event {
	return log.Fatal()
}

// With returns a context with the given fields.
func With() zerolog.Context {
	return log.With()
}
