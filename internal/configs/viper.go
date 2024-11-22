package configs

import (
	"context"
	"github.com/fsnotify/fsnotify"
	"github.com/morzik45/go-queue/internal/logs"
	"github.com/spf13/viper"
	"log/slog"
)

func GetConfig(ctx context.Context) *viper.Viper {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/app/")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		panic(err)
	}

	// Инициализируем логирование (не уверен, что это надо делать тут, но только тут мы получаем env)
	logs.Init(ctx, v.GetBool("is_dev"), v.Sub("logging"))

	v.OnConfigChange(func(e fsnotify.Event) {
		slog.Info("Config file changed:", slog.String("path", e.Name))
		logs.SetLevel(v.GetString("logging.level"))
	})
	v.WatchConfig()

	slog.Info("Using config file:", slog.String("path", v.ConfigFileUsed()))
	return v
}
