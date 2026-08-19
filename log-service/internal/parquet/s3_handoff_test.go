package parquet

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/minio/minio-go/v7"
)

type uploadedObject struct {
	bucket   string
	key      string
	filePath string
	opts     minio.PutObjectOptions
}

type fakeS3Uploader struct {
	uploaded chan uploadedObject
	blockCh  <-chan struct{}
}

func (f *fakeS3Uploader) FPutObject(ctx context.Context, bucketName, objectName, filePath string, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	if f.blockCh != nil {
		select {
		case <-ctx.Done():
			return minio.UploadInfo{}, ctx.Err()
		case <-f.blockCh:
		}
	}
	if f.uploaded != nil {
		f.uploaded <- uploadedObject{
			bucket:   bucketName,
			key:      objectName,
			filePath: filePath,
			opts:     opts,
		}
	}
	return minio.UploadInfo{}, nil
}

func waitForFileCleanup(t *testing.T, path string) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected uploaded file to be cleaned up, stat err=%v", err)
	}
}

func waitForIndexFileCount(t *testing.T, dir string, want int) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		index, err := readIndexFile(dir)
		if err == nil && len(index.Files) == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	index, err := readIndexFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Files) != want {
		t.Fatalf("expected index file count %d, got %+v", want, index.Files)
	}
}

func TestCalculateMD5(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "file.parquet")
	if err := os.WriteFile(filePath, []byte("parquet"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := calculateMD5(filePath)
	if err != nil {
		t.Fatal(err)
	}
	const want = "5c8844a97bf2298a6724856911dde080"
	if got != want {
		t.Fatalf("expected MD5 %s, got %s", want, got)
	}
}

func TestAsyncS3HandoffUploadsInBackground(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "crawl_log_1.parquet")
	if err := os.WriteFile(filePath, []byte("parquet"), 0o644); err != nil {
		t.Fatal(err)
	}

	uploader := &fakeS3Uploader{uploaded: make(chan uploadedObject, 1)}
	handoff, err := newAsyncS3Handoff(uploader, AsyncS3HandoffConfig{
		Bucket:    "bucket-a",
		KeyPrefix: "archive",
		Workers:   1,
		QueueSize: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := handoff.Close(); err != nil {
			t.Errorf("close handoff: %v", err)
		}
	})

	if err := handoff.HandoffFinalizedFile(FinalizedParquetFile{
		Table:      tableCrawlLog,
		Collection: "collection-a",
		Path:       filePath,
		RowCount:   3,
	}); err != nil {
		t.Fatal(err)
	}

	select {
	case uploaded := <-uploader.uploaded:
		expectedKey := "archive/crawl_log/collection-a/crawl_log_1.parquet"
		if uploaded.bucket != "bucket-a" {
			t.Fatalf("expected bucket bucket-a, got %s", uploaded.bucket)
		}
		if uploaded.key != expectedKey {
			t.Fatalf("expected key %s, got %s", expectedKey, uploaded.key)
		}
		if uploaded.filePath != filePath {
			t.Fatalf("expected path %s, got %s", filePath, uploaded.filePath)
		}
		if uploaded.opts.ContentType != parquetContentType {
			t.Fatalf("expected content type %s, got %s", parquetContentType, uploaded.opts.ContentType)
		}
		expectedMetadata := map[string]string{"md5": "5c8844a97bf2298a6724856911dde080"}
		if len(uploaded.opts.UserMetadata) != len(expectedMetadata) || uploaded.opts.UserMetadata["md5"] != expectedMetadata["md5"] {
			t.Fatalf("expected user metadata %v, got %v", expectedMetadata, uploaded.opts.UserMetadata)
		}
		waitForFileCleanup(t, filePath)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for async s3 upload")
	}
	if err := handoff.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestAsyncS3HandoffChecksumFailureRetainsIndexEntry(t *testing.T) {
	t.Parallel()

	collectionDir := filepath.Join(t.TempDir(), tableCrawlLog, "collection-a")
	filePath := filepath.Join(collectionDir, "crawl_log_1.parquet")
	if err := os.MkdirAll(filePath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := appendIndexEntry(collectionDir, indexEntry{Name: filepath.Base(filePath), RowCount: 3}); err != nil {
		t.Fatal(err)
	}

	uploader := &fakeS3Uploader{uploaded: make(chan uploadedObject, 1)}
	uploadErrors := make(chan error, 1)
	handoff, err := newAsyncS3Handoff(uploader, AsyncS3HandoffConfig{
		Bucket:    "bucket-a",
		Workers:   1,
		QueueSize: 1,
		OnError: func(_ FinalizedParquetFile, err error) {
			uploadErrors <- err
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := handoff.Close(); err != nil {
			t.Errorf("close handoff: %v", err)
		}
	})

	if err := handoff.HandoffFinalizedFile(FinalizedParquetFile{
		Table:      tableCrawlLog,
		Collection: "collection-a",
		Path:       filePath,
		RowCount:   3,
	}); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-uploadErrors:
		if !strings.Contains(err.Error(), "calculate md5") {
			t.Fatalf("expected checksum error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for checksum error")
	}

	select {
	case uploaded := <-uploader.uploaded:
		t.Fatalf("expected checksum failure to skip upload, got %+v", uploaded)
	case <-time.After(50 * time.Millisecond):
	}
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected source path to remain after checksum failure: %v", err)
	}
	waitForIndexFileCount(t, collectionDir, 1)
}

func TestStorageCloseDoesNotWaitForAsyncS3Upload(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	blockCh := make(chan struct{})
	uploader := &fakeS3Uploader{
		uploaded: make(chan uploadedObject, 2),
		blockCh:  blockCh,
	}
	handoff, err := newAsyncS3Handoff(uploader, AsyncS3HandoffConfig{
		Bucket:    "bucket-a",
		Workers:   1,
		QueueSize: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := handoff.Close(); err != nil {
			t.Errorf("close handoff: %v", err)
		}
	})

	store, err := New(dir, 100, WithPostCloseHandoff(handoff))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WriteCrawlLog(&logV1.CrawlLog{
		WarcId:              "w-async",
		ExecutionId:         "exec-async",
		CollectionFinalName: "collection-async",
	}); err != nil {
		t.Fatal(err)
	}

	closed := make(chan error, 1)
	go func() {
		closed <- store.Close()
	}()

	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("storage close blocked on async upload")
	}

	close(blockCh)

	select {
	case uploaded := <-uploader.uploaded:
		if uploaded.bucket != "bucket-a" {
			t.Fatalf("expected bucket bucket-a, got %s", uploaded.bucket)
		}
		waitForFileCleanup(t, uploaded.filePath)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delayed async upload to finish")
	}
	if err := handoff.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestAsyncS3HandoffCloseWaitsForActiveUpload(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "handoff")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(dir, "crawl_log_1.parquet")
	if err := os.WriteFile(filePath, []byte("parquet"), 0o644); err != nil {
		t.Fatal(err)
	}

	blockCh := make(chan struct{})
	var releaseOnce sync.Once
	releaseUpload := func() { releaseOnce.Do(func() { close(blockCh) }) }
	handoff, err := newAsyncS3Handoff(&fakeS3Uploader{blockCh: blockCh}, AsyncS3HandoffConfig{
		BaseDir:      dir,
		Bucket:       "bucket-a",
		Workers:      1,
		QueueSize:    1,
		ScanInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		releaseUpload()
		if err := handoff.Close(); err != nil {
			t.Errorf("close handoff: %v", err)
		}
	})

	if err := handoff.HandoffFinalizedFile(FinalizedParquetFile{Path: filePath}); err != nil {
		t.Fatal(err)
	}
	closed := make(chan error, 1)
	go func() {
		closed <- handoff.Close()
	}()

	select {
	case err := <-closed:
		t.Fatalf("close returned while upload was active: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	releaseUpload()
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("remove handoff directory after close: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("handoff directory still exists after removal: %v", err)
	}
}

func TestDelayedS3HandoffUploadsAfterRetention(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initialTime := time.Date(2026, time.April, 29, 12, 0, 0, 0, time.UTC)
	var currentUnixNano atomic.Int64
	currentUnixNano.Store(initialTime.UnixNano())
	uploader := &fakeS3Uploader{uploaded: make(chan uploadedObject, 1)}
	handoff, err := newAsyncS3Handoff(uploader, AsyncS3HandoffConfig{
		BaseDir:      dir,
		Bucket:       "bucket-a",
		KeyPrefix:    "archive",
		UploadDelay:  72 * time.Hour,
		ScanInterval: 24 * time.Hour,
		Workers:      1,
		QueueSize:    1,
		Now: func() time.Time {
			return time.Unix(0, currentUnixNano.Load()).UTC()
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := handoff.Close(); err != nil {
			t.Errorf("close handoff: %v", err)
		}
	})

	store, err := New(dir, 100, WithPostCloseHandoff(handoff))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WriteCrawlLog(&logV1.CrawlLog{
		WarcId:              "w-delayed",
		ExecutionId:         "exec-delayed",
		CollectionFinalName: "collection-delayed",
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	collectionDir := filepath.Join(dir, tableCrawlLog, "collection-delayed")
	index, err := readIndexFile(collectionDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Files) != 1 {
		t.Fatalf("expected one finalized delayed file, got %+v", index.Files)
	}
	currentTime := time.UnixMilli(index.Files[0].FinalizedAtUnixMilli).UTC()
	currentUnixNano.Store(currentTime.UnixNano())

	select {
	case uploaded := <-uploader.uploaded:
		t.Fatalf("expected no upload before retention elapsed, got %+v", uploaded)
	case <-time.After(50 * time.Millisecond):
	}

	currentTime = currentTime.Add(72*time.Hour - time.Second)
	currentUnixNano.Store(currentTime.UnixNano())
	if err := handoff.scanEligibleFiles(); err != nil {
		t.Fatal(err)
	}

	select {
	case uploaded := <-uploader.uploaded:
		t.Fatalf("expected no upload before retention threshold, got %+v", uploaded)
	case <-time.After(50 * time.Millisecond):
	}

	currentTime = currentTime.Add(time.Second)
	currentUnixNano.Store(currentTime.UnixNano())
	if err := handoff.scanEligibleFiles(); err != nil {
		t.Fatal(err)
	}

	select {
	case uploaded := <-uploader.uploaded:
		expectedKey := "archive/crawl_log/collection-delayed/"
		if uploaded.bucket != "bucket-a" {
			t.Fatalf("expected bucket bucket-a, got %s", uploaded.bucket)
		}
		if len(uploaded.key) <= len(expectedKey) || uploaded.key[:len(expectedKey)] != expectedKey {
			t.Fatalf("expected key prefix %s, got %s", expectedKey, uploaded.key)
		}
		waitForFileCleanup(t, uploaded.filePath)
		waitForIndexFileCount(t, collectionDir, 0)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delayed upload after retention")
	}
	if err := handoff.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCleanupUploadedFileRemovesIndexEntry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	collectionDir := filepath.Join(dir, tableCrawlLog, "collection-a")
	if err := os.MkdirAll(collectionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(collectionDir, "crawl_log_1.parquet")
	if err := os.WriteFile(filePath, []byte("parquet"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := appendIndexEntry(collectionDir, indexEntry{Name: filepath.Base(filePath), RowCount: 1}); err != nil {
		t.Fatal(err)
	}

	if err := cleanupUploadedFile(FinalizedParquetFile{
		Table:      tableCrawlLog,
		Collection: "collection-a",
		Path:       filePath,
		RowCount:   1,
	}); err != nil {
		t.Fatal(err)
	}

	index, err := readIndexFile(collectionDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Files) != 0 {
		t.Fatalf("expected cleanup to remove index entry, got %+v", index.Files)
	}
}
