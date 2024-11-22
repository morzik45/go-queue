package logs

import (
	"context"
	"fmt"
	"github.com/morzik45/go-queue/pkg/utils"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/viper"
)

var slogLevel = new(slog.LevelVar)

func Init(ctx context.Context, isDev bool, cfg *viper.Viper) {

	var (
		path  string
		level = "info"
	)

	if cfg != nil {
		path = cfg.GetString("path")

		if cfg.IsSet("level") {
			level = cfg.GetString("level")
		}
	}
	SetLevel(level)

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     slogLevel,
	}

	logWriter := os.Stdout
	if path != "" && !isDev {
		var err error
		logWriter, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			slog.Error("failed to open log file", slog.String("path", path), slog.Any("error", err))
			panic(err)
		} else {
			go utils.CloseOnContext(ctx, logWriter)
		}
	}
	var handler slog.Handler
	if isDev {
		handler = slog.NewTextHandler(logWriter, opts)
	} else {
		handler = slog.NewJSONHandler(logWriter, opts)
	}

	handler = NewCtxHandler(handler) // add context values
	slog.SetDefault(slog.New(handler))

	return
}

func SetLevel(level string) {
	l, err := levelFromString(level)
	if err != nil {
		slog.Error("failed to set log level", slog.String("level", level), slog.Any("error", err))
		return
	}
	slogLevel.Set(l)
}

func levelFromString(level string) (slog.Level, error) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug, nil
	case "INFO":
		return slog.LevelInfo, nil
	case "WARN":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid log level: %s", level)
	}
}
