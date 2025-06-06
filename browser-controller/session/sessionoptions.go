/*
 * Copyright 2020 National Library of Norway.
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

package session

import (
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/database"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/logwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/screenshotwriter"
)

// Option configures session Registry.
type Option interface {
	apply(*Session)
}

// funcOption wraps a function that modifies Option into an
// implementation of the ParserOption interface.
type funcOption struct {
	f func(*Session)
}

func (fpo *funcOption) apply(po *Session) {
	fpo.f(po)
}

func newFuncOption(f func(*Session)) *funcOption {
	return &funcOption{
		f: f,
	}
}

func WithBrowserHost(host string) Option {
	return newFuncOption(func(s *Session) {
		s.browserHost = host
	})
}

func WithBrowserPort(port int) Option {
	return newFuncOption(func(s *Session) {
		s.browserPort = port
	})
}

func WithProxyHost(host string) Option {
	return newFuncOption(func(s *Session) {
		s.proxyHost = host
	})
}

func WithProxyPort(port int) Option {
	return newFuncOption(func(s *Session) {
		s.proxyPort = port
	})
}

func WithConfigCache(dbAdapter database.ConfigCache) Option {
	return newFuncOption(func(s *Session) {
		s.configCache = dbAdapter
	})
}

func WithScreenshotWriter(sw screenshotwriter.ScreenshotWriter) Option {
	return newFuncOption(func(s *Session) {
		s.screenShotWriter = sw
	})
}

func WithLogWriter(lc logwriter.LogWriter) Option {
	return newFuncOption(func(s *Session) {
		s.logWriter = lc
	})
}
