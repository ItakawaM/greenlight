package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
)

const (
	LevelFatal = slog.Level(12)
)

var LevelNames = map[slog.Leveler]string{
	LevelFatal: "FATAL",
}

// New returns a *slog.Logger configured to write structured JSON logs,
// with an added FATAL level for use during startup failures.
func New(out io.Writer, minLevel slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(out, &slog.HandlerOptions{
		Level: minLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				level, ok := a.Value.Any().(slog.Level)
				if ok {
					levelLabel, exists := LevelNames[level]
					if !exists {
						levelLabel = level.String()
					}

					a.Value = slog.StringValue(levelLabel)
				}
			}

			return a
		},
	})

	return slog.New(handler)
}

// LogFatal is a helper function that logs a Level Fatal error using
// slog.Logger and exits with code 1.
func LogFatal(logger *slog.Logger, msg string, attrs ...slog.Attr) {
	logger.LogAttrs(context.Background(), LevelFatal, msg,
		attrs...)
	os.Exit(1)
}
