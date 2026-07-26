package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/ItakawaM/greenlight/internal/data"
	"github.com/ItakawaM/greenlight/internal/jsonlog"
	"github.com/ItakawaM/greenlight/internal/limiter"
	"github.com/ItakawaM/greenlight/internal/mailer"
	"github.com/ItakawaM/greenlight/internal/validator"
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
	smtp struct {
		host     string
		port     int
		username string
		password string
		from     string
	}
}

type application struct {
	config  config
	logger  *slog.Logger
	models  *data.Models
	limiter *limiter.TokenBucketLimiter
	mailer  *mailer.Mailer
}

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

func loadConfig(v *validator.Validator) config {
	var cfg config
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

	// SMTP Settings
	flag.StringVar(&cfg.smtp.host, "smtp-host", os.Getenv("SMTP_HOST"), "SMTP host")
	port, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		v.AddError("smtp-port", "must be an integer")
	}
	flag.IntVar(&cfg.smtp.port, "smtp-port", port, "SMTP port (25|465|587|2525)")
	flag.StringVar(&cfg.smtp.username, "smtp-username", os.Getenv("SMTP_USERNAME"), "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", os.Getenv("SMTP_PASSWORD"), "SMTP password")
	flag.StringVar(&cfg.smtp.from, "smtp-from", os.Getenv("SMTP_FROM"), "SMTP from")

	flag.Parse()

	v.Check(cfg.port > 0 && cfg.port <= 65535, "port", "must be between 0 and 65535")
	v.Check(validator.PermittedValue(cfg.environment, "development", "staging", "production"), "env", "must be one of (development|staging|production)")

	v.Check(validator.NotBlank(cfg.postgres.dsn), "postgres-dsn", "must be not blank")
	v.Check(cfg.postgres.maxOpenConns > 0 && cfg.postgres.maxOpenConns <= 1000, "postgres-max-open-conns", "must be between 0 and 1000")

	_, err = time.ParseDuration(cfg.postgres.maxIdleTime)
	if err != nil {
		v.AddError("postgres-max-idle-time", "must be a valid time")
	}

	v.Check(validator.NotBlank(cfg.redis.dsn), "redis-dsn", "must be not blank")

	v.Check(cfg.limiter.rps > 0, "limiter-rps", "must be greater than 0")
	v.Check(cfg.limiter.burst > 0, "limiter-burst", "must be greater than 0")

	v.Check(validator.NotBlank(cfg.smtp.host), "smtp-host", "must be not blank")
	v.Check(validator.PermittedValue(cfg.smtp.port, 25, 465, 587, 2525), "smtp-port", "must be one of (25|465|587|2525)")
	v.Check(validator.NotBlank(cfg.smtp.username), "smtp-username", "must be not blank")
	v.Check(validator.NotBlank(cfg.smtp.password), "smtp-password", "must be not blank")
	v.Check(validator.NotBlank(cfg.smtp.from), "smtp-from", "must be not blank")

	return cfg
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
