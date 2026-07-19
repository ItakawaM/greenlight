package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/ItakawaM/greenlight/internal/data"
	"github.com/ItakawaM/greenlight/internal/jsonlog"
	"github.com/ItakawaM/greenlight/internal/limiter"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const version = "1.0.0"

type config struct {
	port        int
	environment string
	postgres    struct {
		dsn          string
		maxOpenConns int
		maxIdleTime  string
	}
	redis struct {
		dsn string
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
}

type application struct {
	config  config
	logger  *slog.Logger
	models  *data.Models
	limiter *limiter.TokenBucketLimiter
}

func main() {
	var cfg config
	loadConfig(&cfg) // TODO: Define better config parsing and validation

	logger := jsonlog.New(os.Stdout, slog.LevelInfo)

	postgres, err := openPostgres(cfg)
	if err != nil {
		logger.LogAttrs(context.Background(), jsonlog.LevelFatal, "postgres connection failed",
			slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer postgres.Close() // Graceful shutdown will be implemented later

	logger.LogAttrs(context.Background(), slog.LevelInfo, "postgres connection pool established")

	redisClient, err := openRedis(cfg)
	if err != nil {
		logger.LogAttrs(context.Background(), jsonlog.LevelFatal, "redis connection failed",
			slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer redisClient.Close() // Graceful shutdown will be implemented later

	logger.LogAttrs(context.Background(), slog.LevelInfo, "redis connection client established")

	app := &application{
		config:  cfg,
		logger:  logger,
		models:  data.NewModels(postgres),
		limiter: limiter.New(redisClient, cfg.limiter.burst, cfg.limiter.rps),
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	logger.LogAttrs(context.Background(), slog.LevelInfo, "starting server",
		slog.String("addr", srv.Addr),
		slog.String("env", cfg.environment))

	if err = srv.ListenAndServe(); err != nil {
		logger.LogAttrs(context.Background(), jsonlog.LevelFatal, "server failed",
			slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func loadConfig(cfg *config) {
	// Server Settings
	flag.IntVar(&cfg.port, "port", 4000, "API Server Port")
	flag.StringVar(&cfg.environment, "env", "development", "Environment (development|staging|production)")

	// Postgres Settings
	flag.StringVar(&cfg.postgres.dsn, "postgres-dsn", os.Getenv("POSTGRES_URL"), "PostgreSQL DSN")
	flag.IntVar(&cfg.postgres.maxOpenConns, "postgres-max-open-conns", 25, "PostgreSQL maximum open connections")
	flag.StringVar(&cfg.postgres.maxIdleTime, "postgres-max-idle-time", "15m", "PostgreSQL maximum connection idle time")

	// Redis Settings
	flag.StringVar(&cfg.redis.dsn, "redis-dsn", os.Getenv("REDIS_URL"), "Redis DSN")

	// Limiter Settings
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate Limiter maximum requests/second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate Limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.Parse()
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
