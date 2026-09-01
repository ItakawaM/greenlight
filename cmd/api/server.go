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

	"github.com/ItakawaM/greenlight/internal/logging"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// serve is a wrapper around http.Server.ListenAndServe
// that implements graceful shutdown.
func (app *application) serve() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", app.config.Server.Host, app.config.Server.Port),
		Handler:      app.routes(),
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// optional metrics server.
	var metricsSrv *http.Server
	if app.config.Metrics.Enabled {
		metricsMux := http.NewServeMux()
		metricsMux.Handle("/metrics", promhttp.HandlerFor(
			app.metrics.Registry,
			promhttp.HandlerOpts{
				ErrorLog:      logging.NewSlogProm(app.logger),
				ErrorHandling: promhttp.HTTPErrorOnError,
			},
		))

		metricsSrv = &http.Server{
			Addr:         fmt.Sprintf("%s:%d", app.config.Server.Host, app.config.Metrics.Port),
			Handler:      metricsMux,
			ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
			IdleTimeout:  time.Minute,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
		}
	}

	shutdownErr := make(chan error)
	quit := make(chan os.Signal, 1)
	done := make(chan struct{})

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		defer signal.Stop(quit)

		select {
		case s := <-quit:
			app.logger.Info("shutting down server",
				slog.String("signal", s.String()))

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := srv.Shutdown(ctx); err != nil {
				shutdownErr <- err
				return
			}

			if metricsSrv != nil {
				if err := metricsSrv.Shutdown(ctx); err != nil {
					shutdownErr <- err
					return
				}
			}

			app.logger.Info("completing background tasks")

			// wait for all background goroutines
			app.wg.Wait()
			shutdownErr <- nil

		case <-done:
			// server already stopped on its own
			// nothing to shut down, exit without sending on shutdownErr
		}
	}()

	if metricsSrv != nil {
		go func() {
			app.logger.Info("starting metrics server",
				slog.String("addr", metricsSrv.Addr))

			// the actual healthcheck of metrics is managed by docker-compose
			app.metricsHealthy.Store(true)
			err := metricsSrv.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				app.logger.Error("metrics server error", slog.Any("error", err))
				app.metricsHealthy.Store(false)
			}
		}()
	}

	app.logger.Info("starting server",
		slog.String("addr", srv.Addr),
		slog.String("env", app.config.Server.Environment))

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

	app.logger.Info("stopped server",
		slog.String("addr", srv.Addr),
		slog.String("env", app.config.Server.Environment))

	return nil
}
