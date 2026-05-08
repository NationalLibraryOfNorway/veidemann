package parquet

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const indexFileName = ".index.json"

type indexFile struct {
	Files []indexEntry `json:"files"`
}

type indexEntry struct {
	Name                 string `json:"name"`
	RowCount             int64  `json:"rowCount"`
	FinalizedAtUnixMilli int64  `json:"finalizedAtUnixMilli,omitempty"`
}

func appendIndexEntry(dir string, entry indexEntry) error {
	index, err := readIndexFile(dir)
	if err != nil {
		return err
	}
	index.Files = append(index.Files, entry)
	return writeIndexFile(dir, index)
}

func removeIndexEntry(dir, name string) error {
	index, err := readIndexFile(dir)
	if err != nil {
		return err
	}

	filtered := index.Files[:0]
	for _, file := range index.Files {
		if file.Name == name {
			continue
		}
		filtered = append(filtered, file)
	}
	index.Files = filtered
	return writeIndexFile(dir, index)
}

func readIndexFile(dir string) (indexFile, error) {
	path := filepath.Join(dir, indexFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return indexFile{}, nil
		}
		return indexFile{}, err
	}
	if len(data) == 0 {
		return indexFile{}, nil
	}

	var index indexFile
	if err := json.Unmarshal(data, &index); err != nil {
		return indexFile{}, fmt.Errorf("read index file %s: %w", path, err)
	}
	return index, nil
}

func writeIndexFile(dir string, index indexFile) error {
	path := filepath.Join(dir, indexFileName)
	tmpPath := path + ".tmp"
	data, err := json.Marshal(index)
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func loadFinalizedParquetFiles(baseDir, table string) ([]FinalizedParquetFile, error) {
	rootDir := baseDir
	if table != "" {
		rootDir = filepath.Join(baseDir, table)
	}
	if _, err := os.Stat(rootDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	indexPaths := make([]string, 0)
	err := filepath.WalkDir(rootDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Name() == indexFileName {
			indexPaths = append(indexPaths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(indexPaths)

	files := make([]FinalizedParquetFile, 0)
	for _, indexPath := range indexPaths {
		dir := filepath.Dir(indexPath)
		relDir, err := filepath.Rel(baseDir, dir)
		if err != nil {
			return nil, err
		}
		parts := strings.Split(relDir, string(os.PathSeparator))
		if len(parts) != 2 {
			return nil, fmt.Errorf("unexpected parquet index path %s", indexPath)
		}

		index, err := readIndexFile(dir)
		if err != nil {
			return nil, err
		}
		pruned := false
		filtered := index.Files[:0]
		for _, file := range index.Files {
			path := filepath.Join(dir, file.Name)
			info, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					pruned = true
					continue
				}
				return nil, err
			}
			filtered = append(filtered, file)

			finalizedAt := info.ModTime().UTC()
			if file.FinalizedAtUnixMilli > 0 {
				finalizedAt = time.UnixMilli(file.FinalizedAtUnixMilli).UTC()
			}
			files = append(files, FinalizedParquetFile{
				Table:       parts[0],
				Collection:  collectionFromDirName(parts[1]),
				Path:        path,
				RowCount:    file.RowCount,
				FinalizedAt: finalizedAt,
			})
		}
		if pruned {
			index.Files = filtered
			if err := writeIndexFile(dir, index); err != nil {
				return nil, err
			}
		}
	}
	return files, nil
}

func (s *Storage) indexedParquetFiles(table string) ([]string, error) {
	finalizedFiles, err := loadFinalizedParquetFiles(s.baseDir, table)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(finalizedFiles))
	for _, file := range finalizedFiles {
		files = append(files, file.Path)
	}
	return files, nil
}
