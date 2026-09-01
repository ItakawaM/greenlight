#!/bin/sh
set -e

if [ -z "$METRICS_PORT" ]; then
  echo "ERROR: METRICS_PORT is not set" >&2
  exit 1
fi

sed "s/\${METRICS_PORT}/$METRICS_PORT/g" /etc/prometheus/prometheus.yml.template > /etc/prometheus/prometheus.yml

exec /bin/prometheus \
  --config.file=/etc/prometheus/prometheus.yml \
  --storage.tsdb.path=/prometheus \
  --storage.tsdb.retention.time=15d