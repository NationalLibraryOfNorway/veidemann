package recorderproxy

import "github.com/prometheus/client_golang/prometheus"

var activeConnections = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "recorderproxy_active_connections",
	Help: "Current number of accepted recorderproxy connections.",
})

var idleConnectionTimeouts = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "recorderproxy_idle_connection_timeouts_total",
	Help: "Total number of downstream connections closed after an idle request-header timeout.",
})

func init() {
	prometheus.MustRegister(activeConnections, idleConnectionTimeouts)
}
