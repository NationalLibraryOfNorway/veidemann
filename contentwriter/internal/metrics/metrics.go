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

package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "veidemann"
	subsystem = "contentwriter"
)

var uploadDuration = promauto.NewHistogram(prometheus.HistogramOpts{
	Namespace: namespace,
	Subsystem: subsystem,
	Name:      "upload_duration_seconds",
	Help:      "Duration of operations in seconds.",
	// 1s, 10s, 30s, 1m, 10m, 30m
	Buckets: []float64{1, 10, 30, 60, 600, 1800},
})

var writtenSize = promauto.NewHistogram(prometheus.HistogramOpts{
	Namespace: namespace,
	Subsystem: subsystem,
	Name:      "written_file_size_bytes",
	Help:      "Size of files as reported by the writer (bytes).",
	Buckets:   []float64{1e6, 1e8, 5e8, 1e9},
})

var onDiskSize = promauto.NewHistogram(prometheus.HistogramOpts{
	Namespace: namespace,
	Subsystem: subsystem,
	Name:      "on_disk_file_size_bytes",
	Help:      "Size of files on disk at upload time (bytes).",
	Buckets:   []float64{1e6, 1e8, 5e8, 1e9},
})

var uploadedSize = promauto.NewHistogram(prometheus.HistogramOpts{
	Namespace: namespace,
	Subsystem: subsystem,
	Name:      "uploaded_file_size_bytes",
	Help:      "Size of uploaded files as reported by the uploader (bytes).",
	Buckets:   []float64{1e6, 1e8, 5e8, 1e9},
})

var uploadSizeMismatch = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: namespace,
	Subsystem: subsystem,
	Name:      "upload_size_mismatch_total",
	Help:      "Count of uploads where uploaded size differs from on-disk size.",
})

var onDiskStatFailed = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: namespace,
	Subsystem: subsystem,
	Name:      "on_disk_stat_failed_total",
	Help:      "Count of failures to stat a file after upload.",
})

func WrittenSizeBytes(n int64)       { writtenSize.Observe(float64(n)) }
func OnDiskSizeBytes(n int64)        { onDiskSize.Observe(float64(n)) }
func UploadedSizeBytes(n int64)      { uploadedSize.Observe(float64(n)) }
func UploadDuration(d time.Duration) { uploadDuration.Observe(d.Seconds()) }
func UploadSizeMismatch()            { uploadSizeMismatch.Inc() }
func OnDiskStatFailed()              { onDiskStatFailed.Inc() }
