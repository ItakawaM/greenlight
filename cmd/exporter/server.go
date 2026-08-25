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

// serve implements graceful shutdown for the exporter.
func (e *exporter) serve() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", e.config.Metrics.Port),
		Handler:      e.routes(),
		ErrorLog:     slog.NewLogLogger(e.logger.Handler(), slog.LevelError),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	shutdownErr := make(chan error)
	quit := make(chan os.Signal, 1)
	done := make(chan struct{})

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		defer signal.Stop(quit)

		select {
		case s := <-quit:
			e.logger.Info("shutting down exporter",
				slog.String("signal", s.String()))

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := srv.Shutdown(ctx); err != nil {
				shutdownErr <- err
				return
			}

			shutdownErr <- nil

		case <-done:
			// server already stopped on its own
			// nothing to shut down, exit without sending on shutdownErr
		}
	}()

	e.logger.Info("starting exporter",
		slog.String("addr", srv.Addr),
	)

	// we have to check if ListenAndServe itself was successful
	err := srv.ListenAndServe()
	close(done)

	// if ListenAndServe returns a http.ErrServerClosed it means that
	// the graceful shutdown has started
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// we have to check if there were any errors during the shutdown itself
	if err := <-shutdownErr; err != nil {
		return err
	}

	e.logger.Info("stopped exporter",
		slog.String("addr", srv.Addr),
	)

	return nil
}
