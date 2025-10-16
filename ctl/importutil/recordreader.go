// Copyright © 2019 National Library of Norway
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package importutil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
)

type State struct {
	fileName string
	recNum   int
	err      error
}

func (r *State) GetFilename() string { return r.fileName }
func (r *State) GetRecordNum() int   { return r.recNum }
func (r *State) GetError() error     { return r.err }

type recordReader struct {
	recordDecoder RecordDecoder
	curFile       *os.File
	dir           *os.File
	curFileName   string
	curRecNum     int
	filePattern   string
}

type RecordDecoder interface {
	Init(r io.Reader, suffix string)
	Read(v interface{}) (err error)
}

func NewRecordReader(fileOrDir string, decoder RecordDecoder, filePattern string) (l *recordReader, err error) {
	l = &recordReader{
		recordDecoder: decoder,
		filePattern:   filePattern,
	}

	if fileOrDir == "-" {
		l.recordDecoder.Init(os.Stdin, "")
	} else {
		if hasMeta(fileOrDir) {
			l.filePattern = filepath.Base(fileOrDir)
			fileOrDir = filepath.Dir(fileOrDir)
		}
		var f *os.File
		f, err = os.Open(fileOrDir)
		if err != nil {
			return nil, fmt.Errorf("could not open file '%s': %w", fileOrDir, err)
		}
		if fi, _ := f.Stat(); fi.IsDir() {
			l.dir = f
			err = l.initRecordReader()
			if err != nil {
				return nil, fmt.Errorf("could not open file '%s': %w", fileOrDir, err)
			}
		} else {
			l.curFileName = f.Name()
			log.Info().Str("filename", l.curFileName).Msg("Reading file")
			l.recordDecoder.Init(f, filepath.Ext(f.Name()))
		}
	}
	return
}

// hasMeta reports whether path contains any of the magic characters
// recognized by filepath.Match.
func hasMeta(path string) bool {
	magicChars := `*?[`
	if strings.HasPrefix(runtime.GOOS, "windows") {
		magicChars = `*?[\`
	}
	return strings.ContainsAny(path, magicChars)
}

func (l *recordReader) initRecordReader() error {
	if l.curFile != nil {
		_ = l.curFile.Close()
	}

	if l.dir == nil {
		return io.EOF
	}

	var f []os.FileInfo
	var err error
	for {
		f, err = l.dir.Readdir(1)
		if err != nil {
			return err
		}
		fi := f[0]

		if !fi.IsDir() {
			var match bool
			if match, err = filepath.Match(l.filePattern, fi.Name()); match && err == nil {
				l.curFile, err = os.Open(filepath.Join(l.dir.Name(), fi.Name()))
				if err != nil {
					return fmt.Errorf("failed to open file \"%s\": %w", fi.Name(), err)
				}
				l.curRecNum = 0
				l.curFileName = l.curFile.Name()

				log.Info().Str("filename", l.curFileName).Msg("Reading file")
				l.recordDecoder.Init(l.curFile, filepath.Ext(l.curFile.Name()))
				break
			}
		}
	}
	return nil
}

func (l *recordReader) Next(v interface{}) (*State, error) {
	err := l.recordDecoder.Read(v)
	if errors.Is(err, io.EOF) {
		if err = l.initRecordReader(); err != nil {
			return nil, err
		}
		return l.Next(v)
	}
	l.curRecNum++

	return &State{
		fileName: l.curFileName,
		recNum:   l.curRecNum,
		err:      err,
	}, nil
}
