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
	defaultAsyncS3QueueSize = 64
	defaultAsyncS3Workers   = 2
	parquetContentType      = "application/vnd.apache.parquet"
)

var ErrAsyncS3HandoffClosed = errors.New("async s3 handoff is closed")

type AsyncS3HandoffConfig struct {
	Bucket        string
	KeyPrefix     string
	QueueSize     int
	Workers       int
	UploadTimeout time.Duration
	OnError       func(file FinalizedParquetFile, err error)
}

type s3FileUploader interface {
	FPutObject(ctx context.Context, bucketName, objectName, filePath string, opts minio.PutObjectOptions) (minio.UploadInfo, error)
}

type AsyncS3Handoff struct {
	client        s3FileUploader
	bucket        string
	keyPrefix     string
	queue         chan FinalizedParquetFile
	uploadTimeout time.Duration
	onError       func(file FinalizedParquetFile, err error)
	stopCh        chan struct{}
	closeOnce     sync.Once
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

	handoff := &AsyncS3Handoff{
		client:        client,
		bucket:        cfg.Bucket,
		keyPrefix:     cfg.KeyPrefix,
		queue:         make(chan FinalizedParquetFile, cfg.QueueSize),
		uploadTimeout: cfg.UploadTimeout,
		onError:       cfg.OnError,
		stopCh:        make(chan struct{}),
	}
	for range cfg.Workers {
		go handoff.runWorker()
	}
	return handoff, nil
}

func (h *AsyncS3Handoff) HandoffFinalizedFile(file FinalizedParquetFile) error {
	select {
	case <-h.stopCh:
		return ErrAsyncS3HandoffClosed
	default:
	}

	select {
	case h.queue <- file:
		return nil
	default:
		go h.upload(file)
		return nil
	}
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

func (h *AsyncS3Handoff) upload(file FinalizedParquetFile) {
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
