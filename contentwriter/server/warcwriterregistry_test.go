/*
 * Copyright 2021 National Library of Norway.
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

package server

import (
	"regexp"
	"testing"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/api/config"
	"github.com/nlnwa/gowarc"
	"github.com/stretchr/testify/assert"
)

func Test_timeToNextRotation(t *testing.T) {
	ts1 := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
	ts2 := time.Date(2021, 11, 20, 11, 29, 59, 0, time.Local)
	ts3 := time.Date(2021, 12, 31, 23, 59, 59, 0, time.Local)
	type args struct {
		now time.Time
		p   config.Collection_RotationPolicy
	}
	tests := []struct {
		name   string
		args   args
		want   time.Duration
		wantOk bool
	}{
		{"none", args{ts1, config.Collection_NONE}, 0, false},
		{"none", args{ts2, config.Collection_NONE}, 0, false},
		{"none", args{ts3, config.Collection_NONE}, 0, false},
		{"hourly", args{ts1, config.Collection_HOURLY}, time.Minute * 60, true},
		{"hourly", args{ts2, config.Collection_HOURLY}, time.Minute*30 + time.Second*1, true},
		{"hourly", args{ts3, config.Collection_HOURLY}, time.Second * 1, true},
		{"daily", args{ts1, config.Collection_DAILY}, time.Hour * 24, true},
		{"daily", args{ts2, config.Collection_DAILY}, time.Hour*12 + time.Minute*30 + time.Second*1, true},
		{"daily", args{ts3, config.Collection_DAILY}, time.Second * 1, true},
		{"monthly", args{ts1, config.Collection_MONTHLY}, time.Hour * 24 * 31, true},
		{"monthly", args{ts2, config.Collection_MONTHLY}, time.Hour*(24*10+12) + time.Minute*30 + time.Second*1, true},
		{"monthly", args{ts3, config.Collection_MONTHLY}, time.Second * 1, true},
		{"yearly", args{ts1, config.Collection_YEARLY}, time.Hour * 24 * 365, true},
		{"yearly", args{ts2, config.Collection_YEARLY}, time.Hour*(24*(10+31)+12) + time.Minute*30 + time.Second*1, true},
		{"yearly", args{ts3, config.Collection_YEARLY}, time.Second * 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := timeToNextRotation(tt.args.now, tt.args.p)
			if got != tt.want {
				t.Errorf("timeToNextRotation() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.wantOk {
				t.Errorf("timeToNextRotation() got1 = %v, want %v", got1, tt.wantOk)
			}
		})
	}
}

func Test_createFileRotationKey(t *testing.T) {
	ts1 := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
	ts2 := time.Date(2021, 11, 20, 11, 29, 59, 0, time.Local)
	ts3 := time.Date(2021, 12, 31, 23, 59, 59, 0, time.Local)
	type args struct {
		now time.Time
		p   config.Collection_RotationPolicy
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"none", args{ts1, config.Collection_NONE}, ""},
		{"none", args{ts2, config.Collection_NONE}, ""},
		{"none", args{ts3, config.Collection_NONE}, ""},
		{"hourly", args{ts1, config.Collection_HOURLY}, "2021010100"},
		{"hourly", args{ts2, config.Collection_HOURLY}, "2021112011"},
		{"hourly", args{ts3, config.Collection_HOURLY}, "2021123123"},
		{"daily", args{ts1, config.Collection_DAILY}, "20210101"},
		{"daily", args{ts2, config.Collection_DAILY}, "20211120"},
		{"daily", args{ts3, config.Collection_DAILY}, "20211231"},
		{"monthly", args{ts1, config.Collection_MONTHLY}, "202101"},
		{"monthly", args{ts2, config.Collection_MONTHLY}, "202111"},
		{"monthly", args{ts3, config.Collection_MONTHLY}, "202112"},
		{"yearly", args{ts1, config.Collection_YEARLY}, "2021"},
		{"yearly", args{ts2, config.Collection_YEARLY}, "2021"},
		{"yearly", args{ts3, config.Collection_YEARLY}, "2021"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createFileRotationKey(tt.args.now, tt.args.p); got != tt.want {
				t.Errorf("createFileRotationKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getFilename(t *testing.T) {
	ng := &gowarc.PatternNameGenerator{
		Directory: "",
		Prefix:    createFilePrefix("foo", config.Collection_UNDEFINED, time.Now(), config.Collection_NONE),
		Serial:    0,
	}
	d1, f1 := ng.NewWarcfileName()
	d2, f2 := ng.NewWarcfileName()
	assert.Regexp(t, regexp.MustCompile(`foo-\d{14}-0001-(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|.+)\.warc`), f1)
	assert.Equal(t, "", d1)
	assert.Regexp(t, regexp.MustCompile(`foo-\d{14}-0002-(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|.+)\.warc`), f2)
	assert.Equal(t, "", d2)

	ng = &gowarc.PatternNameGenerator{
		Directory: "",
		Prefix:    createFilePrefix("foo", config.Collection_UNDEFINED, time.Now(), config.Collection_YEARLY),
		Serial:    0,
	}
	d1, f1 = ng.NewWarcfileName()
	d2, f2 = ng.NewWarcfileName()
	assert.Regexp(t, regexp.MustCompile(`foo_\d{4}-\d{14}-0001-(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|.+)\.warc`), f1)
	assert.Equal(t, "", d1)
	assert.Regexp(t, regexp.MustCompile(`foo_\d{4}-\d{14}-0002-(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|.+)\.warc`), f2)
	assert.Equal(t, "", d2)

	ng = &gowarc.PatternNameGenerator{
		Directory: "myDir",
		Prefix:    createFilePrefix("foo", config.Collection_DNS, time.Now(), config.Collection_MONTHLY),
		Serial:    0,
	}
	d1, f1 = ng.NewWarcfileName()
	d2, f2 = ng.NewWarcfileName()
	assert.Regexp(t, regexp.MustCompile(`foo_DNS_\d{6}-\d{14}-0001-(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|.+)\.warc`), f1)
	assert.Equal(t, "myDir", d1)
	assert.Regexp(t, regexp.MustCompile(`foo_DNS_\d{6}-\d{14}-0002-(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|.+)\.warc`), f2)
	assert.Equal(t, "myDir", d2)

	ng = &gowarc.PatternNameGenerator{
		Directory: "myDir",
		Prefix:    createFilePrefix("foo", config.Collection_DNS, time.Now(), config.Collection_DAILY),
		Serial:    0,
	}
	d1, f1 = ng.NewWarcfileName()
	d2, f2 = ng.NewWarcfileName()
	assert.Regexp(t, regexp.MustCompile(`foo_DNS_\d{8}-\d{14}-0001-(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|.+)\.warc`), f1)
	assert.Equal(t, "myDir", d1)
	assert.Regexp(t, regexp.MustCompile(`foo_DNS_\d{8}-\d{14}-0002-(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|.+)\.warc`), f2)
	assert.Equal(t, "myDir", d2)
}
