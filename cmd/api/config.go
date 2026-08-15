package main

import (
	"flag"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ItakawaM/greenlight/internal/validator"
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
	cors struct {
		trustedOrigins map[string]struct{}
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
	smtpPort := os.Getenv("SMTP_PORT")
	flag.StringVar(&smtpPort, "smtp-port", smtpPort, "SMTP port (25|465|587|2525)")
	flag.StringVar(&cfg.smtp.host, "smtp-host", os.Getenv("SMTP_HOST"), "SMTP host")
	flag.StringVar(&cfg.smtp.username, "smtp-username", os.Getenv("SMTP_USERNAME"), "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", os.Getenv("SMTP_PASSWORD"), "SMTP password")
	flag.StringVar(&cfg.smtp.from, "smtp-from", os.Getenv("SMTP_FROM"), "SMTP from")

	// CORS
	flag.Func("cors-trusted-origins", "Trusted CORS Origins (space separated)", func(s string) error {
		cfg.cors.trustedOrigins = make(map[string]struct{})
		for origin := range strings.FieldsSeq(s) {
			cfg.cors.trustedOrigins[origin] = struct{}{}
		}
		return nil
	})

	flag.Parse()

	if validator.NotBlank(smtpPort) {
		var err error
		cfg.smtp.port, err = strconv.Atoi(smtpPort)
		if err != nil {
			v.AddError("smtp-port", "must be an integer")
		}
	}

	v.Check(cfg.port > 0 && cfg.port <= 65535, "port", "must be between 0 and 65535")
	v.Check(validator.PermittedValue(cfg.environment, "development", "staging", "production"), "env", "must be one of (development|staging|production)")

	v.Check(validator.NotBlank(cfg.postgres.dsn), "postgres-dsn", "must be not blank")
	v.Check(cfg.postgres.maxOpenConns > 0 && cfg.postgres.maxOpenConns <= 1000, "postgres-max-open-conns", "must be between 0 and 1000")

	if _, err := time.ParseDuration(cfg.postgres.maxIdleTime); err != nil {
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

	if _, err := mail.ParseAddress(cfg.smtp.from); err != nil {
		v.AddError("smtp-from", "must be a valid RFC 5322 named address")
	}

	return cfg
}
