package parquet

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParquetWriterReaderRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		prototype any
		rows      []any
		read      func(string) ([]any, error)
	}{
		{
			name:      "crawl rows",
			prototype: &crawlLogRow{},
			rows: []any{
				&crawlLogRow{
					WarcID:              "crawl-1",
					ExecutionID:         "exec-crawl",
					JobExecutionID:      "job-crawl",
					CollectionFinalName: "collection-crawl",
					StatusCode:          200,
					Size:                1234,
					FetchTimeMs:         25,
					Retries:             1,
					RequestedURI:        "https://example.com/a",
					ResponseURI:         "https://example.com/a",
					DiscoveryPath:       "P",
					Referrer:            "https://referrer.example/a",
					ContentType:         "text/html",
					BlockDigest:         "sha1:block-a",
					PayloadDigest:       "sha1:payload-a",
					StorageRef:          "storage-a",
					RecordType:          "response",
					WarcRefersTo:        "ref-a",
					IPAddress:           "127.0.0.1",
					Method:              "GET",
					TimeStamp:           1710000000000,
					FetchTimeStamp:      1710000001000,
					ErrorCode:           0,
				},
				&crawlLogRow{
					WarcID:              "crawl-2",
					ExecutionID:         "exec-crawl",
					JobExecutionID:      "job-crawl",
					CollectionFinalName: "collection-crawl",
					StatusCode:          404,
					Size:                5678,
					FetchTimeMs:         40,
					Retries:             2,
					RequestedURI:        "https://example.com/b",
					ResponseURI:         "https://example.com/b",
					DiscoveryPath:       "P,L",
					Referrer:            "https://referrer.example/b",
					ContentType:         "text/plain",
					BlockDigest:         "sha1:block-b",
					PayloadDigest:       "sha1:payload-b",
					StorageRef:          "storage-b",
					RecordType:          "resource",
					WarcRefersTo:        "ref-b",
					IPAddress:           "127.0.0.2",
					Method:              "HEAD",
					TimeStamp:           1710000002000,
					FetchTimeStamp:      1710000003000,
					ErrorCode:           42,
					ErrorMsg:            "synthetic error",
					ErrorDetail:         "synthetic detail",
				},
			},
			read: func(path string) ([]any, error) {
				rows, err := readCrawlRowsFromFile(path)
				if err != nil {
					return nil, err
				}
				result := make([]any, len(rows))
				for i := range rows {
					result[i] = rows[i]
				}
				return result, nil
			},
		},
		{
			name:      "page rows",
			prototype: &pageLogRow{},
			rows: []any{
				&pageLogRow{
					WarcID:              "page-1",
					ExecutionID:         "exec-page",
					JobExecutionID:      "job-page",
					CollectionFinalName: "collection-page",
					URI:                 "https://example.com/page-1",
					Referrer:            "https://referrer.example/page-1",
					Method:              "GET",
					Outlinks:            "[\"https://out.example/a\",\"\"]",
				},
				&pageLogRow{
					WarcID:              "page-2",
					ExecutionID:         "exec-page",
					JobExecutionID:      "job-page",
					CollectionFinalName: "collection-page",
					URI:                 "https://example.com/page-2",
					Referrer:            "https://referrer.example/page-2",
					Method:              "POST",
					Outlinks:            "[\"https://out.example/b\"]",
				},
			},
			read: func(path string) ([]any, error) {
				rows, err := readPageRowsFromFile(path)
				if err != nil {
					return nil, err
				}
				result := make([]any, len(rows))
				for i := range rows {
					result[i] = rows[i]
				}
				return result, nil
			},
		},
		{
			name:      "resource rows",
			prototype: &resourceRow{},
			rows: []any{
				&resourceRow{
					PageID:        "page-1",
					WarcID:        "resource-1",
					URI:           "https://example.com/resource-1",
					Referrer:      "https://referrer.example/resource-1",
					ResourceType:  "image",
					ContentType:   "image/png",
					DiscoveryPath: "L,E",
					Method:        "GET",
					StatusCode:    200,
					FromCache:     true,
					Renderable:    false,
				},
				&resourceRow{
					PageID:        "page-1",
					WarcID:        "resource-2",
					URI:           "https://example.com/resource-2",
					Referrer:      "https://referrer.example/resource-2",
					ResourceType:  "script",
					ContentType:   "application/javascript",
					DiscoveryPath: "L,X",
					Method:        "GET",
					StatusCode:    500,
					FromCache:     false,
					Renderable:    true,
					ErrorCode:     13,
					ErrorMsg:      "resource error",
					ErrorDetail:   "resource detail",
				},
			},
			read: func(path string) ([]any, error) {
				rows, err := readResourceRowsFromFile(path)
				if err != nil {
					return nil, err
				}
				result := make([]any, len(rows))
				for i := range rows {
					result[i] = rows[i]
				}
				return result, nil
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			tmpPath := filepath.Join(dir, "rows.parquet.open")
			finalPath := filepath.Join(dir, "rows.parquet")

			writeFn, closeFn, err := newParquetWriter(tmpPath, tc.prototype)
			if err != nil {
				t.Fatal(err)
			}

			for _, row := range tc.rows {
				if err := writeFn(row); err != nil {
					t.Fatal(err)
				}
			}
			if err := closeFn(); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(tmpPath, finalPath); err != nil {
				t.Fatal(err)
			}

			if _, err := os.Stat(finalPath); err != nil {
				t.Fatalf("expected finalized parquet file to exist: %v", err)
			}
			if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
				t.Fatalf("expected temp parquet file to be renamed away, got err=%v", err)
			}

			got, err := tc.read(finalPath)
			if err != nil {
				t.Fatal(err)
			}
			want := dereferenceRows(tc.rows)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("unexpected rows after round trip:\nwant: %#v\ngot:  %#v", want, got)
			}
		})
	}
}

func dereferenceRows(rows []any) []any {
	result := make([]any, len(rows))
	for i, row := range rows {
		value := reflect.ValueOf(row)
		if value.Kind() == reflect.Pointer && !value.IsNil() {
			result[i] = value.Elem().Interface()
			continue
		}
		result[i] = row
	}
	return result
}
