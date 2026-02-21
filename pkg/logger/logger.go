package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

// Init initializes the logger with the given environment.
// In development mode, it uses a console writer with colors and caller info.
// In production mode, it outputs JSON for log aggregation.
func Init(env string) {
	zerolog.TimeFieldFormat = time.RFC3339

	if env == "development" {
		log = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}).With().Timestamp().Caller().Logger()
	} else {
		log = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}
}

// Debug returns a debug level event.
func Debug() *zerolog.Event {
	return log.Debug()
}

// Info returns an info level event.
func Info() *zerolog.Event {
	return log.Info()
}

// Warn returns a warning level event.
func Warn() *zerolog.Event {
	return log.Warn()
}

// Error returns an error level event.
func Error() *zerolog.Event {
	return log.Error()
}

// Fatal returns a fatal level event.
// Calling Msg() on the returned event will cause os.Exit(1).
func Fatal() *zerolog.Event {
	return log.Fatal()
}

// With returns the logger context for adding fields.
func With() zerolog.Context {
	return log.With()
}

// Logger returns the underlying zerolog.Logger instance.
func Logger() zerolog.Logger {
	return log
}

// SetGlobalLevel sets the global log level.
func SetGlobalLevel(level zerolog.Level) {
	zerolog.SetGlobalLevel(level)
}
