package parquet

import (
	"context"
	"os"
	"path/filepath"
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
	defer func() { _ = handoff.Close() }()

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
		if uploaded.opts.UserMetadata["veidemann-row-count"] != "3" {
			t.Fatalf("expected row count metadata 3, got %s", uploaded.opts.UserMetadata["veidemann-row-count"])
		}
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Fatalf("expected uploaded file to be cleaned up, stat err=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for async s3 upload")
	}
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
	defer func() { _ = handoff.Close() }()

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
		if _, err := os.Stat(uploaded.filePath); !os.IsNotExist(err) {
			t.Fatalf("expected uploaded file to be cleaned up, stat err=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delayed async upload to finish")
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
