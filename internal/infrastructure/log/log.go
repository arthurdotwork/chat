package log

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

func Config(ctx context.Context) {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == "level" {
				lowerCaseLevel := strings.ToLower(a.Value.String())

				return slog.Attr{
					Key:   "severity",
					Value: slog.StringValue(lowerCaseLevel),
				}
			}

			if a.Key == "msg" {
				return slog.Attr{
					Key:   "message",
					Value: a.Value,
				}
			}

			return a
		},
	})

	logger := slog.New(jsonHandler)
	slog.SetDefault(logger)
}
