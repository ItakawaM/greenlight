package main

import (
	"log/slog"
	"os"

	"github.com/ItakawaM/greenlight/internal/config"
	"github.com/ItakawaM/greenlight/internal/database"
	"github.com/ItakawaM/greenlight/internal/jsonlog"
	"github.com/ItakawaM/greenlight/internal/metrics"
	"github.com/ItakawaM/greenlight/internal/validator"
)

type exporter struct {
	config  *config.Config
	logger  *slog.Logger
	metrics *metrics.ExporterMetrics
}

func main() {
	logger := jsonlog.New(os.Stdout, slog.LevelInfo)

	v := validator.New()
	cfg := config.Load(logger).
		WithMetrics(v).
		WithPostgreSQL(v).
		WithRedis(v)
	if !v.Valid() {
		var attrs []slog.Attr
		for key, value := range v.Errors {
			attrs = append(attrs, slog.Any(key, value))
		}
		jsonlog.LogFatal(logger, "invalid values provided in settings", attrs...)
	}
	logger.Info("settings loaded successfully")

	postgres, err := database.NewPostgreSQL(
		cfg.PostgreSQL.Host,
		cfg.PostgreSQL.Port,
		cfg.PostgreSQL.Username,
		cfg.PostgreSQL.Password,
		cfg.PostgreSQL.Database,
		cfg.PostgreSQL.MaxOpenConns,
		cfg.PostgreSQL.MaxIdleTime,
		cfg.PostgreSQL.SSL,
	)
	if err != nil {
		jsonlog.LogFatal(logger, "postgres connection failed", slog.String("error", err.Error()))
		return
	}
	defer postgres.Close() // Graceful shutdown will be implemented later

	logger.Info("postgres connection pool established")

	redisClient, err := database.NewRedis(
		cfg.Redis.Host,
		cfg.Redis.Port,
		cfg.Redis.Password,
	)
	if err != nil {
		jsonlog.LogFatal(logger, "redis connection failed", slog.String("error", err.Error()))
		return
	}
	defer redisClient.Close() // Graceful shutdown will be implemented later

	logger.Info("redis client connection established")

	service := &exporter{
		config:  cfg,
		logger:  logger,
		metrics: metrics.NewExporterMetrics(redisClient, postgres),
	}

	if err := service.serve(); err != nil {
		jsonlog.LogFatal(logger, "server failed", slog.String("error", err.Error()))
	}
}
