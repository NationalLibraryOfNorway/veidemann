package parquet

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

const indexFileName = ".index.json"

type indexFile struct {
	Files []indexEntry `json:"files"`
}

type indexEntry struct {
	Name     string `json:"name"`
	RowCount int64  `json:"rowCount"`
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

func (s *Storage) indexedParquetFiles(table string) ([]string, error) {
	tableDir := filepath.Join(s.baseDir, table)
	if _, err := os.Stat(tableDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	indexPaths := make([]string, 0)
	err := filepath.WalkDir(tableDir, func(path string, entry fs.DirEntry, err error) error {
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

	files := make([]string, 0)
	for _, indexPath := range indexPaths {
		dir := filepath.Dir(indexPath)
		index, err := readIndexFile(dir)
		if err != nil {
			return nil, err
		}
		pruned := false
		filtered := index.Files[:0]
		for _, file := range index.Files {
			path := filepath.Join(dir, file.Name)
			if _, err := os.Stat(path); err != nil {
				if os.IsNotExist(err) {
					pruned = true
					continue
				}
				return nil, err
			}
			filtered = append(filtered, file)
			files = append(files, path)
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
