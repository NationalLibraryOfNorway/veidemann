package upload

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/contentwriter/internal/metrics"
	"github.com/minio/minio-go/v7"
	"github.com/rs/zerolog/log"
)

const (
	defaultQueueSize    = 1024
	defaultScanInterval = time.Minute
)

var ErrManagerClosed = errors.New("upload manager is closed")

type Uploader interface {
	Upload(ctx context.Context, filePath string, md5sum string) (minio.UploadInfo, error)
}

type ManagerConfig struct {
	QueueSize     int
	WarcDir       string
	FallbackDir   string
	InstanceID    string
	ScanInterval  time.Duration
	UploadTimeout time.Duration
	Uploader      Uploader
}

type Manager struct {
	queue         chan string
	uploader      Uploader
	warcDir       string
	fallbackDir   string
	scanInterval  time.Duration
	uploadTimeout time.Duration

	closeCh     chan struct{}
	scannerDone chan struct{}
	closeOnce   sync.Once

	mu      sync.Mutex
	pending map[string]struct{}
}

func NewManager(cfg ManagerConfig) (*Manager, error) {
	warcDir := filepath.Clean(cfg.WarcDir)
	fallbackDir := ""
	if cfg.FallbackDir != "" {
		resolvedFallbackDir, err := fallbackSubdir(cfg.FallbackDir, cfg.InstanceID)
		if err != nil {
			return nil, err
		}
		fallbackDir = resolvedFallbackDir
	}

	if cfg.Uploader == nil {
		if fallbackDir == "" {
			return nil, errors.New("uploader or fallback dir is required")
		}
	}
	if cfg.WarcDir == "" {
		return nil, errors.New("warc dir is required")
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = defaultQueueSize
	}
	if cfg.ScanInterval <= 0 {
		cfg.ScanInterval = defaultScanInterval
	}
	if fallbackDir != "" && fallbackDir == warcDir {
		return nil, errors.New("fallback dir must differ from warc dir")
	}

	return &Manager{
		queue:         make(chan string, cfg.QueueSize),
		uploader:      cfg.Uploader,
		warcDir:       warcDir,
		fallbackDir:   fallbackDir,
		scanInterval:  cfg.ScanInterval,
		uploadTimeout: cfg.UploadTimeout,
		closeCh:       make(chan struct{}),
		scannerDone:   make(chan struct{}),
		pending:       make(map[string]struct{}),
	}, nil
}

func fallbackSubdir(root string, instanceID string) (string, error) {
	cleanRoot := filepath.Clean(root)
	resolvedInstanceID := sanitizePathComponent(instanceID)
	if resolvedInstanceID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return "", fmt.Errorf("failed to resolve instance id for fallback dir: %w", err)
		}
		resolvedInstanceID = sanitizePathComponent(hostname)
	}
	if resolvedInstanceID == "" {
		return "", errors.New("instance id for fallback dir is required")
	}
	return filepath.Join(cleanRoot, resolvedInstanceID), nil
}

func sanitizePathComponent(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	builder := strings.Builder{}
	builder.Grow(len(trimmed))
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '.', r == '-', r == '_':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}

	cleaned := strings.Trim(builder.String(), ".")
	if cleaned == "" || cleaned == ".." {
		return ""
	}
	return cleaned
}

func (m *Manager) Enqueue(path string) error {
	return m.enqueue(path)
}

func (m *Manager) ScanNow() error {
	if err := m.scanDirs(m.scanDirsForStartup()); err != nil {
		return err
	}
	m.refreshFallbackBacklogMetrics()
	return nil
}

// Close signals that no more files should be discovered or enqueued.
// Run will finish remaining queued paths and return.
func (m *Manager) Close() {
	m.closeOnce.Do(func() {
		close(m.closeCh)
		<-m.scannerDone
		close(m.queue)
	})
}

func (m *Manager) Run(ctx context.Context) error {
	scannerErrCh := make(chan error, 1)
	go m.runScanner(scannerErrCh)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-scannerErrCh:
			if err != nil {
				return err
			}
		case path, ok := <-m.queue:
			if !ok {
				return nil
			}
			if err := m.process(ctx, path); err != nil {
				return err
			}
		}
	}
}

func (m *Manager) runScanner(errCh chan<- error) {
	defer close(m.scannerDone)

	if err := m.scanDirs(m.scanDirsForStartup()); err != nil {
		if !errors.Is(err, ErrManagerClosed) {
			select {
			case errCh <- err:
			default:
			}
		}
		return
	}
	m.refreshFallbackBacklogMetrics()

	if m.fallbackDir == "" || m.uploader == nil {
		return
	}

	ticker := time.NewTicker(m.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.closeCh:
			return
		case <-ticker.C:
			if err := m.scanDirs([]string{m.fallbackDir}); err != nil {
				if !errors.Is(err, ErrManagerClosed) {
					select {
					case errCh <- err:
					default:
					}
				}
				return
			}
			m.refreshFallbackBacklogMetrics()
		}
	}
}

func (m *Manager) scanDirsForStartup() []string {
	dirs := []string{m.warcDir}
	if m.fallbackDir != "" && m.uploader != nil {
		dirs = append(dirs, m.fallbackDir)
	}
	return dirs
}

func (m *Manager) scanDirs(dirs []string) error {
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		candidateFn := uploadCandidates
		if m.uploader == nil && dir == m.warcDir {
			candidateFn = uploadCandidatesClosed
		}
		paths, err := candidateFn(dir)
		if err != nil {
			return fmt.Errorf("failed to scan upload candidates in %s: %w", dir, err)
		}
		for _, path := range paths {
			if err := m.enqueue(path); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Manager) enqueue(path string) error {
	cleanPath := filepath.Clean(path)
	if cleanPath == "." {
		return nil
	}

	select {
	case <-m.closeCh:
		return ErrManagerClosed
	default:
	}

	m.mu.Lock()
	if _, ok := m.pending[cleanPath]; ok {
		m.mu.Unlock()
		return nil
	}
	m.pending[cleanPath] = struct{}{}
	m.mu.Unlock()

	select {
	case <-m.closeCh:
		m.clearPending(cleanPath)
		return ErrManagerClosed
	case m.queue <- cleanPath:
		return nil
	}
}

func (m *Manager) clearPending(path string) {
	m.mu.Lock()
	delete(m.pending, path)
	m.mu.Unlock()
}

func (m *Manager) process(ctx context.Context, path string) error {
	cleanPath := filepath.Clean(path)
	if _, err := os.Stat(cleanPath); err != nil {
		m.clearPending(cleanPath)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to stat queued file: %s: %w", cleanPath, err)
	}

	fromFallback := isWithinDir(cleanPath, m.fallbackDir)
	if m.uploader == nil {
		if m.fallbackDir == "" || fromFallback {
			m.clearPending(cleanPath)
			m.refreshFallbackBacklogMetrics()
			return nil
		}

		fallbackPath, moveErr := moveFileToDir(cleanPath, m.fallbackDir)
		if moveErr != nil {
			metrics.FallbackMoveFailed()
			m.clearPending(cleanPath)
			return fmt.Errorf("failed to move file to fallback storage: %s: %w", cleanPath, moveErr)
		}

		metrics.FallbackMoved()
		log.Warn().
			Str("file", cleanPath).
			Str("fallback_file", fallbackPath).
			Msg("S3 upload not configured, moved file to fallback storage")

		m.clearPending(cleanPath)
		m.refreshFallbackBacklogMetrics()
		return nil
	}

	if fromFallback {
		metrics.FallbackRetryAttempt()
	}

	uploadCtx := ctx
	if m.uploadTimeout > 0 {
		var cancel context.CancelFunc
		uploadCtx, cancel = context.WithTimeout(ctx, m.uploadTimeout)
		defer cancel()
	}

	err := finalize(uploadCtx, cleanPath, m.uploader)
	if err == nil {
		m.clearPending(cleanPath)
		m.refreshFallbackBacklogMetrics()
		return nil
	}

	metrics.UploadFailed()
	if m.fallbackDir == "" {
		m.clearPending(cleanPath)
		return err
	}

	if fromFallback {
		log.Warn().Err(err).Str("file", cleanPath).Msg("Upload failed for fallback file, will retry")
		m.clearPending(cleanPath)
		m.refreshFallbackBacklogMetrics()
		return nil
	}

	fallbackPath, moveErr := moveFileToDir(cleanPath, m.fallbackDir)
	if moveErr != nil {
		metrics.FallbackMoveFailed()
		m.clearPending(cleanPath)
		return fmt.Errorf("failed to move file to fallback storage after upload error: %s: %w", cleanPath, moveErr)
	}

	metrics.FallbackMoved()
	log.Warn().Err(err).
		Str("file", cleanPath).
		Str("fallback_file", fallbackPath).
		Msg("Upload failed, moved file to fallback storage")

	m.clearPending(cleanPath)
	m.refreshFallbackBacklogMetrics()
	return nil
}

func (m *Manager) refreshFallbackBacklogMetrics() {
	if m.fallbackDir == "" {
		metrics.SetFallbackBacklog(0, 0)
		return
	}

	files, bytes, err := backlogStats(m.fallbackDir)
	if err != nil {
		log.Warn().Err(err).Str("dir", m.fallbackDir).Msg("Failed to refresh fallback backlog metrics")
		return
	}
	metrics.SetFallbackBacklog(files, bytes)
}

func finalize(ctx context.Context, filePath string, uploader Uploader) error {
	md5sum, err := calculateMD5(filePath)
	if err != nil {
		return err
	}

	start := time.Now()
	info, err := uploader.Upload(ctx, filePath, md5sum)
	dur := time.Since(start)

	if err != nil {
		return fmt.Errorf("failed to upload file: %s: %w", filePath, err)
	}

	metrics.UploadDuration(dur)
	metrics.UploadedSizeBytes(info.Size)

	if st, err := os.Stat(filePath); err == nil {
		diskSize := st.Size()
		metrics.OnDiskSizeBytes(diskSize)

		if diskSize != info.Size {
			metrics.UploadSizeMismatch()

			log.Warn().
				Str("key", info.Key).
				Str("file", filePath).
				Int64("disk_size", diskSize).
				Int64("uploaded_size", info.Size).
				Int64("delta", info.Size-diskSize).
				Dur("duration", dur).
				Msg("Uploaded size differs from on-disk size")
		}
	} else {
		metrics.OnDiskStatFailed()
		log.Warn().Err(err).Str("file", filePath).Msg("Failed to stat file after upload")
	}

	log.Debug().
		Str("key", info.Key).
		Int64("size", info.Size).
		Str("etag", info.ETag).
		Str("duration", time.Since(start).String()).
		Str("md5", md5sum).
		Msg("Uploaded file")

	if err = os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to remove file after upload: %s: %w", filePath, err)
	}

	return nil
}
