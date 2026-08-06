package parquet

import (
	"io"
	"os"

	parquetgo "github.com/parquet-go/parquet-go"
)

func readCrawlRowsFromFile(path string) ([]crawlLogRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	reader := parquetgo.NewGenericReader[crawlLogRow](file)
	defer func() { _ = reader.Close() }()
	return readAllRows(reader)
}

func readPageRowsFromFile(path string) ([]pageLogRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	reader := parquetgo.NewGenericReader[pageLogRow](file)
	defer func() { _ = reader.Close() }()
	return readAllRows(reader)
}

func readResourceRowsFromFile(path string) ([]resourceRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	reader := parquetgo.NewGenericReader[resourceRow](file)
	defer func() { _ = reader.Close() }()
	return readAllRows(reader)
}

type rowReader[T any] interface {
	Read(rows []T) (int, error)
}

func readAllRows[T any](reader rowReader[T]) ([]T, error) {
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
