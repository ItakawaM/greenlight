package config

import (
	"log/slog"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ItakawaM/greenlight/internal/validator"
	"github.com/joho/godotenv"
)

// ServerConfig represents API settings.
type ServerConfig struct {
	Host        string
	Port        int
	Environment string
	Version     string
}

// PostgreSQLConfig represents PostgreSQL settings.
type PostgreSQLConfig struct {
	Host         string
	Port         int
	Username     string
	Password     string
	Database     string
	SSL          bool
	MaxOpenConns int
	MaxIdleTime  time.Duration
}

// RedisConfig represents Redis settings.
type RedisConfig struct {
	Host     string
	Port     int
	Password string
}

// LimiterConfig represents rate-limiting settings.
type LimiterConfig struct {
	RPS     float64
	Burst   int
	Enabled bool
}

// SMTPConfig represents SMTP server/relay settings for mailing.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// CORSConfig represents CORS settings.
type CORSConfig struct {
	TrustedOrigins map[string]struct{}
}

// Config represents API and dependencies settings.
type Config struct {
	Server     ServerConfig
	PostgreSQL PostgreSQLConfig
	Redis      RedisConfig
	Limiter    LimiterConfig
	SMTP       SMTPConfig
	CORS       CORSConfig
}

// LoadConfig loads configuration values from .env.
// Fallback to OS Variables if no .env file is present (example: Docker Compose).
func LoadConfig(v *validator.Validator, logger *slog.Logger) *Config {
	if err := godotenv.Load(); err != nil {
		logger.Warn("no .env file found: fallback to os variables")
	}

	var cfg Config
	cfg.Server.Host = os.Getenv("SERVER_HOST")
	cfg.Server.Port = envInt("SERVER_PORT", v)
	cfg.Server.Environment = os.Getenv("SERVER_ENVIRONMENT")
	cfg.Server.Version = os.Getenv("SERVER_VERSION")

	cfg.PostgreSQL.Host = os.Getenv("POSTGRES_HOST")
	cfg.PostgreSQL.Port = envInt("POSTGRES_PORT", v)
	cfg.PostgreSQL.Username = os.Getenv("POSTGRES_USERNAME")
	cfg.PostgreSQL.Password = os.Getenv("POSTGRES_PASSWORD")
	cfg.PostgreSQL.Database = os.Getenv("POSTGRES_DATABASE")
	cfg.PostgreSQL.SSL = envBool("POSTGRES_SSL", v)
	cfg.PostgreSQL.MaxOpenConns = envInt("POSTGRES_MAX_OPEN_CONNS", v)
	cfg.PostgreSQL.MaxIdleTime = envDuration("POSTGRES_MAX_IDLE_TIME", v)

	cfg.Redis.Host = os.Getenv("REDIS_HOST")
	cfg.Redis.Port = envInt("REDIS_PORT", v)
	cfg.Redis.Password = os.Getenv("REDIS_PASSWORD")

	cfg.Limiter.RPS = envFloat("LIMITER_RPS", v)
	cfg.Limiter.Burst = envInt("LIMITER_BURST", v)
	cfg.Limiter.Enabled = envBool("LIMITER_ENABLED", v)

	cfg.SMTP.Host = os.Getenv("SMTP_HOST")
	cfg.SMTP.Port = envInt("SMTP_PORT", v)
	cfg.SMTP.Username = os.Getenv("SMTP_USERNAME")
	cfg.SMTP.Password = os.Getenv("SMTP_PASSWORD")
	cfg.SMTP.From = os.Getenv("SMTP_FROM")

	cfg.CORS.TrustedOrigins = envSet("CORS_TRUSTED_ORIGINS", v)

	v.Check(validator.NotBlank(cfg.Server.Host), "SERVER_HOST", "must be not blank")
	v.Check(isValidPort(cfg.Server.Port), "SERVER_PORT", "must be between 0 and 65535")
	v.Check(validator.NotBlank(cfg.Server.Version), "SERVER_VERSION", "must be not blank")
	v.Check(validator.PermittedValue(cfg.Server.Environment, "development", "staging", "production"), "SERVER_ENVIRONMENT", "must be one of (development, staging, production)")

	v.Check(validator.NotBlank(cfg.PostgreSQL.Host), "POSTGRES_HOST", "must be not blank")
	v.Check(isValidPort(cfg.PostgreSQL.Port), "POSTGRES_PORT", "must be between 0 and 65535")
	v.Check(validator.NotBlank(cfg.PostgreSQL.Username), "POSTGRES_USERNAME", "must be not blank")
	v.Check(validator.NotBlank(cfg.PostgreSQL.Password), "POSTGRES_PASSWORD", "must be not blank")
	v.Check(validator.NotBlank(cfg.PostgreSQL.Database), "POSTGRES_DATABASE", "must be not blank")
	v.Check(cfg.PostgreSQL.MaxOpenConns > 0 && cfg.PostgreSQL.MaxOpenConns <= 1000, "POSTGRES_MAX_OPEN_CONNS", "must be between 0 and 1000")
	v.Check(cfg.PostgreSQL.MaxIdleTime > 0, "POSTGRES_MAX_IDLE_TIME", "must be > 0")

	v.Check(validator.NotBlank(cfg.Redis.Host), "REDIS_HOST", "must be not blank")
	v.Check(isValidPort(cfg.Redis.Port), "REDIS_PORT", "must be between 0 and 65535")
	v.Check(validator.NotBlank(cfg.Redis.Password), "REDIS_PASSWORD", "must be not blank")

	v.Check(cfg.Limiter.RPS > 0, "LIMITER_RPS", "must be > 0")
	v.Check(cfg.Limiter.Burst > 0, "LIMITER_BURST", "must be > 0")

	v.Check(validator.NotBlank(cfg.SMTP.Host), "SMTP_HOST", "must be not blank")
	v.Check(validator.PermittedValue(cfg.SMTP.Port, 25, 465, 587, 2525), "SMTP_PORT", "must be one of (25, 465, 587, 2525)")
	v.Check(validator.NotBlank(cfg.SMTP.Username), "SMTP_USERNAME", "must be not blank")
	v.Check(validator.NotBlank(cfg.SMTP.Password), "SMTP_PASSWORD", "must be not blank")
	if _, err := mail.ParseAddress(cfg.SMTP.From); err != nil {
		v.AddError("SMTP_FROM", "must be a valid RFC 5322 named address")
	}

	return &cfg
}

// envInt loads an integer value from env and validates it.
func envInt(key string, v *validator.Validator) int {
	val := os.Getenv(key)
	if val == "" {
		v.AddError(key, "must be set")
		return 0
	}

	parsed, err := strconv.Atoi(val)
	if err != nil {
		v.AddError(key, "must be an integer")
	}

	return parsed
}

// envFloat loads a float value from env and validates it.
func envFloat(key string, v *validator.Validator) float64 {
	val := os.Getenv(key)
	if val == "" {
		v.AddError(key, "must be set")
		return 0.0
	}

	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		v.AddError(key, "must be a float")
	}

	return parsed
}

// envDuration loads a time.Duration value from env and validates it.
func envDuration(key string, v *validator.Validator) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		v.AddError(key, "must be set")
		return time.Duration(0)
	}

	parsed, err := time.ParseDuration(val)
	if err != nil {
		v.AddError(key, "must be a valid duration")
	}

	return parsed
}

// envSet loads a map[string]struct{} value from env and validates it.
func envSet(key string, v *validator.Validator) map[string]struct{} {
	val := os.Getenv(key)
	if val == "" {
		v.AddError(key, "must be set")
		return nil
	}

	parsed := make(map[string]struct{})
	for entry := range strings.FieldsSeq(val) {
		parsed[entry] = struct{}{}
	}

	return parsed
}

// envBool loads a bool value from env and validates it.
func envBool(key string, v *validator.Validator) bool {
	val := os.Getenv(key)
	if val == "" {
		v.AddError(key, "must be set")
		return false
	}

	parsed, err := strconv.ParseBool(val)
	if err != nil {
		v.AddError(key, "must be a bool")
	}

	return parsed
}

// isValidPort checks whether val is a valid port.
func isValidPort(val int) bool {
	return val > 0 && val <= 65535
}
