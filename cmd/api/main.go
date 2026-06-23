package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

type config struct {
	port        int
	environment string
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
	flag.Parse()

	infoLogger := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLogger := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Llongfile)

	if cfg.environment != "development" && cfg.environment != "staging" && cfg.environment != "production" {
		errorLogger.Fatalf("invalid environment provided: %s", cfg.environment)
	}

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
