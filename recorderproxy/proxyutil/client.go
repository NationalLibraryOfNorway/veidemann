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

package main

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/recorderproxy"
)

func get(ctx context.Context, url, attemptID string, client *http.Client, timeout time.Duration) (int, []byte, error) {
	log := logger.LogWithPhase("http_client")

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set(constants.HeaderAttemptId, attemptID)

	if timeout > 0 {
		ctx, cancel := context.WithTimeout(req.Context(), timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	if logger.IsLevelEnabled(logger.DebugLevel) {
		tr, r := recorderproxy.DecorateRequest(client.Transport, req)
		client = &http.Client{Transport: tr}
		req = r
	}

	log.Infof("submitting request: %v %v %v", req.Method, req.URL, req.Proto)
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	txt, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, txt, nil
}
