package metrics

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	goredis "github.com/redis/go-redis/v9"
)

// redisCollector implements prometheus.Collector, collects and parses redis INFO metrics.
type redisCollector struct {
	client *goredis.Client

	// up is a gauge that represents whether or not redis is up.
	up *prometheus.Desc

	// Clients

	// connectedClients is a gauge that represents the number of active clients.
	connectedClients *prometheus.Desc
	// blockedClients is a gauge that represents the number of awaiting clients.
	blockedClients *prometheus.Desc

	// Memory

	// usedMemory is a gauge that represents the memory used in bytes.
	usedMemory *prometheus.Desc
	// maxMemory is a gauge that represents maximum bytes of memory.
	maxMemory *prometheus.Desc
	// memFragmentationRatio is a gauge that represents the ratio between physical memory
	// allocated by the OS and the memory actually used by Redis.
	memFragmentationRatio *prometheus.Desc

	// Persistence

	// rdbLastBgsaveStatus is a gauge that represents the last save status of BGSAVE.
	rdbLastBgsaveStatus *prometheus.Desc
	// aofLastWriteStatus is a gauge that represents the last write status of AOF.
	aofLastWriteStatus *prometheus.Desc
	// rdbChangesSinceLastSave is a gauge that represents the number of operations
	// that changed the dataset since the last successful RDB snapshot or save.
	rdbChangesSinceLastSave *prometheus.Desc

	// Stats

	// keySpaceHits is a counter that represents the total number of successful lookup of keys in the main dictionary.
	keySpaceHits *prometheus.Desc
	// keySpaceMisses is a counter that represents the total number of failed lookup of keys in the main dictionary.
	keySpaceMisses *prometheus.Desc
	// expiredKeys is a counter that represents the total number of key expiration events.
	expiredKeys *prometheus.Desc
	// evictedKeys is a counter that represents the total number of evicted keys due to maxmemory limit.
	evictedKeys *prometheus.Desc
	// totalConnectionsReceived is a counter that represents the total number of connections accepted by the server.
	totalConnectionsReceived *prometheus.Desc
	// rejectedConnections is a counter that represents the total number of connections rejected because of maxclients limit.
	rejectedConnections *prometheus.Desc
	// totalErrorReplies is a counter that represents the total number of issued error replies,
	totalErrorReplies *prometheus.Desc
}

// newRedisCollector initializes a new redis metric collector.
func newRedisCollector(client *goredis.Client) *redisCollector {
	return &redisCollector{
		client: client,
		up:     prometheus.NewDesc("redis_up", "Whether or not redis is up", nil, nil),

		connectedClients: prometheus.NewDesc("redis_clients_connected_clients", "Number of active clients", nil, nil),
		blockedClients:   prometheus.NewDesc("redis_clients_blocked_clients", "Number of awaiting clients", nil, nil),

		usedMemory:            prometheus.NewDesc("redis_memory_used_memory_bytes", "Memory used in bytes", nil, nil),
		maxMemory:             prometheus.NewDesc("redis_memory_max_memory_bytes", "Maximum memory in bytes", nil, nil),
		memFragmentationRatio: prometheus.NewDesc("redis_memory_fragmentation_ratio", "Memory Fragmentation ratio", nil, nil),

		rdbLastBgsaveStatus:     prometheus.NewDesc("redis_persistence_rdb_last_bgsave_status", "Last BGSAVE status", nil, nil),
		aofLastWriteStatus:      prometheus.NewDesc("redis_persistence_aof_last_write_status", "Last AOF write status", nil, nil),
		rdbChangesSinceLastSave: prometheus.NewDesc("redis_persistence_rdb_changes_since_last_save", "Number of operations since last save", nil, nil),

		keySpaceHits:             prometheus.NewDesc("redis_stats_keyspace_hits_total", "Total number of key space hits", nil, nil),
		keySpaceMisses:           prometheus.NewDesc("redis_stats_keyspace_misses_total", "Total number of key space misses", nil, nil),
		expiredKeys:              prometheus.NewDesc("redis_stats_expired_keys_total", "Total number of expired keys", nil, nil),
		evictedKeys:              prometheus.NewDesc("redis_stats_evicted_keys_total", "Total number of evicted keys", nil, nil),
		totalConnectionsReceived: prometheus.NewDesc("redis_stats_connections_received_total", "Total number of connections received", nil, nil),
		rejectedConnections:      prometheus.NewDesc("redis_stats_rejected_connections_total", "Total number of rejected connections", nil, nil),
		totalErrorReplies:        prometheus.NewDesc("redis_stats_error_replies_total", "Total number of error replies", nil, nil),
	}
}

// Describe implements prometheus.Collector.
func (rc *redisCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- rc.up

	ch <- rc.connectedClients
	ch <- rc.blockedClients

	ch <- rc.usedMemory
	ch <- rc.maxMemory
	ch <- rc.memFragmentationRatio

	ch <- rc.rdbLastBgsaveStatus
	ch <- rc.aofLastWriteStatus
	ch <- rc.rdbChangesSinceLastSave

	ch <- rc.keySpaceHits
	ch <- rc.keySpaceMisses
	ch <- rc.expiredKeys
	ch <- rc.evictedKeys
	ch <- rc.totalConnectionsReceived
	ch <- rc.rejectedConnections
	ch <- rc.totalErrorReplies
}

// Collect implements prometheus.Collector.
func (rc *redisCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	info, err := rc.client.Info(ctx).Result()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(rc.up, prometheus.GaugeValue, 0)
		return
	}
	ch <- prometheus.MustNewConstMetric(rc.up, prometheus.GaugeValue, 1)

	fields := rc.parseInfo(info)

	rc.collectGauge(ch, fields, "connected_clients", rc.connectedClients)
	rc.collectGauge(ch, fields, "blocked_clients", rc.blockedClients)

	rc.collectGauge(ch, fields, "used_memory", rc.usedMemory)
	rc.collectGauge(ch, fields, "maxmemory", rc.maxMemory)
	rc.collectGauge(ch, fields, "mem_fragmentation_ratio", rc.memFragmentationRatio)

	rc.collectGaugeBoolean(ch, fields, "rdb_last_bgsave_status", rc.rdbLastBgsaveStatus)
	rc.collectGaugeBoolean(ch, fields, "aof_last_write_status", rc.aofLastWriteStatus)
	rc.collectGauge(ch, fields, "rdb_changes_since_last_save", rc.rdbChangesSinceLastSave)

	rc.collectCounter(ch, fields, "keyspace_hits", rc.keySpaceHits)
	rc.collectCounter(ch, fields, "keyspace_misses", rc.keySpaceMisses)
	rc.collectCounter(ch, fields, "expired_keys", rc.expiredKeys)
	rc.collectCounter(ch, fields, "evicted_keys", rc.evictedKeys)
	rc.collectCounter(ch, fields, "total_connections_received", rc.totalConnectionsReceived)
	rc.collectCounter(ch, fields, "rejected_connections", rc.rejectedConnections)
	rc.collectCounter(ch, fields, "total_error_replies", rc.totalErrorReplies)
}

// collectGauge collects a Gauge metric from the provided field.
func (rc *redisCollector) collectGauge(ch chan<- prometheus.Metric, fields map[string]string, key string, desc *prometheus.Desc) {
	if value, ok := fields[key]; ok {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, f)
		}
	}
}

// collectGaugeBoolean collects a Gauge 1/0 ok/err metric from the provided field.
func (rc *redisCollector) collectGaugeBoolean(ch chan<- prometheus.Metric, fields map[string]string, key string, desc *prometheus.Desc) {
	if value, ok := fields[key]; ok {
		if value == "ok" {
			ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, 1)
		} else {
			ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, 0)
		}
	}
}

// collectCounter collects a Counter metric from the provided field.
func (rc *redisCollector) collectCounter(ch chan<- prometheus.Metric, fields map[string]string, key string, desc *prometheus.Desc) {
	if value, ok := fields[key]; ok {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, f)
		}
	}
}

// parseInfo parses redis INFO output into a map.
func (rc *redisCollector) parseInfo(info string) map[string]string {
	out := make(map[string]string)
	for line := range strings.SplitSeq(info, "\r\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		row := strings.SplitN(line, ":", 2)
		if len(row) != 2 {
			continue
		}
		out[row[0]] = row[1]
	}

	return out
}
