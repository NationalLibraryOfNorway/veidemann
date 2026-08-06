package recentlog

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
)

const metricsNamespace = "veidemann"

type Metrics struct {
	entries      *prometheus.GaugeVec
	evicted      *prometheus.CounterVec
	writeFailure *prometheus.CounterVec
	payloadBytes *prometheus.HistogramVec
}

func NewMetrics(registerer prometheus.Registerer, databasePath string) *Metrics {
	metrics := &Metrics{
		entries: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: "recent_logs",
			Name:      "entries",
			Help:      "Number of crawl and page logs retained in the recent SQLite read store.",
		}, []string{"type"}),
		evicted: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: "recent_logs",
			Name:      "evicted_total",
			Help:      "Number of logs evicted from the recent SQLite read store by retention.",
		}, []string{"type"}),
		writeFailure: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: "recent_logs",
			Name:      "write_failures_total",
			Help:      "Number of failed writes to the recent SQLite read store.",
		}, []string{"type"}),
		payloadBytes: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Subsystem: "recent_logs",
			Name:      "payload_bytes",
			Help:      "Serialized protobuf size of logs written to the recent SQLite read store.",
			Buckets:   prometheus.ExponentialBuckets(256, 2, 17),
		}, []string{"type"}),
	}
	registerer.MustRegister(
		metrics.entries,
		metrics.evicted,
		metrics.writeFailure,
		metrics.payloadBytes,
		newDatabaseFileCollector(databasePath),
	)
	for _, logType := range []string{CrawlLogType, PageLogType} {
		metrics.entries.WithLabelValues(logType)
		metrics.evicted.WithLabelValues(logType)
		metrics.writeFailure.WithLabelValues(logType)
		metrics.payloadBytes.WithLabelValues(logType)
	}
	return metrics
}

func (m *Metrics) setEntries(logType string, count int64) {
	if m != nil {
		m.entries.WithLabelValues(logType).Set(float64(count))
	}
}

func (m *Metrics) addEvicted(logType string, count int64) {
	if m != nil && count > 0 {
		m.evicted.WithLabelValues(logType).Add(float64(count))
	}
}

func (m *Metrics) recordWriteFailure(logType string) {
	if m != nil {
		m.writeFailure.WithLabelValues(logType).Inc()
	}
}

func (m *Metrics) observePayload(logType string, size int) {
	if m != nil {
		m.payloadBytes.WithLabelValues(logType).Observe(float64(size))
	}
}

type databaseFileCollector struct {
	databasePath string
	description  *prometheus.Desc
}

func newDatabaseFileCollector(databasePath string) prometheus.Collector {
	return &databaseFileCollector{
		databasePath: databasePath,
		description: prometheus.NewDesc(
			prometheus.BuildFQName(metricsNamespace, "recent_logs", "database_file_bytes"),
			"Size of the SQLite main database and sidecar files.",
			[]string{"file"},
			nil,
		),
	}
}

func (c *databaseFileCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.description
}

func (c *databaseFileCollector) Collect(ch chan<- prometheus.Metric) {
	files := []struct {
		label string
		path  string
	}{
		{label: "main", path: c.databasePath},
		{label: "wal", path: c.databasePath + "-wal"},
		{label: "shm", path: c.databasePath + "-shm"},
	}
	for _, file := range files {
		var size float64
		if info, err := os.Stat(file.path); err == nil {
			size = float64(info.Size())
		}
		ch <- prometheus.MustNewConstMetric(c.description, prometheus.GaugeValue, size, file.label)
	}
}
