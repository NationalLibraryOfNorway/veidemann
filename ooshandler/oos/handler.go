/*
 * Copyright 2019 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package oos

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/kennygrant/sanitize"
	"golang.org/x/sync/singleflight"
)

const (
	maxElements = 500000
	probCollide = 0.00000001
)

type Handler struct {
	bf          *bloom.BloomFilter
	dir         string
	bloomSF     singleflight.Group
	fmu         sync.Mutex // protects fileMutexes
	fileMutexes map[string]*sync.Mutex
}

func NewHandler(dir string) (*Handler, error) {
	bf := bloom.NewWithEstimates(maxElements, probCollide)
	h := &Handler{
		bf:          bf,
		dir:         dir,
		fileMutexes: make(map[string]*sync.Mutex),
	}
	return h, h.ImportExisting()
}

func (o *Handler) ImportExisting() error {
	files, err := os.ReadDir(o.dir)
	if err != nil {
		return err
	}

	i := 0
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		file := filepath.Join(o.dir, f.Name())
		n, err := o.parseFile(file)
		if err != nil {
			slog.Error("Error importing existing URIs", "file", file, "err", err)
			continue
		}
		i += n
	}
	slog.Info("Imported existing URIs", "count", i)
	return nil
}

func (o *Handler) parseFile(file string) (uris int, err error) {
	f, err := os.Open(file)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	count := 0

	buf := bufio.NewReader(f)
	for {
		l, err := buf.ReadString('\n')
		if err == io.EOF {
			return count, nil
		}
		if err != nil {
			return count, err
		}

		line := strings.Trim(l, "\n")
		if line == "" {
			continue
		}
		u, _, err := o.parseUriAndGroup(line)
		if err != nil {
			slog.Warn("Failed to parse URI  and group", "err", err, "line", line)
			continue
		}
		o.bloomContains(u)
		count++
	}
}

func (o *Handler) Handle(uri string) (exists bool) {
	u, g, err := o.parseUriAndGroup(uri)
	if err != nil {
		slog.Warn("Error parsing uri", "err", err)
		return false
	}

	v, _, _ := o.bloomSF.Do(u.Host, func() (interface{}, error) {
		exists = o.bloomContains(u)
		if exists {
			exists = o.isInFile(u, g)
		} else {
			o.write(u, g)
		}
		return exists, nil
	})

	return v.(bool)
}

func (o *Handler) parseUriAndGroup(uri string) (*url.URL, string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, "", fmt.Errorf("error parsing uri '%v': %v", uri, err)
	}

	parts := strings.Split(sanitize.Name(u.Hostname()), ".")
	var g string
	if len(parts) >= 2 {
		// Get last part of host and maximum two chars from the next to last part of host
		// www.example.com => com_ex
		// 127.0.0.1       => 1_0
		l := Min(2, len(parts[len(parts)-2]))
		g = parts[len(parts)-1:][0] + "_" + parts[len(parts)-2:][0][:l]
	} else {
		g = u.Host
	}
	return u, g, nil
}

func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func (o *Handler) bloomContains(uri *url.URL) bool {
	return o.bf.TestOrAddString(uri.Host)
}

func (o *Handler) write(uri *url.URL, group string) {
	file := o.createFileName(group)
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		slog.Error("Error opening file for writing", "file", file, "err", err)
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = fmt.Fprintf(f, "%s://%s\n", uri.Scheme, uri.Host)
}

func (o *Handler) isInFile(uri *url.URL, group string) bool {
	c := o.getFileLock(group)
	defer c.Unlock()

	file := o.createFileName(group)
	f, err := os.OpenFile(file, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		slog.Warn("Error opening file", "file", file, "err", err)
		return false
	}
	defer func() { _ = f.Close() }()

	var exists bool
	buf := bufio.NewReader(f)
	for {
		l, err := buf.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				slog.Warn("Error reading next line from file", "file", file, "err", err)
			}
			break
		}
		l = strings.Trim(l, "\n")
		if l == fmt.Sprintf("%s://%s", uri.Scheme, uri.Host) {
			exists = true
			break
		}
	}

	if !exists {
		_, err := f.Seek(0, 2)
		if err != nil {
			slog.Error("Error seeking to end of file", "file", file, "err", err)
		} else {
			_, _ = fmt.Fprintf(f, "%s://%s\n", uri.Scheme, uri.Host)
		}
	}

	return exists
}

func (o *Handler) createFileName(group string) string {
	return path.Join(o.dir, "uri_"+group+".txt")
}

func (o *Handler) getFileLock(key string) *sync.Mutex {
	o.fmu.Lock()
	defer o.fmu.Unlock()
	var c *sync.Mutex
	var ok bool
	if c, ok = o.fileMutexes[key]; ok {
		c.Lock()
	} else {
		c = new(sync.Mutex)
		c.Lock()
		o.fileMutexes[key] = c
	}

	return c
}
