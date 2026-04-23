package parquet

import (
	"io"
	"os"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	parquetgo "github.com/parquet-go/parquet-go"
)

func (s *Storage) readAllCrawlRows() ([]crawlLogRow, error) {
	files, err := s.indexedParquetFiles(tableCrawlLog)
	if err != nil {
		return nil, err
	}

	rows := make([]crawlLogRow, 0)
	for _, file := range files {
		fileRows, err := readCrawlRowsFromFile(file)
		if err != nil {
			return nil, err
		}
		rows = append(rows, fileRows...)
	}
	return rows, nil
}

func (s *Storage) readAllPageRows() ([]pageLogRow, error) {
	files, err := s.indexedParquetFiles(tablePageLog)
	if err != nil {
		return nil, err
	}

	rows := make([]pageLogRow, 0)
	for _, file := range files {
		fileRows, err := readPageRowsFromFile(file)
		if err != nil {
			return nil, err
		}
		rows = append(rows, fileRows...)
	}
	return rows, nil
}

func (s *Storage) readResourcesByPageID() (map[string][]*logV1.PageLog_Resource, error) {
	files, err := s.indexedParquetFiles(tableResource)
	if err != nil {
		return nil, err
	}

	resourcesByPageID := make(map[string][]*logV1.PageLog_Resource)
	for _, file := range files {
		fileRows, err := readResourceRowsFromFile(file)
		if err != nil {
			return nil, err
		}
		for i := range fileRows {
			resourcesByPageID[fileRows[i].PageID] = append(resourcesByPageID[fileRows[i].PageID], resourceRowToProto(&fileRows[i]))
		}
	}
	return resourcesByPageID, nil
}

func readCrawlRowsFromFile(path string) ([]crawlLogRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	reader := parquetgo.NewGenericReader[crawlLogRow](file)
	defer func() { _ = reader.Close() }()
	return readAll(reader)
}

func readPageRowsFromFile(path string) ([]pageLogRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	reader := parquetgo.NewGenericReader[pageLogRow](file)
	defer func() { _ = reader.Close() }()
	return readAll(reader)
}

func readResourceRowsFromFile(path string) ([]resourceRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	reader := parquetgo.NewGenericReader[resourceRow](file)
	defer func() { _ = reader.Close() }()
	return readAll(reader)
}

type genericReader[T any] interface {
	Read(rows []T) (int, error)
}

func readAll[T any](reader genericReader[T]) ([]T, error) {
	batch := make([]T, 1024)
	result := make([]T, 0)
	for {
		n, err := reader.Read(batch)
		if n > 0 {
			result = append(result, batch[:n]...)
		}
		if err == io.EOF {
			return result, nil
		}
		if err != nil {
			return nil, err
		}
	}
}
