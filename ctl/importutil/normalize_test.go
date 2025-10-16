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

package importutil

import (
	"testing"
)

func Test_importer_normalizeUri(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantUri string
		wantErr bool
	}{
		{"1", "http://www.example.com", "http://www.example.com/", false},
		{"2", "http://www.example.com#hash", "http://www.example.com/", false},
		{"3", "http://www.example.com?query", "http://www.example.com/?query", false},
		{"4", "http://www.example.com?query#hash", "http://www.example.com/?query", false},
		{"5", "http://www.example.com/", "http://www.example.com/", false},
		{"6", "http://www.example.com/#hash", "http://www.example.com/", false},
		{"7", "http://www.example.com/?query", "http://www.example.com/?query", false},
		{"8", "http://www.example.com/?query#hash", "http://www.example.com/?query", false},
		{"9", "http://www.example.com/foo/bar", "http://www.example.com/foo/bar", false},
		{"10", "http://www.example.com/foo/bar#hash", "http://www.example.com/foo/bar", false},
		{"11", "http://www.example.com/foo/bar?query", "http://www.example.com/foo/bar?query", false},
		{"12", "http://www.example.com/foo/bar?query#hash", "http://www.example.com/foo/bar?query", false},
		{"13", "https://www.example.com/foo/bar#hash", "https://www.example.com/foo/bar", false},
		{"14", "HTTPS://www.example.com/foo/bar#hash", "https://www.example.com/foo/bar", false},
		{"15", "https://www.Example.Com/foo/bar#hash", "https://www.example.com/foo/bar", false},
		{"16", "http://www.example.com/foo/", "http://www.example.com/foo/", false},
		{"17", "http://www.example.com/foo", "http://www.example.com/foo", false},
		{"18", "https://www.example.com/foo/", "https://www.example.com/foo/", false},
		{"19", "https://www.example.com/foo", "https://www.example.com/foo", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &UriKeyNormalizer{}
			got, err := n.Normalize(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("normalizeUri() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.wantUri {
				t.Errorf("normalizeUri() uri = %v, want %v", got, tt.wantUri)
			}
		})
	}
}

func Test_UriKeyNormalizer_toplevelFlag(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantKey string
		wantErr bool
	}{
		{"1", "http://www.example.com", "http://www.example.com/", false},
		{"2", "http://www.example.com#hash", "http://www.example.com/", false},
		{"3", "http://www.example.com?query", "http://www.example.com/", false},
		{"4", "http://www.example.com?query#hash", "http://www.example.com/", false},
		{"5", "http://www.example.com/", "http://www.example.com/", false},
		{"6", "http://www.example.com/#hash", "http://www.example.com/", false},
		{"7", "http://www.example.com/?query", "http://www.example.com/", false},
		{"8", "http://www.example.com/?query#hash", "http://www.example.com/", false},
		{"9", "http://www.example.com/foo/bar", "http://www.example.com/", false},
		{"10", "http://www.example.com/foo/bar#hash", "http://www.example.com/", false},
		{"11", "http://www.example.com/foo/bar?query", "http://www.example.com/", false},
		{"12", "http://www.example.com/foo/bar?query#hash", "http://www.example.com/", false},
		{"13", "https://www.example.com/foo/bar#hash", "https://www.example.com/", false},
		{"14", "HTTPS://www.example.com/foo/bar#hash", "https://www.example.com/", false},
		{"15", "https://www.Example.Com/foo/bar#hash", "https://www.example.com/", false},
		{"16", "http://www.example.com/foo/", "http://www.example.com/", false},
		{"17", "http://www.example.com/foo", "http://www.example.com/", false},
		{"18", "https://www.example.com/foo/", "https://www.example.com/", false},
		{"19", "https://www.example.com/foo", "https://www.example.com/", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &UriKeyNormalizer{Toplevel: true}
			got, err := n.Normalize(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("Normalize() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.wantKey {
				t.Errorf("Normalize() key = %v, want %v", got, tt.wantKey)
			}
		})
	}
}

func Test_UriKeyNormalizer_ignoreSchemeFlag(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantKey string
		wantErr bool
	}{
		{"1", "http://www.example.com", "//www.example.com/", false},
		{"2", "http://www.example.com#hash", "//www.example.com/", false},
		{"3", "http://www.example.com?query", "//www.example.com/?query", false},
		{"4", "http://www.example.com?query#hash", "//www.example.com/?query", false},
		{"5", "http://www.example.com/", "//www.example.com/", false},
		{"6", "http://www.example.com/#hash", "//www.example.com/", false},
		{"7", "http://www.example.com/?query", "//www.example.com/?query", false},
		{"8", "http://www.example.com/?query#hash", "//www.example.com/?query", false},
		{"9", "http://www.example.com/foo/bar", "//www.example.com/foo/bar", false},
		{"10", "http://www.example.com/foo/bar#hash", "//www.example.com/foo/bar", false},
		{"11", "http://www.example.com/foo/bar?query", "//www.example.com/foo/bar?query", false},
		{"12", "http://www.example.com/foo/bar?query#hash", "//www.example.com/foo/bar?query", false},
		{"13", "https://www.example.com/foo/bar#hash", "//www.example.com/foo/bar", false},
		{"14", "HTTPS://www.example.com/foo/bar#hash", "//www.example.com/foo/bar", false},
		{"15", "https://www.Example.Com/foo/bar#hash", "//www.example.com/foo/bar", false},
		{"16", "http://www.example.com/foo/", "//www.example.com/foo/", false},
		{"17", "http://www.example.com/foo", "//www.example.com/foo", false},
		{"18", "https://www.example.com/foo/", "//www.example.com/foo/", false},
		{"19", "https://www.example.com/foo", "//www.example.com/foo", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &UriKeyNormalizer{IgnoreScheme: true}
			got, err := i.Normalize(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("Normalize() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.wantKey {
				t.Errorf("Normalize() key = %v, want %v", got, tt.wantKey)
			}
		})
	}
}
