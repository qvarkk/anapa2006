package logger

import (
	"log/slog"
	"os"
)

type Slogger struct {
	Logger *slog.Logger
}

func NewSlogger(debug bool) *Slogger {
	var level slog.Leveler
	if debug {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		AddSource: debug,
		Level:     level,
	}

	return &Slogger{
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, opts)),
	}
}
