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

const namespace = "fai"

var duration = promauto.NewHistogram(prometheus.HistogramOpts{
	Namespace: namespace,
	Name:      "duration_seconds",
	Help:      "Duration of operations in seconds.",
	// 1s, 10s, 30s, 1m, 10m, 30m
	Buckets: []float64{1, 10, 30, 60, 600, 1800},
})

var filesize = promauto.NewHistogram(prometheus.HistogramOpts{
	Namespace: namespace,
	Name:      "file_size_bytes",
	Help:      "Size of files in bytes.",
	// 1MB, 100MB, 500MB, 1GB
	Buckets: []float64{1000000, 100000000, 500000000, 1000000000},
})

// Size records the size of the given file.
func Size(size int64) {
	filesize.Observe(float64(size))
}

func Duration(d time.Duration) {
	duration.Observe(float64(d))
}
