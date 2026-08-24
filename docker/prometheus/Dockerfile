FROM prom/prometheus:v3.14.0

COPY prometheus.yml /tmp/prometheus.yml
COPY entrypoint.sh /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]