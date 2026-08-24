FROM prom/prometheus:v3.14.0

COPY prometheus.yml /etc/prometheus/prometheus.yml.template
COPY entrypoint.sh /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]