package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/nonobeam/golang-stock-trading/internal/config"
)

var logger zerolog.Logger

func Init(cfg *config.Config) {
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)

	if cfg.AppEnv == "development" {
		logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}).With().
			Timestamp().
			Str("app", cfg.AppName).
			Str("env", cfg.AppEnv).
			Logger()
	} else {
		logger = zerolog.New(os.Stdout).With().
			Timestamp().
			Str("app", cfg.AppName).
			Str("env", cfg.AppEnv).
			Logger()
	}

	log.Logger = logger
}

func Get() *zerolog.Logger {
	return &logger
}

func Info() *zerolog.Event {
	return logger.Info()
}

func Error() *zerolog.Event {
	return logger.Error()
}

func Debug() *zerolog.Event {
	return logger.Debug()
}

func Warn() *zerolog.Event {
	return logger.Warn()
}

func Fatal() *zerolog.Event {
	return logger.Fatal()
}

func WithField(key string, value interface{}) zerolog.Logger {
	return logger.With().Interface(key, value).Logger()
}
