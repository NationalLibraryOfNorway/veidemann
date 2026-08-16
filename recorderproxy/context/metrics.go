package context

import "github.com/prometheus/client_golang/prometheus"

var contentWriterTerminalTimeouts = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "recorderproxy_contentwriter_terminal_timeouts_total",
	Help: "Total ContentWriter terminal operations canceled after their deadline.",
})

func init() {
	prometheus.MustRegister(
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "recorderproxy_open_sessions",
			Help: "Current records whose terminal operations have not all been attempted.",
		}, func() float64 { return float64(OpenSessions()) }),
		contentWriterTerminalTimeouts,
	)
}
