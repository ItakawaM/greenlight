package main

import (
	"expvar"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type pgxPoolStats struct {
	AcquireCount            int64 `json:"acquire_count"`
	AcquireDuration         int64 `json:"acquire_duration_ns"`
	AcquiredConns           int32 `json:"acquired_conns"`
	CanceledAcquireCount    int64 `json:"canceled_acquire_count"`
	ConstructingConns       int32 `json:"constructing_conns"`
	EmptyAcquireCount       int64 `json:"empty_acquire_count"`
	IdleConns               int32 `json:"idle_conns"`
	MaxConns                int32 `json:"max_conns"`
	MaxIdleDestroyCount     int64 `json:"max_idle_destroy_count"`
	MaxLifetimeDestroyCount int64 `json:"max_lifetime_destroy_count"`
	NewConnsCount           int64 `json:"new_conns_count"`
	TotalConns              int32 `json:"total_conns"`
}

func publishGlobalMetrics(postgres *pgxpool.Pool, redis *redis.Client) {
	expvar.NewString("version").Set(version)

	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))

	expvar.Publish("postgres", expvar.Func(func() any {
		s := postgres.Stat()
		return pgxPoolStats{
			AcquireCount:            s.AcquireCount(),
			AcquireDuration:         int64(s.AcquireDuration()),
			AcquiredConns:           s.AcquiredConns(),
			CanceledAcquireCount:    s.CanceledAcquireCount(),
			ConstructingConns:       s.ConstructingConns(),
			EmptyAcquireCount:       s.EmptyAcquireCount(),
			IdleConns:               s.IdleConns(),
			MaxConns:                s.MaxConns(),
			MaxIdleDestroyCount:     s.MaxIdleDestroyCount(),
			MaxLifetimeDestroyCount: s.MaxLifetimeDestroyCount(),
			NewConnsCount:           s.NewConnsCount(),
			TotalConns:              s.TotalConns(),
		}
	}))

	expvar.Publish("redis", expvar.Func(func() any {
		return redis.PoolStats()
	}))

	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))
}
