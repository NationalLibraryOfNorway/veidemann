// Copyright 2026 National Library of Norway.
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"testing"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
)

func TestRobotsUserAgent(t *testing.T) {
	for _, configured := range []string{"veidemann/development", ""} {
		sess := &Session{
			browserConfig: &configV1.BrowserConfig{UserAgent: configured},
			UserAgent:     "Mozilla/5.0 Chrome/test " + configured,
		}
		if got := sess.RobotsUserAgent(); got != configured {
			t.Errorf("RobotsUserAgent() = %q, want %q", got, configured)
		}
	}
}
