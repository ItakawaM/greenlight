package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// serve is a wrapper around http.Server.ListenAndServe
// that implements graceful shutdown.
func (app *application) serve() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	shutdownErr := make(chan error)
	go func() {
		quit := make(chan os.Signal, 1)

		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		app.logger.Info("shutting down server",
			slog.String("signal", s.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		shutdownErr <- srv.Shutdown(ctx)
	}()

	app.logger.Info("starting server",
		slog.String("addr", srv.Addr),
		slog.String("env", app.config.environment))

	// if ListenAndServe returns a http.ErrServerClosed it means that
	// the graceful shutdown has started
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// we have to check if there were any errors during the shutdown itself
	if err := <-shutdownErr; err != nil {
		return err
	}

	app.logger.Info("stopped server",
		slog.String("addr", srv.Addr),
		slog.String("env", app.config.environment))

	return nil
}
