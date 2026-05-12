package upload

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
)

type stubUploader struct {
	mu    sync.Mutex
	calls []string
	fn    func(filePath string, attempt int) error
}

func (u *stubUploader) Upload(ctx context.Context, filePath string, md5sum string) (minio.UploadInfo, error) {
	u.mu.Lock()
	u.calls = append(u.calls, filePath)
	attempt := len(u.calls)
	u.mu.Unlock()

	if u.fn != nil {
		if err := u.fn(filePath, attempt); err != nil {
			return minio.UploadInfo{}, err
		}
	}

	st, err := os.Stat(filePath)
	if err != nil {
		return minio.UploadInfo{}, err
	}

	return minio.UploadInfo{
		Key:  filepath.Base(filePath),
		Size: st.Size(),
		ETag: "etag",
	}, nil
}

func (u *stubUploader) Calls() []string {
	u.mu.Lock()
	defer u.mu.Unlock()

	paths := make([]string, len(u.calls))
	copy(paths, u.calls)
	return paths
}

func TestManagerStartupScanUploadsExistingFiles(t *testing.T) {
	warcDir := t.TempDir()
	fallbackRoot := t.TempDir()
	instanceID := "contentwriter-a"
	fallbackDir := filepath.Join(fallbackRoot, instanceID)
	assert.NoError(t, os.MkdirAll(fallbackDir, 0o755))
	warcFile := writeUploadFile(t, warcDir, "existing.warc")
	fallbackFile := writeUploadFile(t, fallbackDir, "retry.warc.gz")
	uploader := &stubUploader{}

	manager, err := NewManager(ManagerConfig{
		QueueSize:    8,
		WarcDir:      warcDir,
		FallbackDir:  fallbackRoot,
		InstanceID:   instanceID,
		ScanInterval: time.Hour,
		Uploader:     uploader,
	})
	if !assert.NoError(t, err) {
		return
	}

	errCh := make(chan error, 1)
	go func() { errCh <- manager.Run(context.Background()) }()

	assert.Eventually(t, func() bool {
		_, warcErr := os.Stat(warcFile)
		_, fallbackErr := os.Stat(fallbackFile)
		return errors.Is(warcErr, os.ErrNotExist) &&
			errors.Is(fallbackErr, os.ErrNotExist) &&
			len(uploader.Calls()) == 2
	}, time.Second, 10*time.Millisecond)

	manager.Close()
	assert.NoError(t, <-errCh)
}

func TestManagerDeduplicatesQueuedPaths(t *testing.T) {
	warcDir := t.TempDir()
	uploader := &stubUploader{}

	manager, err := NewManager(ManagerConfig{
		QueueSize: 4,
		WarcDir:   warcDir,
		Uploader:  uploader,
	})
	if !assert.NoError(t, err) {
		return
	}

	errCh := make(chan error, 1)
	go func() { errCh <- manager.Run(context.Background()) }()
	<-manager.scannerDone

	path := writeUploadFile(t, warcDir, "dedupe.warc")
	assert.NoError(t, manager.Enqueue(path))
	assert.NoError(t, manager.Enqueue(path))

	assert.Eventually(t, func() bool {
		_, err := os.Stat(path)
		return errors.Is(err, os.ErrNotExist) && len(uploader.Calls()) == 1
	}, time.Second, 10*time.Millisecond)

	manager.Close()
	assert.NoError(t, <-errCh)
}

func TestManagerMovesToFallbackAndRetries(t *testing.T) {
	warcDir := t.TempDir()
	fallbackRoot := t.TempDir()
	instanceID := "contentwriter-a"
	uploader := &stubUploader{
		fn: func(filePath string, attempt int) error {
			if isWithinDir(filePath, warcDir) {
				return errors.New("s3 unavailable")
			}
			return nil
		},
	}

	manager, err := NewManager(ManagerConfig{
		QueueSize:    8,
		WarcDir:      warcDir,
		FallbackDir:  fallbackRoot,
		InstanceID:   instanceID,
		ScanInterval: 10 * time.Millisecond,
		Uploader:     uploader,
	})
	if !assert.NoError(t, err) {
		return
	}

	errCh := make(chan error, 1)
	go func() { errCh <- manager.Run(context.Background()) }()
	time.Sleep(20 * time.Millisecond)

	path := writeUploadFile(t, warcDir, "fallback.warc.gz")
	fallbackDir := filepath.Join(fallbackRoot, instanceID)
	fallbackPath := filepath.Join(fallbackDir, filepath.Base(path))
	assert.NoError(t, manager.Enqueue(path))

	assert.Eventually(t, func() bool {
		calls := uploader.Calls()
		_, activeErr := os.Stat(path)
		_, fallbackErr := os.Stat(fallbackPath)
		return len(calls) >= 2 &&
			isWithinDir(calls[0], warcDir) &&
			isWithinDir(calls[1], fallbackDir) &&
			errors.Is(activeErr, os.ErrNotExist) &&
			errors.Is(fallbackErr, os.ErrNotExist)
	}, 2*time.Second, 10*time.Millisecond)

	manager.Close()
	assert.NoError(t, <-errCh)
}

func TestManagersWithSharedFallbackRootUseDifferentSubdirs(t *testing.T) {
	fallbackRoot := t.TempDir()
	uploaderA := &stubUploader{}
	uploaderB := &stubUploader{}

	managerA, err := NewManager(ManagerConfig{
		QueueSize:    4,
		WarcDir:      t.TempDir(),
		FallbackDir:  fallbackRoot,
		InstanceID:   "contentwriter-a",
		ScanInterval: time.Hour,
		Uploader:     uploaderA,
	})
	if !assert.NoError(t, err) {
		return
	}
	managerB, err := NewManager(ManagerConfig{
		QueueSize:    4,
		WarcDir:      t.TempDir(),
		FallbackDir:  fallbackRoot,
		InstanceID:   "contentwriter-b",
		ScanInterval: time.Hour,
		Uploader:     uploaderB,
	})
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, filepath.Join(fallbackRoot, "contentwriter-a"), managerA.fallbackDir)
	assert.Equal(t, filepath.Join(fallbackRoot, "contentwriter-b"), managerB.fallbackDir)

	assert.NoError(t, os.MkdirAll(managerA.fallbackDir, 0o755))
	assert.NoError(t, os.MkdirAll(managerB.fallbackDir, 0o755))
	fileA := writeUploadFile(t, managerA.fallbackDir, "a.warc.gz")
	fileB := writeUploadFile(t, managerB.fallbackDir, "b.warc.gz")

	errChA := make(chan error, 1)
	errChB := make(chan error, 1)
	go func() { errChA <- managerA.Run(context.Background()) }()
	go func() { errChB <- managerB.Run(context.Background()) }()

	assert.Eventually(t, func() bool {
		callsA := uploaderA.Calls()
		callsB := uploaderB.Calls()
		_, errA := os.Stat(fileA)
		_, errB := os.Stat(fileB)
		return len(callsA) == 1 && len(callsB) == 1 &&
			callsA[0] == fileA && callsB[0] == fileB &&
			errors.Is(errA, os.ErrNotExist) && errors.Is(errB, os.ErrNotExist)
	}, time.Second, 10*time.Millisecond)

	managerA.Close()
	managerB.Close()
	assert.NoError(t, <-errChA)
	assert.NoError(t, <-errChB)
}

func TestManagerReturnsErrorWhenUploadFailsWithoutFallback(t *testing.T) {
	warcDir := t.TempDir()
	uploader := &stubUploader{
		fn: func(filePath string, attempt int) error {
			return errors.New("s3 unavailable")
		},
	}

	manager, err := NewManager(ManagerConfig{
		QueueSize: 4,
		WarcDir:   warcDir,
		Uploader:  uploader,
	})
	if !assert.NoError(t, err) {
		return
	}

	errCh := make(chan error, 1)
	go func() { errCh <- manager.Run(context.Background()) }()
	<-manager.scannerDone

	path := writeUploadFile(t, warcDir, "failure.warc")
	assert.NoError(t, manager.Enqueue(path))

	select {
	case runErr := <-errCh:
		if assert.Error(t, runErr) {
			assert.Contains(t, runErr.Error(), "failed to upload file")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for upload manager error")
	}

	manager.Close()
	_, statErr := os.Stat(path)
	assert.NoError(t, statErr)
	assert.Len(t, uploader.Calls(), 1)
}

func TestManagerAllowsFallbackOnlyMode(t *testing.T) {
	manager, err := NewManager(ManagerConfig{
		QueueSize:   4,
		WarcDir:     t.TempDir(),
		FallbackDir: t.TempDir(),
	})
	if !assert.NoError(t, err) {
		return
	}

	assert.NotNil(t, manager)
}

func TestManagerMovesClosedFilesToFallbackWithoutUploader(t *testing.T) {
	warcDir := t.TempDir()
	fallbackRoot := t.TempDir()
	instanceID := "writer-a"

	manager, err := NewManager(ManagerConfig{
		QueueSize:   4,
		WarcDir:     warcDir,
		FallbackDir: fallbackRoot,
		InstanceID:  instanceID,
	})
	if !assert.NoError(t, err) {
		return
	}

	errCh := make(chan error, 1)
	go func() { errCh <- manager.Run(context.Background()) }()
	<-manager.scannerDone

	closedPath := writeUploadFile(t, warcDir, "closed.warc")
	openPath := writeUploadFile(t, warcDir, "active.open")

	assert.NoError(t, manager.ScanNow())

	fallbackPath := filepath.Join(fallbackRoot, instanceID, "closed.warc")
	assert.Eventually(t, func() bool {
		_, closedErr := os.Stat(closedPath)
		_, openErr := os.Stat(openPath)
		_, fallbackErr := os.Stat(fallbackPath)
		return errors.Is(closedErr, os.ErrNotExist) && openErr == nil && fallbackErr == nil
	}, time.Second, 10*time.Millisecond)

	manager.Close()
	assert.NoError(t, <-errCh)
}

func writeUploadFile(t *testing.T, dir string, name string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("test content"), 0o644); err != nil {
		t.Fatalf("failed to write test file %s: %v", path, err)
	}
	return path
}
