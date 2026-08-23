package main

import (
	"log/slog"
	"os"
	"sync"
	"sync/atomic"

	"github.com/ItakawaM/greenlight/internal/config"
	"github.com/ItakawaM/greenlight/internal/data"
	"github.com/ItakawaM/greenlight/internal/database"
	"github.com/ItakawaM/greenlight/internal/jsonlog"
	"github.com/ItakawaM/greenlight/internal/limiter"
	"github.com/ItakawaM/greenlight/internal/mailer"
	"github.com/ItakawaM/greenlight/internal/metrics"
	"github.com/ItakawaM/greenlight/internal/validator"
)

type application struct {
	config  *config.Config
	logger  *slog.Logger
	models  *data.Models
	limiter *limiter.TokenBucketLimiter
	mailer  *mailer.Mailer

	metrics *metrics.Metrics
	// metricsHealthy represents whether the metrics server, if enabled,
	// managed to start without any errors
	metricsHealthy atomic.Bool

	wg sync.WaitGroup
}

// @title           Greenlight API
// @version         0.1.0
// @description     A JSON REST API for retrieving and managing movie information, built while working through Let's Go Further by Alex Edwards.
// @contact.name    API Support
// @license.name    MIT
// @license.url     https://github.com/ItakawaM/greenlight/blob/main/LICENSE
//
// @BasePath        /v1
//
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Stateful bearer token. Send as: Authorization: Bearer <token>
func main() {
	logger := jsonlog.New(os.Stdout, slog.LevelInfo)

	cfgValidator := validator.New()
	cfg := config.LoadConfig(cfgValidator, logger)
	if !cfgValidator.Valid() {
		var attrs []slog.Attr
		for key, value := range cfgValidator.Errors {
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
	)
	if err != nil {
		jsonlog.LogFatal(logger, "postgres connection failed", slog.String("error", err.Error()))
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
	}
	defer redisClient.Close() // Graceful shutdown will be implemented later

	logger.Info("redis client connection established")

	mailClient, err := mailer.New(
		cfg.SMTP.Host,
		cfg.SMTP.Port,
		cfg.SMTP.Username,
		cfg.SMTP.Password,
		cfg.SMTP.From,
		logger,
	)
	if err != nil {
		jsonlog.LogFatal(logger, "smtp client setup failed", slog.String("error", err.Error()))
	}

	logger.Info("smtp client established")

	// optional metrics
	var metricsReg *metrics.Metrics = nil
	if cfg.Metrics.Enabled {
		metricsReg = metrics.NewMetrics()
	}

	// optional rate limiting
	var limiterBucket *limiter.TokenBucketLimiter = nil
	if cfg.Limiter.Enabled {
		limiterBucket = limiter.New(
			redisClient,
			cfg.Limiter.Burst,
			cfg.Limiter.RPS,
		)
	}

	app := &application{
		config:  cfg,
		logger:  logger,
		models:  data.NewModels(postgres),
		limiter: limiterBucket,
		mailer:  mailClient,
		metrics: metricsReg,
	}

	if app.config.Metrics.Enabled {
		app.metricsHealthy.Store(true)
	}

	if err = app.serve(); err != nil {
		jsonlog.LogFatal(logger, "server failed", slog.String("error", err.Error()))
	}
}
