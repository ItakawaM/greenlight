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
	"github.com/jackc/pgx/v5/pgxpool"
)

const version = "1.0.0"

type config struct {
	port        int
	environment string
	db          struct {
		dsn          string
		maxOpenConns int
		maxIdleTime  string
	}
}

type application struct {
	config config
	logger *slog.Logger
	models *data.Models
}

func main() {
	var cfg config
	flag.IntVar(&cfg.port, "port", 4000, "API Server Port")
	flag.StringVar(&cfg.environment, "env", "development", "Environment (development|staging|production)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("DATABASE_URL"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL Max Open Connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL Max Connection Idle Time")
	flag.Parse()

	logger := jsonlog.New(os.Stdout, slog.LevelInfo)
	if cfg.environment != "development" && cfg.environment != "staging" && cfg.environment != "production" {
		logger.LogAttrs(context.Background(), jsonlog.LevelFatal, "invalid environment provided",
			slog.String("environment", cfg.environment))
		os.Exit(1)
	}

	db, err := openDB(cfg)
	if err != nil {
		logger.LogAttrs(context.Background(), jsonlog.LevelFatal, "db connection failed",
			slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close() // Graceful shutdown will be implemented later

	logger.LogAttrs(context.Background(), slog.LevelInfo, "database connection pool established")

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
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

func openDB(cfg config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}

	poolCfg.MaxConns = int32(cfg.db.maxOpenConns)
	poolCfg.MaxConnIdleTime = duration

	db, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		return nil, err
	}

	return db, nil
}
