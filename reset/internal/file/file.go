package file

import (
	"log"
	"os"
	"path"
)

type DirectoryCleaner struct {
	Directories []string
}

func NewDirectoryCleaner(directories []string) DirectoryCleaner {
	return DirectoryCleaner{
		Directories: directories,
	}
}

func (f DirectoryCleaner) RemoveFiles() {
	for _, directory := range f.Directories {
		files, err := os.ReadDir(directory)
		if err != nil {
			log.Printf("Failed to read directory: %s\n", err.Error())
			continue
		}
		log.Printf("Removing all files from directory: %s\n", directory)

		for _, file := range files {
			err := os.Remove(path.Join(directory, file.Name()))
			if err != nil {
				log.Printf("Failed to delete file: %s %s\n", file.Name(), err.Error())
				continue
			}
		}
	}
}
