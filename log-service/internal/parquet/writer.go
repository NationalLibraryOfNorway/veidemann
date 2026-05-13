package parquet

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	parquetgo "github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/compress/snappy"
)

type writerState struct {
	collection    string
	collectionDir string
	table         string
	dir           string
	tmpPath       string
	finalPath     string
	lineCount     int64
	writeFn       func(src any) error
	closeFn       func() error
}

func (s *Storage) writeRow(table, collection string, row any) error {
	collection = normalizeCollection(collection)

	s.mu.Lock()
	defer s.mu.Unlock()

	writer, err := s.getOrCreateWriterLocked(table, collection, row)
	if err != nil {
		return err
	}
	if err := writer.writeFn(row); err != nil {
		return err
	}
	writer.lineCount++
	if writer.lineCount >= s.maxLinesPerFile {
		if err := s.rotateLocked(writer, row); err != nil {
			return err
		}
	}
	return nil
}

func (s *Storage) rotateLocked(writer *writerState, rowPrototype any) error {
	if err := s.closeWriterLocked(writer); err != nil {
		return err
	}
	nextWriter, err := s.newWriterLocked(writer.table, writer.collection, rowPrototype)
	if err != nil {
		return err
	}
	s.writers[writerKey(writer.table, writer.collection)] = nextWriter
	return nil
}

func (s *Storage) getOrCreateWriterLocked(table, collection string, rowPrototype any) (*writerState, error) {
	key := writerKey(table, collection)
	if writer, ok := s.writers[key]; ok {
		return writer, nil
	}
	writer, err := s.newWriterLocked(table, collection, rowPrototype)
	if err != nil {
		return nil, err
	}
	s.writers[key] = writer
	return writer, nil
}

func (s *Storage) newWriterLocked(table, collection string, rowPrototype any) (*writerState, error) {
	collectionDir := collectionDirName(collection)
	dir := filepath.Join(s.baseDir, table, collectionDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate file ID: %w", err)
	}
	baseName := fmt.Sprintf("%s_%s.parquet", table, id)
	finalPath := filepath.Join(dir, baseName)
	tmpPath := finalPath + ".open"
	writeFn, closeFn, err := newParquetWriter(tmpPath, rowPrototype)
	if err != nil {
		return nil, err
	}

	return &writerState{
		collection:    collection,
		collectionDir: collectionDir,
		table:         table,
		dir:           dir,
		tmpPath:       tmpPath,
		finalPath:     finalPath,
		writeFn:       writeFn,
		closeFn:       closeFn,
	}, nil
}

func (s *Storage) closeWriterLocked(writer *writerState) error {
	if writer == nil || writer.closeFn == nil {
		return nil
	}

	closeFn := writer.closeFn
	writer.closeFn = nil
	writer.writeFn = nil

	if err := closeFn(); err != nil {
		return err
	}
	if err := os.Rename(writer.tmpPath, writer.finalPath); err != nil {
		return err
	}
	finalizedAt := time.Now().UTC()
	if err := appendIndexEntry(writer.dir, indexEntry{Name: filepath.Base(writer.finalPath), RowCount: writer.lineCount, FinalizedAtUnixMilli: finalizedAt.UnixMilli()}); err != nil {
		return err
	}
	if s.handoff != nil {
		if err := s.handoff.HandoffFinalizedFile(FinalizedParquetFile{
			Table:       writer.table,
			Collection:  writer.collection,
			Path:        writer.finalPath,
			RowCount:    writer.lineCount,
			FinalizedAt: finalizedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func normalizeCollection(collection string) string {
	collection = strings.TrimSpace(collection)
	if collection == "" {
		return "default"
	}
	return collection
}

func collectionDirName(collection string) string {
	return url.PathEscape(normalizeCollection(collection))
}

func collectionFromDirName(collection string) string {
	unescaped, err := url.PathUnescape(collection)
	if err != nil {
		return normalizeCollection(collection)
	}
	return normalizeCollection(unescaped)
}

func writerKey(table, collection string) string {
	return table + "::" + collection
}

func newParquetWriter(filePath string, rowPrototype any) (func(any) error, func() error, error) {
	file, err := os.Create(filePath)
	if err != nil {
		return nil, nil, err
	}

	closeWithFile := func(closeWriterFn func() error) func() error {
		return func() error {
			if err := closeWriterFn(); err != nil {
				_ = file.Close()
				return err
			}
			return file.Close()
		}
	}

	switch rowPrototype.(type) {
	case *crawlLogRow:
		writer := parquetgo.NewGenericWriter[crawlLogRow](file, parquetgo.Compression(&snappy.Codec{}))
		return func(src any) error {
			row, ok := src.(*crawlLogRow)
			if !ok {
				return fmt.Errorf("unexpected row type %T for crawl writer", src)
			}
			_, err := writer.Write([]crawlLogRow{*row})
			return err
		}, closeWithFile(writer.Close), nil
	case *pageLogRow:
		writer := parquetgo.NewGenericWriter[pageLogRow](file, parquetgo.Compression(&snappy.Codec{}))
		return func(src any) error {
			row, ok := src.(*pageLogRow)
			if !ok {
				return fmt.Errorf("unexpected row type %T for page writer", src)
			}
			_, err := writer.Write([]pageLogRow{*row})
			return err
		}, closeWithFile(writer.Close), nil
	case *resourceRow:
		writer := parquetgo.NewGenericWriter[resourceRow](file, parquetgo.Compression(&snappy.Codec{}))
		return func(src any) error {
			row, ok := src.(*resourceRow)
			if !ok {
				return fmt.Errorf("unexpected row type %T for resource writer", src)
			}
			_, err := writer.Write([]resourceRow{*row})
			return err
		}, closeWithFile(writer.Close), nil
	default:
		_ = file.Close()
		return nil, nil, fmt.Errorf("unsupported parquet row type %T", rowPrototype)
	}
}
