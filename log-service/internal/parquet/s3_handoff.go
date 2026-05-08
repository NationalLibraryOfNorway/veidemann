package parquet

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
)

const (
	defaultAsyncS3QueueSize    = 64
	defaultAsyncS3Workers      = 2
	defaultAsyncS3ScanInterval = time.Minute
	parquetContentType         = "application/vnd.apache.parquet"
)

var ErrAsyncS3HandoffClosed = errors.New("async s3 handoff is closed")

type AsyncS3HandoffConfig struct {
	BaseDir       string
	Bucket        string
	KeyPrefix     string
	QueueSize     int
	Workers       int
	ScanInterval  time.Duration
	UploadDelay   time.Duration
	UploadTimeout time.Duration
	Now           func() time.Time
	OnError       func(file FinalizedParquetFile, err error)
}

type s3FileUploader interface {
	FPutObject(ctx context.Context, bucketName, objectName, filePath string, opts minio.PutObjectOptions) (minio.UploadInfo, error)
}

type AsyncS3Handoff struct {
	client        s3FileUploader
	baseDir       string
	bucket        string
	keyPrefix     string
	queue         chan FinalizedParquetFile
	scanInterval  time.Duration
	uploadDelay   time.Duration
	uploadTimeout time.Duration
	now           func() time.Time
	onError       func(file FinalizedParquetFile, err error)
	stopCh        chan struct{}
	closeOnce     sync.Once
	mu            sync.Mutex
	pending       map[string]struct{}
}

var _ PostCloseHandoff = (*AsyncS3Handoff)(nil)

func NewAsyncS3Handoff(client *minio.Client, cfg AsyncS3HandoffConfig) (*AsyncS3Handoff, error) {
	return newAsyncS3Handoff(client, cfg)
}

func newAsyncS3Handoff(client s3FileUploader, cfg AsyncS3HandoffConfig) (*AsyncS3Handoff, error) {
	if client == nil {
		return nil, errors.New("s3 client must not be nil")
	}
	if cfg.Bucket == "" {
		return nil, errors.New("s3 bucket must not be empty")
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = defaultAsyncS3QueueSize
	}
	if cfg.Workers <= 0 {
		cfg.Workers = defaultAsyncS3Workers
	}
	if cfg.UploadDelay < 0 {
		return nil, errors.New("s3 upload delay must not be negative")
	}
	if cfg.BaseDir == "" && cfg.UploadDelay > 0 {
		return nil, errors.New("s3 base dir must not be empty when upload delay is set")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.ScanInterval <= 0 {
		cfg.ScanInterval = defaultAsyncS3ScanInterval
	}

	handoff := &AsyncS3Handoff{
		client:        client,
		baseDir:       cfg.BaseDir,
		bucket:        cfg.Bucket,
		keyPrefix:     cfg.KeyPrefix,
		queue:         make(chan FinalizedParquetFile, cfg.QueueSize),
		scanInterval:  cfg.ScanInterval,
		uploadDelay:   cfg.UploadDelay,
		uploadTimeout: cfg.UploadTimeout,
		now:           cfg.Now,
		onError:       cfg.OnError,
		stopCh:        make(chan struct{}),
		pending:       make(map[string]struct{}),
	}
	for i := 0; i < cfg.Workers; i++ {
		go handoff.runWorker()
	}
	if handoff.baseDir != "" {
		go handoff.runScanner()
	}
	return handoff, nil
}

func (h *AsyncS3Handoff) HandoffFinalizedFile(file FinalizedParquetFile) error {
	if file.FinalizedAt.IsZero() {
		file.FinalizedAt = h.now().UTC()
	}
	if !h.isEligible(file, h.now().UTC()) {
		return nil
	}
	return h.enqueue(file)
}

func (h *AsyncS3Handoff) Close() error {
	h.closeOnce.Do(func() {
		close(h.stopCh)
	})
	return nil
}

func (h *AsyncS3Handoff) runWorker() {
	for {
		select {
		case <-h.stopCh:
			return
		case file := <-h.queue:
			h.upload(file)
		}
	}
}

func (h *AsyncS3Handoff) runScanner() {
	if err := h.scanEligibleFiles(); err != nil && !errors.Is(err, ErrAsyncS3HandoffClosed) {
		h.reportError(FinalizedParquetFile{Path: h.baseDir}, err)
	}

	ticker := time.NewTicker(h.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			if err := h.scanEligibleFiles(); err != nil && !errors.Is(err, ErrAsyncS3HandoffClosed) {
				h.reportError(FinalizedParquetFile{Path: h.baseDir}, err)
			}
		}
	}
}

func (h *AsyncS3Handoff) scanEligibleFiles() error {
	if h.baseDir == "" {
		return nil
	}
	select {
	case <-h.stopCh:
		return ErrAsyncS3HandoffClosed
	default:
	}

	files, err := loadFinalizedParquetFiles(h.baseDir, "")
	if err != nil {
		return err
	}
	now := h.now().UTC()
	for _, file := range files {
		if !h.isEligible(file, now) {
			continue
		}
		if err := h.enqueue(file); err != nil {
			if errors.Is(err, ErrAsyncS3HandoffClosed) {
				return nil
			}
			return err
		}
	}
	return nil
}

func (h *AsyncS3Handoff) isEligible(file FinalizedParquetFile, now time.Time) bool {
	if h.uploadDelay <= 0 {
		return true
	}
	if file.FinalizedAt.IsZero() {
		return true
	}
	return !file.FinalizedAt.Add(h.uploadDelay).After(now)
}

func (h *AsyncS3Handoff) enqueue(file FinalizedParquetFile) error {
	select {
	case <-h.stopCh:
		return ErrAsyncS3HandoffClosed
	default:
	}

	h.mu.Lock()
	if _, ok := h.pending[file.Path]; ok {
		h.mu.Unlock()
		return nil
	}
	h.pending[file.Path] = struct{}{}
	h.mu.Unlock()

	select {
	case <-h.stopCh:
		h.clearPending(file.Path)
		return ErrAsyncS3HandoffClosed
	case h.queue <- file:
		return nil
	default:
		go h.upload(file)
		return nil
	}
}

func (h *AsyncS3Handoff) clearPending(path string) {
	h.mu.Lock()
	delete(h.pending, path)
	h.mu.Unlock()
}

func (h *AsyncS3Handoff) upload(file FinalizedParquetFile) {
	defer h.clearPending(file.Path)

	ctx := context.Background()
	if h.uploadTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.uploadTimeout)
		defer cancel()
	}

	_, err := h.client.FPutObject(ctx, h.bucket, h.objectKey(file), file.Path, minio.PutObjectOptions{
		ContentType: parquetContentType,
		UserMetadata: map[string]string{
			"veidemann-table":      file.Table,
			"veidemann-collection": file.Collection,
			"veidemann-row-count":  strconv.FormatInt(file.RowCount, 10),
		},
	})
	if err != nil {
		h.reportError(file, err)
		return
	}
	if err := cleanupUploadedFile(file); err != nil {
		h.reportError(file, err)
	}
}

func (h *AsyncS3Handoff) reportError(file FinalizedParquetFile, err error) {
	if h.onError != nil {
		h.onError(file, err)
	}
}

func cleanupUploadedFile(file FinalizedParquetFile) error {
	if err := os.Remove(file.Path); err != nil {
		if os.IsNotExist(err) {
			return removeIndexEntry(filepath.Dir(file.Path), filepath.Base(file.Path))
		}
		return fmt.Errorf("remove uploaded parquet file %s: %w", file.Path, err)
	}
	if err := removeIndexEntry(filepath.Dir(file.Path), filepath.Base(file.Path)); err != nil {
		return fmt.Errorf("remove uploaded parquet index entry for %s: %w", file.Path, err)
	}
	return nil
}

func (h *AsyncS3Handoff) objectKey(file FinalizedParquetFile) string {
	parts := make([]string, 0, 4)
	if h.keyPrefix != "" {
		parts = append(parts, h.keyPrefix)
	}
	parts = append(parts, file.Table, file.Collection, filepath.Base(file.Path))
	return path.Join(parts...)
}
