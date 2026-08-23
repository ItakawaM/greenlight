#!/bin/sh

set -e

if [ -z "$REDIS_PASSWORD" ]; then
  echo "ERROR: REDIS_PASSWORD is not set" >&2
  exit 1
fi

sed "s/\${REDIS_PASSWORD}/$REDIS_PASSWORD/g" /usr/local/etc/redis/redis.conf > /tmp/redis.conf
exec redis-server /tmp/redis.conf