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

package controller

import (
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/frontier"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/logwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/robotsevaluator"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
)

// browserControllerOptions configure the BrowserController. browserControllerOptions are set by the BrowserControllerOption
// values passed to New.
type browserControllerOptions struct {
	listenInterface string
	listenPort      int
	frontier        frontier.Frontier
	robotsEvaluator robotsevaluator.RobotsEvaluator
	logWriter       logwriter.LogWriter
	sessionOpts     []session.Option
	maxSessions     int
	closeTimeout    time.Duration
}

// BrowserControllerOption configures BrowserController.
type BrowserControllerOption interface {
	apply(*browserControllerOptions)
}

// funcBrowserControllerOption wraps a function that modifies browserControllerOptions into an
// implementation of the BrowserControllerOption interface.
type funcBrowserControllerOption struct {
	f func(*browserControllerOptions)
}

func (fco *funcBrowserControllerOption) apply(po *browserControllerOptions) {
	fco.f(po)
}

func newFuncBrowserControllerOption(f func(*browserControllerOptions)) *funcBrowserControllerOption {
	return &funcBrowserControllerOption{
		f: f,
	}
}

func defaultBrowserControllerOptions() browserControllerOptions {
	return browserControllerOptions{
		closeTimeout:    5 * time.Minute,
		maxSessions:     2,
		listenInterface: "",
		listenPort:      8080,
	}
}

func WithListenInterface(listenInterface string) BrowserControllerOption {
	return newFuncBrowserControllerOption(func(c *browserControllerOptions) {
		c.listenInterface = listenInterface
	})
}

func WithListenPort(port int) BrowserControllerOption {
	return newFuncBrowserControllerOption(func(c *browserControllerOptions) {
		c.listenPort = port
	})
}

func WithFrontier(frontier frontier.Frontier) BrowserControllerOption {
	return newFuncBrowserControllerOption(func(c *browserControllerOptions) {
		c.frontier = frontier
	})
}

func WithRobotsEvaluator(robotsevaluator robotsevaluator.RobotsEvaluator) BrowserControllerOption {
	return newFuncBrowserControllerOption(func(c *browserControllerOptions) {
		c.robotsEvaluator = robotsevaluator
	})
}

func WithLogWriter(logWriter logwriter.LogWriter) BrowserControllerOption {
	return newFuncBrowserControllerOption(func(c *browserControllerOptions) {
		c.logWriter = logWriter
	})
}

func WithSessionOptions(opts ...session.Option) BrowserControllerOption {
	return newFuncBrowserControllerOption(func(c *browserControllerOptions) {
		c.sessionOpts = opts
	})
}

func WithMaxConcurrentSessions(maxSessions int) BrowserControllerOption {
	return newFuncBrowserControllerOption(func(c *browserControllerOptions) {
		c.maxSessions = maxSessions
	})
}

func WithCloseTimeout(d time.Duration) BrowserControllerOption {
	return newFuncBrowserControllerOption(func(c *browserControllerOptions) {
		c.closeTimeout = d
	})
}
