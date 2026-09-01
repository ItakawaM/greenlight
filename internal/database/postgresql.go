package database

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgreSQL creates a PostgreSQL pool of connections by building a dsn
// and checks for connectivity by pinging.
// Returns an error if the DSN is wrong, PostgreSQL settings are wrong or
// the app can't ping the PostgreSQL instance.
func NewPostgreSQL(host string, port int, username, password, database string, maxOpenConns int, maxIdleDuration time.Duration, ssl bool) (*pgxpool.Pool, error) {
	sslmode := "disable"
	if ssl {
		sslmode = "verify-full"
	}

	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(username, password),
		Host:   fmt.Sprintf("%s:%d", host, port),
		Path:   database,
	}

	// PostgreSQL exists inside the docker-compose with no connections made outside of it
	// for this setup having sslmode=disable is fine
	// if we ever want to add an external service that communicates with our PostgreSQL
	// we can just flip sslmode=verify-full and provide trusted CA certificates
	q := u.Query()
	q.Set("sslmode", sslmode)
	u.RawQuery = q.Encode()

	dsn := u.String()

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pgxpool config: %w", err)
	}

	if maxOpenConns > math.MaxInt32 {
		return nil, fmt.Errorf("max open connections exceeded: %d", maxOpenConns)
	}

	poolCfg.MaxConns = int32(maxOpenConns)
	poolCfg.MaxConnIdleTime = maxIdleDuration

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pgxpool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgresql: %w", err)
	}

	return pool, nil
}
