#!/bin/sh

set -e

if [ -z "$REDIS_PASSWORD" ]; then
  echo "ERROR: REDIS_PASSWORD is not set" >&2
  exit 1
fi

REDIS_CONF="$(mktemp /usr/local/etc/redis/redis.XXXXXX.conf)"
ESCAPED_PASSWORD=$(printf '%s' "$REDIS_PASSWORD" | sed 's/[\/&\\]/\\&/g')
sed "s/\${REDIS_PASSWORD}/$ESCAPED_PASSWORD/g" /usr/local/etc/redis/redis.conf.template > "$REDIS_CONF"
exec redis-server "$REDIS_CONF"