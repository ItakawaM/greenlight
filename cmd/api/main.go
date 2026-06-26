package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

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
	config      config
	infoLogger  *log.Logger
	errorLogger *log.Logger
}

func main() {
	var cfg config
	flag.IntVar(&cfg.port, "port", 4000, "API Server Port")
	flag.StringVar(&cfg.environment, "env", "development", "Environment (development|staging|production)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("DATABASE_URL"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostreSQL Max Open Connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostreSQL Max Connection Idle Time")
	flag.Parse()

	infoLogger := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLogger := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Llongfile)

	if cfg.environment != "development" && cfg.environment != "staging" && cfg.environment != "production" {
		errorLogger.Fatalf("invalid environment provided: %s", cfg.environment)
	}

	db, err := openDB(cfg)
	if err != nil {
		errorLogger.Fatal(err)
	}
	defer db.Close()

	infoLogger.Print("database connection pool established")

	app := &application{
		config:      cfg,
		infoLogger:  infoLogger,
		errorLogger: errorLogger,
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	infoLogger.Printf("starting %s server on %s", cfg.environment, srv.Addr)
	errorLogger.Fatal(srv.ListenAndServe())
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
