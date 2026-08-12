package recentforward

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	logclient "github.com/NationalLibraryOfNorway/veidemann/log-service/pkg/logservice"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

const (
	ForwardedMetadataKey = "veidemann-log-forwarded"

	CrawlLogType = "crawl"
	PageLogType  = "page"

	OutcomeSuccess      = "success"
	OutcomeFailure      = "failure"
	OutcomeQueueFull    = "queue_full_drop"
	OutcomeShutdownDrop = "shutdown_drop"
)

type LogWriter interface {
	WriteCrawlLog(ctx context.Context, crawlLog *logV1.CrawlLog) error
	WritePageLog(ctx context.Context, pageLog *logV1.PageLog) error
}

type Config struct {
	QueueSize int
	Workers   int
	Timeout   time.Duration
	Metrics   *Metrics
}

type item struct {
	logType  string
	crawlLog *logV1.CrawlLog
	pageLog  *logV1.PageLog
}

type Forwarder struct {
	writer  LogWriter
	timeout time.Duration
	metrics *Metrics

	ctx    context.Context
	cancel context.CancelFunc
	queue  chan item
	wg     sync.WaitGroup

	mu     sync.Mutex
	closed bool
}

func New(writer LogWriter, cfg Config) (*Forwarder, error) {
	if writer == nil {
		return nil, errors.New("recent log writer must not be nil")
	}
	if cfg.QueueSize <= 0 {
		return nil, errors.New("recent forward queue size must be > 0")
	}
	if cfg.Workers <= 0 {
		return nil, errors.New("recent forward workers must be > 0")
	}
	if cfg.Timeout <= 0 {
		return nil, errors.New("recent forward timeout must be > 0")
	}

	ctx, cancel := context.WithCancel(context.Background())
	f := &Forwarder{
		writer:  writer,
		timeout: cfg.Timeout,
		metrics: cfg.Metrics,
		ctx:     ctx,
		cancel:  cancel,
		queue:   make(chan item, cfg.QueueSize),
	}
	for range cfg.Workers {
		f.wg.Add(1)
		go f.runWorker()
	}
	return f, nil
}

func (f *Forwarder) EnqueueCrawlLog(crawlLog *logV1.CrawlLog) {
	if crawlLog == nil {
		return
	}
	f.enqueue(item{
		logType:  CrawlLogType,
		crawlLog: proto.Clone(crawlLog).(*logV1.CrawlLog),
	})
}

func (f *Forwarder) EnqueuePageLog(pageLog *logV1.PageLog) {
	if pageLog == nil {
		return
	}
	f.enqueue(item{
		logType: PageLogType,
		pageLog: proto.Clone(pageLog).(*logV1.PageLog),
	})
}

func (f *Forwarder) enqueue(entry item) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		f.recordDrop(entry, OutcomeShutdownDrop)
		return
	}
	select {
	case f.queue <- entry:
		f.metrics.addQueueDepth(1)
	default:
		f.recordDrop(entry, OutcomeQueueFull)
	}
}

func (f *Forwarder) Close(ctx context.Context) error {
	f.mu.Lock()
	if !f.closed {
		f.closed = true
		close(f.queue)
	}
	f.mu.Unlock()

	done := make(chan struct{})
	go func() {
		f.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		f.cancel()
		return nil
	case <-ctx.Done():
		f.cancel()
		<-done
		for entry := range f.queue {
			f.metrics.addQueueDepth(-1)
			f.recordDrop(entry, OutcomeShutdownDrop)
		}
		return ctx.Err()
	}
}

func (f *Forwarder) runWorker() {
	defer f.wg.Done()
	for {
		select {
		case <-f.ctx.Done():
			return
		case entry, ok := <-f.queue:
			if !ok {
				return
			}
			f.metrics.addQueueDepth(-1)
			if f.ctx.Err() != nil {
				f.recordDrop(entry, OutcomeShutdownDrop)
				return
			}
			f.forward(entry)
		}
	}
}

func (f *Forwarder) forward(entry item) {
	ctx, cancel := context.WithTimeout(f.ctx, f.timeout)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, ForwardedMetadataKey, "true")

	var err error
	switch entry.logType {
	case CrawlLogType:
		err = f.writer.WriteCrawlLog(ctx, entry.crawlLog)
	case PageLogType:
		err = f.writer.WritePageLog(ctx, entry.pageLog)
	}
	if err == nil {
		f.metrics.record(entry.logType, OutcomeSuccess)
		return
	}
	if f.ctx.Err() != nil {
		f.recordDrop(entry, OutcomeShutdownDrop)
		return
	}
	f.metrics.record(entry.logType, OutcomeFailure)
	slog.Error("Failed to forward archived log to recent log service",
		"error", err,
		"type", entry.logType,
		"warcId", entry.warcID(),
		"executionId", entry.executionID(),
	)
}

func (f *Forwarder) recordDrop(entry item, outcome string) {
	f.metrics.record(entry.logType, outcome)
	slog.Warn("Dropped recent-log forward",
		"outcome", outcome,
		"type", entry.logType,
		"warcId", entry.warcID(),
		"executionId", entry.executionID(),
	)
}

func (i item) warcID() string {
	if i.crawlLog != nil {
		return i.crawlLog.GetWarcId()
	}
	return i.pageLog.GetWarcId()
}

func (i item) executionID() string {
	if i.crawlLog != nil {
		return i.crawlLog.GetExecutionId()
	}
	return i.pageLog.GetExecutionId()
}

type Metrics struct {
	queueDepth prometheus.Gauge
	total      *prometheus.CounterVec
}

func NewMetrics(registerer prometheus.Registerer) *Metrics {
	metrics := &Metrics{
		queueDepth: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "veidemann",
			Subsystem: "recent_forward",
			Name:      "queue_depth",
			Help:      "Number of archived logs waiting to be forwarded to the recent log service.",
		}),
		total: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "veidemann",
			Subsystem: "recent_forward",
			Name:      "total",
			Help:      "Outcomes of asynchronous recent-log forwarding.",
		}, []string{"type", "outcome"}),
	}
	registerer.MustRegister(metrics.queueDepth, metrics.total)
	for _, logType := range []string{CrawlLogType, PageLogType} {
		for _, outcome := range []string{OutcomeSuccess, OutcomeFailure, OutcomeQueueFull, OutcomeShutdownDrop} {
			metrics.total.WithLabelValues(logType, outcome)
		}
	}
	return metrics
}

func (m *Metrics) addQueueDepth(delta float64) {
	if m != nil {
		m.queueDepth.Add(delta)
	}
}

func (m *Metrics) record(logType, outcome string) {
	if m != nil {
		m.total.WithLabelValues(logType, outcome).Inc()
	}
}

func NewLogWriter(client logV1.LogClient) LogWriter {
	return &logclient.LogWriter{LogClient: client}
}
