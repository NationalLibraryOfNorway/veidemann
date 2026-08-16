package recorderproxy

import "github.com/prometheus/client_golang/prometheus"

var activeConnections = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "recorderproxy_active_connections",
	Help: "Current number of accepted recorderproxy connections.",
})

func init() {
	prometheus.MustRegister(activeConnections)
}
