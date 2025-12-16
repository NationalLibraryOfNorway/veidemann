/*
 * Copyright 2023 National Library of Norway.
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

package fai

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type options struct {
	sourceDir   string
	sleep       time.Duration
	globPattern string
	inspect     func(context.Context, string) error
}

type option func(opts *options)

func WithSourceDir(dir string) option {
	return func(opts *options) {
		opts.sourceDir = dir
	}
}

func WithSleep(sleep time.Duration) option {
	return func(opts *options) {
		opts.sleep = sleep
	}
}

func WithGlobPattern(globPattern string) option {
	return func(opts *options) {
		opts.globPattern = globPattern
	}
}

func WithInspector(inspector func(context.Context, string) error) option {
	return func(opts *options) {
		opts.inspect = inspector
	}
}

func New(opts ...option) (*options, error) {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	var err error

	// assert source directory exists
	o.sourceDir, err = filepath.Abs(o.sourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path of source directory: %w", err)
	}
	// assert source
	if info, err := os.Stat(o.sourceDir); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("source directory does not exist: %w", err)
	} else if err != nil {
		return nil, fmt.Errorf("failed to stat source directory: %w", err)
	} else if !info.IsDir() {
		return nil, errors.New("source is not a directory")
	}

	o.globPattern = filepath.Join(o.sourceDir, o.globPattern)

	// assert glob pattern is valid
	_, err = filepath.Glob(o.globPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %w", err)
	}

	return o, nil
}

// Run starts the first article inspection.
// Once every sleep duration the inspect function is run on all files in the
// source source directory matching the glob pattern.
// It will run until the context is done or after one pass if the sleep
// duration is configured as zero.
func (f *options) Run(ctx context.Context) error {
	for {
		files, _ := filepath.Glob(f.globPattern)
		for _, file := range files {
			err := f.inspect(ctx, file)
			if err != nil {
				return err
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(f.sleep):
			// if sleep duration is zero it is only one pass
			if f.sleep == 0 {
				return nil
			}
		}
	}
}
