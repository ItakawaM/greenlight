package main

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ItakawaM/greenlight/internal/data"
	"github.com/ItakawaM/greenlight/internal/jsonlog"
	"github.com/ItakawaM/greenlight/internal/limiter"
	"github.com/ItakawaM/greenlight/internal/mailer"
	"github.com/ItakawaM/greenlight/internal/metrics"
	"github.com/ItakawaM/greenlight/internal/validator"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type application struct {
	config  config
	logger  *slog.Logger
	models  *data.Models
	limiter *limiter.TokenBucketLimiter
	mailer  *mailer.Mailer
	wg      sync.WaitGroup
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
	cfg := loadConfig(cfgValidator)
	if !cfgValidator.Valid() {
		var attrs []slog.Attr
		for key, value := range cfgValidator.Errors {
			attrs = append(attrs, slog.Any(key, value))
		}
		jsonlog.LogFatal(logger, "invalid values provided in settings", attrs...)
	}
	logger.Info("settings loaded successfully")

	postgres, err := openPostgres(cfg)
	if err != nil {
		jsonlog.LogFatal(logger, "postgres connection failed", slog.String("error", err.Error()))
	}
	defer postgres.Close() // Graceful shutdown will be implemented later

	logger.Info("postgres connection pool established")

	redisClient, err := openRedis(cfg)
	if err != nil {
		jsonlog.LogFatal(logger, "redis connection failed", slog.String("error", err.Error()))
	}
	defer redisClient.Close() // Graceful shutdown will be implemented later

	logger.Info("redis client connection established")

	mailer, err := mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.from, logger)
	if err != nil {
		jsonlog.LogFatal(logger, "smtp client setup failed", slog.String("error", err.Error()))
	}

	logger.Info("smtp client established")

	metrics.PublishGlobalMetrics(postgres, redisClient, version)

	app := &application{
		config:  cfg,
		logger:  logger,
		models:  data.NewModels(postgres),
		limiter: limiter.New(redisClient, cfg.limiter.burst, cfg.limiter.rps),
		mailer:  mailer,
	}

	if err = app.serve(); err != nil {
		jsonlog.LogFatal(logger, "server failed", slog.String("error", err.Error()))
	}
}

// openPostgres creates a PostgreSQL pool of connections using the config's DSN
// and checks for connectivity by pinging.
// Returns an error if the DSN is wrong, PostgreSQL settings are wrong or
// the app can't ping the PostgreSQL instance.
func openPostgres(cfg config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.postgres.dsn)
	if err != nil {
		return nil, err
	}

	duration, err := time.ParseDuration(cfg.postgres.maxIdleTime)
	if err != nil {
		return nil, err
	}

	poolCfg.MaxConns = int32(cfg.postgres.maxOpenConns)
	poolCfg.MaxConnIdleTime = duration

	postgres, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := postgres.Ping(ctx); err != nil {
		return nil, err
	}

	return postgres, nil
}

// openRedis creates a Redis connection using the config's DSN
// and checks for connectivity by pinging.
// Returns an error if the DSN is wrong or the app can't ping the Redis instance.
func openRedis(cfg config) (*redis.Client, error) {
	opt, err := redis.ParseURL(cfg.redis.dsn)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}
