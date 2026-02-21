package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

func Init(debug bool) {
	zerolog.TimeFieldFormat = time.RFC3339

	if debug {
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

func Debug(msg string) {
	log.Debug().Msg(msg)
}

func Debugf(format string, v ...interface{}) {
	log.Debug().Msgf(format, v...)
}

func Info(msg string) {
	log.Info().Msg(msg)
}

func Infof(format string, v ...interface{}) {
	log.Info().Msgf(format, v...)
}

func Warn(msg string) {
	log.Warn().Msg(msg)
}

func Warnf(format string, v ...interface{}) {
	log.Warn().Msgf(format, v...)
}

func Error(msg string, err error) {
	log.Error().Err(err).Msg(msg)
}

func Errorf(format string, v ...interface{}) {
	log.Error().Msgf(format, v...)
}

func Fatal(msg string, err error) {
	log.Fatal().Err(err).Msg(msg)
}

func Fatalf(format string, v ...interface{}) {
	log.Fatal().Msgf(format, v...)
}

func WithField(key string, value interface{}) zerolog.Logger {
	return log.With().Interface(key, value).Logger()
}

func WithFields(fields map[string]interface{}) zerolog.Logger {
	ctx := log.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return ctx.Logger()
}
