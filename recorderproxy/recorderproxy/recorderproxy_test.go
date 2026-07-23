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

package recorderproxy_test

import (
	"context"
	"crypto/tls"
	stderrors "errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	browsercontrollerV2 "github.com/NationalLibraryOfNorway/veidemann/api/browsercontroller/v2"
	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	dnsresolverV1 "github.com/NationalLibraryOfNorway/veidemann/api/dnsresolver/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/recorderproxy"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/serviceconnections"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
	"github.com/go-test/deep"
	"google.golang.org/protobuf/proto"
)

var (
	acceptAllCerts = &tls.Config{InsecureSkipVerify: true}
)

type clientWant struct {
	status     int
	bodyPrefix string
	err        bool
}

type recordWant struct {
	statusCode          int
	errorCode           errors.ErrorCode
	errorMsg            string
	errorDetailContains string
	responseBlockDigest *bool
	responseBlockSize   *int
}

type test struct {
	name          string
	url           string
	clientTimeout time.Duration
	keepAlive     bool

	wantClient clientWant
	wantRecord recordWant

	wantGrpcRequests *testutil.Requests
}

func init() {
	logger.InitLog("info", "text", false)
}

func boolPtr(v bool) *bool {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func TestRecorderProxy(t *testing.T) {
	s := testutil.NewHttpServers(t)
	defer s.Close()
	grpcServices := testutil.NewGrpcServiceMock()
	defer grpcServices.Close()
	client, recorderProxy, err := localRecorderProxy(t, grpcServices.ClientConn, "")
	if err != nil {
		t.Fatalf("Failed to initialize local recorder proxy: %v", err)
	}
	defer client.CloseIdleConnections()
	defer recorderProxy.Shutdown(context.TODO())
	tests := []test{
		{
			name: "http:success",
			url:  s.SrvHttp.URL + "/a",
			wantClient: clientWant{
				status:     http.StatusOK,
				bodyPrefix: "content from http server",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:          http.StatusOK,
				responseBlockDigest: boolPtr(true),
				responseBlockSize:   intPtr(141),
			},
		},
		{
			name: "https:success",
			url:  s.SrvHttps.URL + "/b",
			wantClient: clientWant{
				status:     http.StatusOK,
				bodyPrefix: "content from https server",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:          http.StatusOK,
				responseBlockDigest: boolPtr(true),
				responseBlockSize:   intPtr(142),
			},
		},
		{
			name: "http:client timeout",
			url:  s.SrvHttp.URL + "/slow",
			wantClient: clientWant{
				status:     0,
				bodyPrefix: "",
				err:        true,
			},
			clientTimeout: 500 * time.Millisecond,
		},
		{
			name: "https:client timeout",
			url:  s.SrvHttps.URL + "/slow",
			wantClient: clientWant{
				status:     0,
				bodyPrefix: "",
				err:        true,
			},
			clientTimeout: 500 * time.Millisecond,
		},
		{
			name: "http:not found",
			url:  s.SrvHttp.URL + "/c",
			wantClient: clientWant{
				status:     http.StatusNotFound,
				bodyPrefix: "404 page not found\n",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:          http.StatusNotFound,
				responseBlockDigest: boolPtr(true),
				responseBlockSize:   intPtr(176),
			},
		},
		{
			name: "https:not found",
			url:  s.SrvHttps.URL + "/c",
			wantClient: clientWant{
				status:     http.StatusNotFound,
				bodyPrefix: "404 page not found\n",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:          http.StatusNotFound,
				responseBlockDigest: boolPtr(true),
				responseBlockSize:   intPtr(176),
			},
		},
		{
			name: "http:server timeout",
			url:  s.SrvHttp.URL + "/extraslow",
			wantClient: clientWant{
				status:     http.StatusServiceUnavailable,
				bodyPrefix: "Code: -404, Msg: EMPTY_RESPONSE, Detail: Empty reply from server",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:          http.StatusServiceUnavailable,
				errorCode:           errors.EmptyResponse,
				errorMsg:            "EMPTY_RESPONSE",
				errorDetailContains: "Empty reply from server",
			},
		},
		{
			name: "https:server timeout",
			url:  s.SrvHttps.URL + "/extraslow",
			wantClient: clientWant{
				status:     http.StatusServiceUnavailable,
				bodyPrefix: "Code: -5, Msg: UNKNOWN_ERROR, Detail: tls: bad record MAC",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:          http.StatusServiceUnavailable,
				errorCode:           errors.RuntimeException,
				errorMsg:            "UNKNOWN_ERROR",
				errorDetailContains: "tls: bad record MAC",
			},
		},
		{
			name: "http:browser controller cancel",
			url:  s.SrvHttp.URL + "/cancel",
			wantClient: clientWant{
				status:     http.StatusServiceUnavailable,
				bodyPrefix: "Code: -5011, Msg: CANCELED_BY_BROWSER, Detail: Cancelled by browser controller",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:          http.StatusServiceUnavailable,
				errorCode:           errors.CanceledByBrowser,
				errorMsg:            "CANCELED_BY_BROWSER",
				errorDetailContains: "Cancelled by browser controller",
			},
		},
		{
			name: "https:browser controller cancel",
			url:  s.SrvHttps.URL + "/cancel",
			wantClient: clientWant{
				status:     http.StatusServiceUnavailable,
				bodyPrefix: "Code: -5011, Msg: CANCELED_BY_BROWSER, Detail: Cancelled by browser controller",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:          http.StatusServiceUnavailable,
				errorCode:           errors.CanceledByBrowser,
				errorMsg:            "CANCELED_BY_BROWSER",
				errorDetailContains: "Cancelled by browser controller",
			},
		},
		{
			name: "http:blocked by robots.txt",
			url:  s.SrvHttp.URL + "/blocked",
			wantClient: clientWant{
				status:     http.StatusServiceUnavailable,
				bodyPrefix: "Code: -9998, Msg: PRECLUDED_BY_ROBOTS, Detail: Robots.txt rules precluded fetch",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:          http.StatusServiceUnavailable,
				errorCode:           errors.PrecludedByRobots,
				errorMsg:            "PRECLUDED_BY_ROBOTS",
				errorDetailContains: "Robots.txt rules precluded fetch",
			},
		},
		{
			name: "https:blocked by robots.txt",
			url:  s.SrvHttps.URL + "/blocked",
			wantClient: clientWant{
				status:     http.StatusServiceUnavailable,
				bodyPrefix: "Code: -9998, Msg: PRECLUDED_BY_ROBOTS, Detail: Robots.txt rules precluded fetch",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:          http.StatusServiceUnavailable,
				errorCode:           errors.PrecludedByRobots,
				errorMsg:            "PRECLUDED_BY_ROBOTS",
				errorDetailContains: "Robots.txt rules precluded fetch",
			},
		},
		{
			name: "http:browser controller error",
			url:  s.SrvHttp.URL + "/bccerr",
			wantClient: clientWant{
				status:     http.StatusOK,
				bodyPrefix: "content from http server",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:          http.StatusOK,
				responseBlockDigest: boolPtr(true),
				responseBlockSize:   intPtr(141),
			},
		},
		{
			name: "https:browser controller error",
			url:  s.SrvHttps.URL + "/bccerr",
			wantClient: clientWant{
				status:     http.StatusOK,
				bodyPrefix: "content from https server",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:          http.StatusOK,
				responseBlockDigest: boolPtr(true),
				responseBlockSize:   intPtr(142),
			},
		},
		{
			name: "http:content writer error",
			url:  s.SrvHttp.URL + "/cwerr",
			wantClient: clientWant{
				status:     http.StatusOK,
				bodyPrefix: "content from http server",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:        http.StatusOK,
				responseBlockSize: intPtr(141),
			},
		},
		{
			name: "https:content writer error",
			url:  s.SrvHttps.URL + "/cwerr",
			wantClient: clientWant{
				status:     http.StatusOK,
				bodyPrefix: "content from https server",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:        http.StatusOK,
				responseBlockSize: intPtr(142),
			},
		},
		{
			name: "http:cached",
			url:  s.SrvHttp.URL + "/cached",
			wantClient: clientWant{
				status:     http.StatusOK,
				bodyPrefix: "content from http server",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:        http.StatusOK,
				responseBlockSize: intPtr(243),
			},
		},
		{
			name: "https:cached",
			url:  s.SrvHttps.URL + "/cached",
			wantClient: clientWant{
				status:     http.StatusOK,
				bodyPrefix: "content from https server",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:        http.StatusOK,
				responseBlockSize: intPtr(244),
			},
		},
		{
			name: "http:no host",
			url:  s.SrvHttp.URL[:len(s.SrvHttp.URL)-2] + "1/no_host",
			wantClient: clientWant{
				status:     http.StatusServiceUnavailable,
				bodyPrefix: "Code: -2, Msg: CONNECT_FAILED, Detail: connection refused",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:          http.StatusServiceUnavailable,
				errorCode:           errors.ConnectFailed,
				errorMsg:            "CONNECT_FAILED",
				errorDetailContains: "connection refused",
				responseBlockSize:   intPtr(138),
			},
		},
		{
			name: "https:no host",
			url:  s.SrvHttps.URL[:len(s.SrvHttps.URL)-2] + "1/no_host",
			wantClient: clientWant{
				status:     http.StatusServiceUnavailable,
				bodyPrefix: "Code: -2, Msg: CONNECT_FAILED, Detail: connection refused",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:          http.StatusServiceUnavailable,
				errorCode:           errors.ConnectFailed,
				errorMsg:            "CONNECT_FAILED",
				errorDetailContains: "connection refused",
				responseBlockSize:   intPtr(138),
			},
		},
		{
			name: "https:upstream tls handshake failure",
			url:  s.SrvHttpsBrokenTLS.URL + "/b",
			wantClient: clientWant{
				status:     http.StatusServiceUnavailable,
				bodyPrefix: "Code: -2, Msg: CONNECT_FAILED, Detail: tls: handshake failure",
				err:        false,
			},
			wantRecord: recordWant{
				statusCode:          http.StatusServiceUnavailable,
				errorCode:           errors.ConnectFailed,
				errorMsg:            "CONNECT_FAILED",
				errorDetailContains: "tls: handshake failure",
				responseBlockDigest: boolPtr(false),
				responseBlockSize:   intPtr(144),
			},
		},
	}

	for i, tt := range tests {
		tt.keepAlive = true

		tt.generateExpectedRequests()
		grpcServices.Clear()

		t.Run(strconv.Itoa(i)+": "+tt.name, func(t *testing.T) {
			statusCode, got, err := get(tt.url, client, tt.clientTimeout)
			t.Logf("GET url=%v status=%v got=%v error=%v", tt.url, statusCode, string(got), err)

			if grpcServices.DoneBC != nil {
				<-grpcServices.DoneBC
			}
			if grpcServices.DoneCW != nil {
				<-grpcServices.DoneCW
			}

			if (err != nil) != tt.wantClient.err {
				t.Errorf(
					"Client get() error = %v, wantErr %v (%v, %s)",
					err,
					tt.wantClient.err,
					statusCode,
					got,
				)
				return
			}

			if statusCode != tt.wantClient.status {
				t.Errorf("Expected status code: %d, got %d", tt.wantClient.status, statusCode)
				return
			}

			if !strings.HasPrefix(string(got), tt.wantClient.bodyPrefix) {
				t.Errorf("Expected '%s' to start with '%s'", got, tt.wantClient.bodyPrefix)
				return
			}

			compareDNS(
				t,
				"DnsResolver",
				tt.wantGrpcRequests.DnsResolverRequests,
				grpcServices.Requests.DnsResolverRequests,
			)

			compareBC(
				t,
				"BrowserController",
				tt,
				tt.wantGrpcRequests.BrowserControllerRequests,
				grpcServices.Requests.BrowserControllerRequests,
			)

			compareCW(
				t,
				"ContentWriter",
				tt.wantGrpcRequests.ContentWriterRequests,
				grpcServices.Requests.ContentWriterRequests,
			)
		})
	}
}

func TestRecorderProxyThroughProxy(t *testing.T) {
	s := testutil.NewHttpServers(t)
	defer s.Close()

	grpcServices := testutil.NewGrpcServiceMock()
	defer grpcServices.Close()

	nextProxy, nextProxyAddr := testutil.NewSecondaryProxy(t, s)
	defer nextProxy.Close()

	client, recorderProxy, err := localRecorderProxy(t, grpcServices.ClientConn, nextProxyAddr)
	if err != nil {
		t.Fatalf("Failed to initialize local recorder proxy: %v", err)
	}
	defer client.CloseIdleConnections()
	defer recorderProxy.Shutdown(context.TODO())

	success := func(name, rawURL string, status int, body string, blockSize int) test {
		return test{
			name: name,
			url:  rawURL,
			wantClient: clientWant{
				status:     status,
				bodyPrefix: body,
			},
			wantRecord: recordWant{
				statusCode:          status,
				responseBlockDigest: boolPtr(true),
				responseBlockSize:   intPtr(blockSize),
			},
		}
	}
	errorResult := func(
		name, rawURL, body string,
		code errors.ErrorCode,
		message, detail string,
	) test {
		return test{
			name: name,
			url:  rawURL,
			wantClient: clientWant{
				status:     http.StatusServiceUnavailable,
				bodyPrefix: body,
			},
			wantRecord: recordWant{
				statusCode:          http.StatusServiceUnavailable,
				errorCode:           code,
				errorMsg:            message,
				errorDetailContains: detail,
			},
		}
	}

	tests := []test{
		success("http:success", s.SrvHttp.URL+"/a", http.StatusOK, "content from http server", 141),
		success("https:success", s.SrvHttps.URL+"/b", http.StatusOK, "content from https server", 142),
		{
			name: "http:client timeout",
			url:  s.SrvHttp.URL + "/slow",
			wantClient: clientWant{
				err: true,
			},
			clientTimeout: 500 * time.Millisecond,
		},
		{
			name: "https:client timeout",
			url:  s.SrvHttps.URL + "/slow",
			wantClient: clientWant{
				err: true,
			},
			clientTimeout: 500 * time.Millisecond,
		},
		success("http:not found", s.SrvHttp.URL+"/c", http.StatusNotFound, "404 page not found\n", 176),
		success("https:not found", s.SrvHttps.URL+"/c", http.StatusNotFound, "404 page not found\n", 176),
		errorResult(
			"http:server timeout",
			s.SrvHttp.URL+"/extraslow",
			"Code: -404, Msg: EMPTY_RESPONSE, Detail: Empty reply from server",
			errors.EmptyResponse,
			"EMPTY_RESPONSE",
			"Empty reply from server",
		),
		errorResult(
			"https:server timeout",
			s.SrvHttps.URL+"/extraslow",
			"Code: -404, Msg: EMPTY_RESPONSE, Detail: Empty reply from server",
			errors.EmptyResponse,
			"EMPTY_RESPONSE",
			"Empty reply from server",
		),
		errorResult(
			"http:browser controller cancel",
			s.SrvHttp.URL+"/cancel",
			"Code: -5011, Msg: CANCELED_BY_BROWSER, Detail: Cancelled by browser controller",
			errors.CanceledByBrowser,
			"CANCELED_BY_BROWSER",
			"Cancelled by browser controller",
		),
		errorResult(
			"https:browser controller cancel",
			s.SrvHttps.URL+"/cancel",
			"Code: -5011, Msg: CANCELED_BY_BROWSER, Detail: Cancelled by browser controller",
			errors.CanceledByBrowser,
			"CANCELED_BY_BROWSER",
			"Cancelled by browser controller",
		),
		errorResult(
			"http:blocked by robots.txt",
			s.SrvHttp.URL+"/blocked",
			"Code: -9998, Msg: PRECLUDED_BY_ROBOTS, Detail: Robots.txt rules precluded fetch",
			errors.PrecludedByRobots,
			"PRECLUDED_BY_ROBOTS",
			"Robots.txt rules precluded fetch",
		),
		errorResult(
			"https:blocked by robots.txt",
			s.SrvHttps.URL+"/blocked",
			"Code: -9998, Msg: PRECLUDED_BY_ROBOTS, Detail: Robots.txt rules precluded fetch",
			errors.PrecludedByRobots,
			"PRECLUDED_BY_ROBOTS",
			"Robots.txt rules precluded fetch",
		),
		success("http:browser controller error", s.SrvHttp.URL+"/bccerr", http.StatusOK, "content from http server", 141),
		success("https:browser controller error", s.SrvHttps.URL+"/bccerr", http.StatusOK, "content from https server", 142),
		{
			name: "http:content writer error",
			url:  s.SrvHttp.URL + "/cwerr",
			wantClient: clientWant{
				status:     http.StatusOK,
				bodyPrefix: "content from http server",
			},
			wantRecord: recordWant{
				statusCode:        http.StatusOK,
				responseBlockSize: intPtr(141),
			},
		},
		{
			name: "https:content writer error",
			url:  s.SrvHttps.URL + "/cwerr",
			wantClient: clientWant{
				status:     http.StatusOK,
				bodyPrefix: "content from https server",
			},
			wantRecord: recordWant{
				statusCode:        http.StatusOK,
				responseBlockSize: intPtr(142),
			},
		},
		{
			name: "http:cached",
			url:  s.SrvHttp.URL + "/cached",
			wantClient: clientWant{
				status:     http.StatusOK,
				bodyPrefix: "content from http server",
			},
			wantRecord: recordWant{
				statusCode:        http.StatusOK,
				responseBlockSize: intPtr(243),
			},
		},
		{
			name: "https:cached",
			url:  s.SrvHttps.URL + "/cached",
			wantClient: clientWant{
				status:     http.StatusOK,
				bodyPrefix: "content from https server",
			},
			wantRecord: recordWant{
				statusCode:        http.StatusOK,
				responseBlockSize: intPtr(244),
			},
		},
		errorResult(
			"http:no host",
			s.SrvHttp.URL[:len(s.SrvHttp.URL)-2]+"1/no_host",
			"Code: -2, Msg: CONNECT_FAILED, Detail: connection refused",
			errors.ConnectFailed,
			"CONNECT_FAILED",
			"connection refused",
		),
		errorResult(
			"https:no host",
			s.SrvHttps.URL[:len(s.SrvHttps.URL)-2]+"1/no_host",
			"Code: -2, Msg: CONNECT_FAILED, Detail: connection refused",
			errors.ConnectFailed,
			"CONNECT_FAILED",
			"connection refused",
		),
		errorResult(
			"https:upstream tls handshake failure",
			s.SrvHttpsBrokenTLS.URL+"/b",
			"Code: -2, Msg: CONNECT_FAILED, Detail: tls: handshake failure",
			errors.ConnectFailed,
			"CONNECT_FAILED",
			"tls: handshake failure",
		),
	}

	for i, tt := range tests {
		tt.keepAlive = true
		tt.generateExpectedRequestsForRecorderProxyThroughProxy()
		grpcServices.Clear()

		t.Run(strconv.Itoa(i)+": "+tt.name, func(t *testing.T) {
			statusCode, got, err := get(tt.url, client, tt.clientTimeout)
			t.Logf("GET url=%v status=%v got=%v error=%v", tt.url, statusCode, string(got), err)

			if grpcServices.DoneBC != nil {
				<-grpcServices.DoneBC
			}
			if grpcServices.DoneCW != nil {
				<-grpcServices.DoneCW
			}

			if (err != nil) != tt.wantClient.err {
				t.Errorf(
					"Client get() error = %v, wantErr %v (%v, %s)",
					err,
					tt.wantClient.err,
					statusCode,
					got,
				)
				return
			}
			if statusCode != tt.wantClient.status {
				t.Errorf("Expected status code: %d, got %d", tt.wantClient.status, statusCode)
				return
			}
			if !strings.HasPrefix(string(got), tt.wantClient.bodyPrefix) {
				t.Errorf("Expected '%s' to start with '%s'", got, tt.wantClient.bodyPrefix)
				return
			}

			compareDNS(t, "DnsResolver", tt.wantGrpcRequests.DnsResolverRequests, grpcServices.Requests.DnsResolverRequests)
			compareBC(t, "BrowserController", tt, tt.wantGrpcRequests.BrowserControllerRequests, grpcServices.Requests.BrowserControllerRequests)
			compareCW(t, "ContentWriter", tt.wantGrpcRequests.ContentWriterRequests, grpcServices.Requests.ContentWriterRequests)
		})
	}
}

func TestRecorderProxyHarvestHeadersBypassBrowserControllerRegister(t *testing.T) {
	s := testutil.NewHttpServers(t)
	defer s.Close()

	grpcServices := testutil.NewGrpcServiceMock()
	defer grpcServices.Close()

	client, recorderProxy, err := localRecorderProxy(t, grpcServices.ClientConn, "")
	if err != nil {
		t.Fatalf("Failed to initialize local recorder proxy: %v", err)
	}
	defer client.CloseIdleConnections()
	defer recorderProxy.Shutdown(context.TODO())

	tt := test{
		name: "https:harvest headers bypass browser controller register",
		url:  s.SrvHttps.URL + "/blocked",
		wantClient: clientWant{
			status:     http.StatusOK,
			bodyPrefix: "content from https server",
		},
		wantRecord: recordWant{
			statusCode:          http.StatusOK,
			responseBlockDigest: boolPtr(true),
			responseBlockSize:   intPtr(142),
		},
		keepAlive: true,
	}
	tt.generateSuccessRequests()

	// Explicit crawl metadata makes the inner GET self-contained, so it does
	// not need BrowserController registration. CONNECT registration and the
	// authoritative completion still happen normally.
	complete := tt.wantGrpcRequests.BrowserControllerRequests[len(tt.wantGrpcRequests.BrowserControllerRequests)-1]
	tt.wantGrpcRequests.BrowserControllerRequests = append(generateConnectOnlyRequests(tt.url), complete)

	headers := http.Header{}
	headers.Set(constants.HeaderCrawlExecutionId, "eid")
	headers.Set(constants.HeaderJobExecutionId, "jid")
	headers.Set(constants.HeaderCollectionId, "col1")

	grpcServices.Clear()

	statusCode, got, err := getWithHeaders(tt.url, client, headers, 0)
	if grpcServices.DoneBC != nil {
		<-grpcServices.DoneBC
	}
	if grpcServices.DoneCW != nil {
		<-grpcServices.DoneCW
	}

	if err != nil {
		t.Fatalf("Client get() error = %v", err)
	}
	if statusCode != tt.wantClient.status {
		t.Fatalf("Expected status code: %d, got %d", tt.wantClient.status, statusCode)
	}
	if !strings.HasPrefix(string(got), tt.wantClient.bodyPrefix) {
		t.Fatalf("Expected '%s' to start with '%s'", got, tt.wantClient.bodyPrefix)
	}

	compareDNS(t, "DnsResolver", tt.wantGrpcRequests.DnsResolverRequests, grpcServices.Requests.DnsResolverRequests)
	compareBC(t, "BrowserController", tt, tt.wantGrpcRequests.BrowserControllerRequests, grpcServices.Requests.BrowserControllerRequests)
	compareCW(t, "ContentWriter", tt.wantGrpcRequests.ContentWriterRequests, grpcServices.Requests.ContentWriterRequests)
}

/**
 * Helper functions
 */

func (tt *test) generateExpectedRequests() {
	switch n := tt.name; {
	case strings.HasSuffix(n, ":client timeout"):
		tt.generateClientTimeoutRequests()
	case strings.HasSuffix(n, ":replace"):
		tt.generateReplaceRequests()
	case strings.HasSuffix(n, ":server timeout"):
		tt.generateServerTimeoutRequests()
	case strings.HasSuffix(n, ":grpc service timeout"):
		tt.generateGrpcServiceTimeoutRequests()
	case strings.HasSuffix(n, ":browser controller cancel"):
		tt.generateBrowserControllerCancelRequests()
	case strings.HasSuffix(n, ":blocked by robots.txt"):
		tt.generateBlockedByRobotsTxtRequests()
	case strings.HasSuffix(n, ":browser controller error"):
		tt.generateBrowserControllerErrorRequests()
	case strings.HasSuffix(n, ":content writer error"):
		tt.generateContentWriterErrorRequests()
	case strings.HasSuffix(n, ":cached"):
		tt.generateCachedRequests()
	case strings.HasSuffix(n, ":no host"):
		tt.generateConnectionRefusedRequests()
	case strings.HasSuffix(n, "handshake failure"):
		tt.generateHandshakeFailureRequests()
	default:
		tt.generateSuccessRequests()
	}
}

func (tt *test) generateExpectedRequestsForRecorderProxyThroughProxy() {
	switch n := tt.name; {
	case strings.HasSuffix(n, ":client timeout"):
		tt.generateClientTimeoutRequests()
	case strings.HasSuffix(n, ":replace"):
		tt.generateReplaceRequests()
	case strings.HasSuffix(n, ":server timeout"):
		tt.generateServerTimeoutRequests()
	case strings.HasSuffix(n, ":grpc service timeout"):
		tt.generateGrpcServiceTimeoutRequests()
	case strings.HasSuffix(n, ":browser controller cancel"):
		tt.generateBrowserControllerCancelRequests()
	case strings.HasSuffix(n, ":blocked by robots.txt"):
		tt.generateBlockedByRobotsTxtRequests()
	case strings.HasSuffix(n, ":browser controller error"):
		tt.generateBrowserControllerErrorRequests()
	case strings.HasSuffix(n, ":content writer error"):
		tt.generateContentWriterErrorRequests()
	case strings.HasSuffix(n, ":cached"):
		tt.generateCachedRequests()
	case strings.HasSuffix(n, ":no host"):
		tt.generateConnectionRefusedThroughProxyRequests()
	case strings.HasSuffix(n, "handshake failure"):
		tt.generateHandshakeFailureRequests()
	default:
		tt.generateSuccessRequests()
	}
}

func (tt *test) parseUrlAndPort() (*url.URL, int) {
	u, _ := url.Parse(tt.url)
	p, _ := strconv.Atoi(u.Port())
	return u, p
}

func (tt *test) clientStatus() int {
	return tt.wantClient.status
}

func (tt *test) clientBodyPrefix() string {
	return tt.wantClient.bodyPrefix
}

func (tt *test) clientBodyLen() int {
	return len(tt.wantClient.bodyPrefix)
}

func (tt *test) recordHTTPStatus() int32 {
	if tt.wantRecord.statusCode != 0 {
		return int32(tt.wantRecord.statusCode)
	}
	return int32(tt.wantClient.status)
}

func (tt *test) recordBlockSize() int64 {
	if tt.wantRecord.responseBlockSize == nil {
		return 0
	}
	return int64(*tt.wantRecord.responseBlockSize)
}

func (tt *test) recordBlockSizeInt32() int32 {
	return int32(tt.recordBlockSize())
}

func (tt *test) recordErrorCode(fallback errors.ErrorCode) int32 {
	if tt.wantRecord.errorCode != 0 {
		return tt.wantRecord.errorCode.Int32()
	}
	return fallback.Int32()
}

func (tt *test) recordErrorMsg(fallback string) string {
	if tt.wantRecord.errorMsg != "" {
		return tt.wantRecord.errorMsg
	}
	return fallback
}

func (tt *test) recordErrorDetail(fallback string) string {
	if tt.wantRecord.errorDetailContains != "" {
		return tt.wantRecord.errorDetailContains
	}
	return fallback
}

func (tt *test) recordError(
	fallbackCode errors.ErrorCode,
	fallbackMsg string,
	fallbackDetail string,
) *commonsV1.Error {
	return &commonsV1.Error{
		Code:   tt.recordErrorCode(fallbackCode),
		Msg:    tt.recordErrorMsg(fallbackMsg),
		Detail: tt.recordErrorDetail(fallbackDetail),
	}
}

func isHttps(uri string) (ok bool, pathStrippedUrl string) {
	u, err := url.Parse(uri)
	if err != nil {
		panic("Failed parsing URL: " + uri)
	}
	if strings.ToLower(u.Scheme) == "https" {
		ok = true
		u.Path = ""
		pathStrippedUrl = u.String()
	}
	return
}

func expectedIP(_ string) string {
	return "127.0.0.1"
}

func generateDnsRequests(rawURL string, alreadyConnected bool) []*dnsresolverV1.ResolveRequest {
	u, _ := url.Parse(rawURL)
	p, _ := strconv.Atoi(u.Port())
	request := &dnsresolverV1.ResolveRequest{
		Host:        u.Hostname(),
		Port:        int32(p),
		ExecutionId: "eid",
		CollectionRef: &configV1.ConfigRef{
			Kind: configV1.Kind_collection,
			Id:   "col1",
		},
	}

	https, _ := isHttps(rawURL)
	switch {
	case https && alreadyConnected:
		return []*dnsresolverV1.ResolveRequest{request}
	case https:
		return []*dnsresolverV1.ResolveRequest{
			request,
			proto.Clone(request).(*dnsresolverV1.ResolveRequest),
		}
	default:
		return []*dnsresolverV1.ResolveRequest{request}
	}
}

func generateConnectOnlyRequests(rawURL string) []*testutil.BrowserControllerRequest {
	_, u := isHttps(rawURL)
	return []*testutil.BrowserControllerRequest{
		generateBccRegisterRequest(http.MethodConnect, u, false),
	}
}

func generateBccRegisterRequest(method, uri string, includeCollectionRef bool) *testutil.BrowserControllerRequest {
	request := &browsercontrollerV2.RegisterResourceRequest{
		ProxyId: 0,
		Method:  method,
		Uri:     uri,
	}
	if includeCollectionRef {
		request.CollectionRef = &configV1.ConfigRef{
			Kind: configV1.Kind_collection,
			Id:   "col1",
		}
	}
	return &testutil.BrowserControllerRequest{RegisterResource: request}
}

func generateBccCompleteRequest(crawlLog *logV1.CrawlLog, cached bool) *testutil.BrowserControllerRequest {
	return &testutil.BrowserControllerRequest{
		CompleteResource: &browsercontrollerV2.CompleteResourceRequest{
			ProxyId:  0,
			CrawlLog: crawlLog,
			Cached:   cached,
		},
	}
}

func generateBccNewRequests(rawURL string, alreadyConnected bool) []*testutil.BrowserControllerRequest {
	return generateBccNewRequestsWithConnectionState(rawURL, alreadyConnected)
}

func generateBccNewRequestsWithConnectionState(rawURL string, alreadyConnected bool) []*testutil.BrowserControllerRequest {
	var r []*testutil.BrowserControllerRequest

	https, u := isHttps(rawURL)
	if https {
		if !alreadyConnected {
			r = append(r, generateBccRegisterRequest(http.MethodConnect, u, false))
		}
		r = append(r, generateBccRegisterRequest(http.MethodGet, rawURL, true))
		return r
	}

	r = append(r, generateBccRegisterRequest(http.MethodGet, rawURL, false))
	return r
}

func generateCwProtocolHeaderRequest(u *url.URL, keepAlive bool) (*contentwriterV1.WriteRequest, int64) {
	k := ""
	if !keepAlive {
		k = "Connection: close\r\n"
	}

	header := fmt.Appendf(
		nil,
		"GET %s HTTP/1.1\r\nHost: %s:%s\r\nAccept-Encoding: gzip\r\n%sUser-Agent: Go-http-client/1.1\r\n\r\n",
		u.RequestURI(),
		u.Hostname(),
		u.Port(),
		k,
	)

	req := &contentwriterV1.WriteRequest{
		Value: &contentwriterV1.WriteRequest_ProtocolHeader{
			ProtocolHeader: &contentwriterV1.Data{
				RecordNum: 0,
				Data:      header,
			},
		},
	}
	return req, int64(len(header))
}

func generateCwProtocolHeaderResponse(status int, contentLength int) *contentwriterV1.WriteRequest {
	nosniffHeader := ""
	if status == http.StatusNotFound {
		nosniffHeader = "X-Content-Type-Options: nosniff\r\n"
	}

	header := fmt.Appendf(
		nil,
		"HTTP/1.1 %d %s\r\nContent-Length: %d\r\nContent-Type: text/plain; charset=utf-8\r\nDate: Wed, 15 May 2019 12:41:02 GMT\r\n%s\r\n",
		status,
		http.StatusText(status),
		contentLength,
		nosniffHeader,
	)

	return &contentwriterV1.WriteRequest{
		Value: &contentwriterV1.WriteRequest_ProtocolHeader{
			ProtocolHeader: &contentwriterV1.Data{
				RecordNum: 1,
				Data:      header,
			},
		},
	}
}

func (tt *test) successCrawlLog(targetURI, ipAddress string) *logV1.CrawlLog {
	return &logV1.CrawlLog{
		WarcId:              "warcid_1",
		StatusCode:          tt.recordHTTPStatus(),
		Size:                tt.recordBlockSize(),
		Method:              "GET",
		RequestedUri:        targetURI,
		ContentType:         "text/plain; charset=utf-8",
		StorageRef:          "storageRef_1",
		RecordType:          "revisit",
		WarcRefersTo:        "revisit_0",
		IpAddress:           ipAddress,
		ExecutionId:         "eid",
		JobExecutionId:      "jid",
		CollectionFinalName: "collection_0",
	}
}

func (tt *test) cachedCrawlLog(targetURI, ipAddress string) *logV1.CrawlLog {
	return &logV1.CrawlLog{
		StatusCode:     tt.recordHTTPStatus(),
		Size:           tt.recordBlockSize(),
		Method:         "GET",
		RequestedUri:   targetURI,
		ContentType:    "text/plain; charset=utf-8",
		IpAddress:      ipAddress,
		ExecutionId:    "eid",
		JobExecutionId: "jid",
	}
}

func (tt *test) errorCrawlLog(
	targetURI string,
	ipAddress string,
	code errors.ErrorCode,
	msg string,
	detail string,
) *logV1.CrawlLog {
	return &logV1.CrawlLog{
		StatusCode:     tt.recordErrorCode(code),
		RequestedUri:   targetURI,
		Method:         "GET",
		RecordType:     "response",
		IpAddress:      ipAddress,
		ExecutionId:    "eid",
		JobExecutionId: "jid",
		Error:          tt.recordError(code, msg, detail),
	}
}

func (tt *test) clientGoneCrawlLog(targetURI, ipAddress string) *logV1.CrawlLog {
	return &logV1.CrawlLog{
		StatusCode:     errors.CanceledByBrowser.Int32(),
		RequestedUri:   targetURI,
		Method:         "GET",
		RecordType:     "response",
		IpAddress:      ipAddress,
		ExecutionId:    "eid",
		JobExecutionId: "jid",
		Error: &commonsV1.Error{
			Code:   errors.CanceledByBrowser.Int32(),
			Msg:    "CANCELED_BY_BROWSER",
			Detail: "Veidemann recorder proxy lost connection to client",
		},
	}
}

func (tt *test) contentWriterRequests(
	u *url.URL,
	targetURI string,
	ipAddress string,
	includeMeta bool,
) []*contentwriterV1.WriteRequest {
	requestHeader, requestLength := generateCwProtocolHeaderRequest(u, tt.keepAlive)

	requests := []*contentwriterV1.WriteRequest{
		requestHeader,
		generateCwProtocolHeaderResponse(tt.clientStatus(), tt.clientBodyLen()),
		{
			Value: &contentwriterV1.WriteRequest_Payload{
				Payload: &contentwriterV1.Data{
					RecordNum: 1,
					Data:      []byte(tt.clientBodyPrefix()),
				},
			},
		},
	}

	if !includeMeta {
		return requests
	}

	requests = append(requests, &contentwriterV1.WriteRequest{
		Value: &contentwriterV1.WriteRequest_Meta{
			Meta: &contentwriterV1.WriteRequestMeta{
				ExecutionId:   "eid",
				TargetUri:     targetURI,
				IpAddress:     ipAddress,
				CollectionRef: &configV1.ConfigRef{Kind: configV1.Kind_collection, Id: "col1"},
				RecordMeta: map[int32]*contentwriterV1.WriteRequestMeta_RecordMeta{
					0: {
						RecordNum:         0,
						Type:              contentwriterV1.RecordType_REQUEST,
						RecordContentType: "application/http; msgtype=request",
						Size:              requestLength,
					},
					1: {
						RecordNum:         1,
						Type:              contentwriterV1.RecordType_RESPONSE,
						RecordContentType: "application/http; msgtype=response",
						Size:              tt.recordBlockSize(),
					},
				},
			},
		},
	})

	return requests
}

func (tt *test) generateRecordedHTTPResponseRequests(alreadyConnected bool) {
	u, _ := tt.parseUrlAndPort()
	targetURI := tt.url
	ipAddress := expectedIP(tt.url)

	r := &testutil.Requests{}
	r.DnsResolverRequests = generateDnsRequests(tt.url, alreadyConnected)

	r.BrowserControllerRequests = append(
		generateBccNewRequests(tt.url, alreadyConnected),
		generateBccCompleteRequest(tt.successCrawlLog(targetURI, ipAddress), false),
	)

	r.ContentWriterRequests = tt.contentWriterRequests(u, targetURI, ipAddress, true)

	tt.wantGrpcRequests = r
}

func (tt *test) generateSuccessRequests() {
	tt.generateRecordedHTTPResponseRequests(false)
}

func (tt *test) generateReplaceRequests() {
	tt.generateRecordedHTTPResponseRequests(true)
}

func (tt *test) generateBrowserControllerErrorRequests() {
	tt.generateRecordedHTTPResponseRequests(false)
}

func (tt *test) generateClientTimeoutRequests() {
	u, _ := tt.parseUrlAndPort()

	r := &testutil.Requests{}

	alreadyConnected := strings.HasPrefix(tt.name, "https:")
	targetURI := tt.url
	ipAddress := expectedIP(tt.url)

	r.DnsResolverRequests = generateDnsRequests(tt.url, alreadyConnected)

	r.BrowserControllerRequests = append(
		generateBccNewRequests(tt.url, alreadyConnected),
		generateBccCompleteRequest(tt.clientGoneCrawlLog(targetURI, ipAddress), false),
	)

	requestHeader, _ := generateCwProtocolHeaderRequest(u, tt.keepAlive)

	upstreamBody := "content from http server"
	if strings.HasPrefix(tt.name, "https:") {
		upstreamBody = "content from https server"
	}

	r.ContentWriterRequests = []*contentwriterV1.WriteRequest{
		requestHeader,
		generateCwProtocolHeaderResponse(http.StatusOK, len(upstreamBody)),
		{
			Value: &contentwriterV1.WriteRequest_Cancel{
				Cancel: "Veidemann recorder proxy lost connection to client",
			},
		},
	}

	tt.wantGrpcRequests = r
}

func (tt *test) generateServerTimeoutRequests() {
	r := &testutil.Requests{}

	alreadyConnected := strings.HasPrefix(tt.name, "https:")
	targetURI := tt.url
	ipAddress := expectedIP(tt.url)

	r.DnsResolverRequests = generateDnsRequests(tt.url, alreadyConnected)

	r.BrowserControllerRequests = append(
		generateBccNewRequests(tt.url, alreadyConnected),
		generateBccCompleteRequest(
			tt.errorCrawlLog(
				targetURI,
				ipAddress,
				errors.EmptyResponse,
				"EMPTY_RESPONSE",
				"Empty reply from server",
			),
			false,
		),
	)

	r.ContentWriterRequests = []*contentwriterV1.WriteRequest{
		{
			Value: &contentwriterV1.WriteRequest_Cancel{
				Cancel: tt.recordErrorDetail("Empty reply from server"),
			},
		},
	}

	tt.wantGrpcRequests = r
}

func (tt *test) generateGrpcServiceTimeoutRequests() {
	u, _ := tt.parseUrlAndPort()

	r := &testutil.Requests{}
	r.DnsResolverRequests = generateDnsRequests(tt.url, false)
	r.BrowserControllerRequests = generateBccNewRequests(tt.url, false)

	requestHeader, _ := generateCwProtocolHeaderRequest(u, tt.keepAlive)
	r.ContentWriterRequests = []*contentwriterV1.WriteRequest{
		requestHeader,
	}

	tt.wantGrpcRequests = r
}

func (tt *test) generateBrowserControllerCancelRequests() {
	r := &testutil.Requests{}

	targetURI := tt.url
	ipAddress := ""
	if https, _ := isHttps(tt.url); https {
		// CONNECT is registered and resolved before BrowserController sees and
		// cancels the inner HTTPS request.
		r.DnsResolverRequests = generateDnsRequests(tt.url, true)
	}

	crawlLog := tt.errorCrawlLog(
		targetURI,
		ipAddress,
		errors.CanceledByBrowser,
		"CANCELED_BY_BROWSER",
		"Cancelled by browser controller",
	)
	crawlLog.ExecutionId = ""
	crawlLog.JobExecutionId = ""

	r.BrowserControllerRequests = append(
		generateBccNewRequests(tt.url, false),
		generateBccCompleteRequest(crawlLog, false),
	)

	r.ContentWriterRequests = []*contentwriterV1.WriteRequest{}

	tt.wantGrpcRequests = r
}

func (tt *test) generateBlockedByRobotsTxtRequests() {
	r := &testutil.Requests{}

	targetURI := tt.url
	ipAddress := ""
	https, _ := isHttps(tt.url)
	if https {
		r.DnsResolverRequests = generateDnsRequests(tt.url, true)
	}

	r.BrowserControllerRequests = generateBccNewRequests(tt.url, false)

	crawlLog := tt.errorCrawlLog(
		targetURI,
		ipAddress,
		errors.PrecludedByRobots,
		"PRECLUDED_BY_ROBOTS",
		"Robots.txt rules precluded fetch",
	)
	crawlLog.ExecutionId = ""
	crawlLog.JobExecutionId = ""

	r.BrowserControllerRequests = append(
		r.BrowserControllerRequests,
		generateBccCompleteRequest(crawlLog, false),
	)

	r.ContentWriterRequests = []*contentwriterV1.WriteRequest{}

	tt.wantGrpcRequests = r
}

func (tt *test) generateContentWriterErrorRequests() {
	u, _ := tt.parseUrlAndPort()

	alreadyConnected := strings.HasPrefix(tt.name, "https:")
	targetURI := tt.url
	ipAddress := expectedIP(tt.url)

	r := &testutil.Requests{}
	r.DnsResolverRequests = generateDnsRequests(tt.url, alreadyConnected)

	r.BrowserControllerRequests = append(
		generateBccNewRequests(tt.url, alreadyConnected),
		generateBccCompleteRequest(
			tt.errorCrawlLog(
				targetURI,
				ipAddress,
				errors.RuntimeException,
				"Error writing to content writer",
				"rpc error: code = InvalidArgument desc = Fake error",
			),
			false,
		),
	)

	r.ContentWriterRequests = tt.contentWriterRequests(u, targetURI, ipAddress, true)

	tt.wantGrpcRequests = r
}

func (tt *test) generateCachedRequests() {
	u, _ := tt.parseUrlAndPort()

	alreadyConnected := false
	targetURI := tt.url
	ipAddress := expectedIP(tt.url)

	r := &testutil.Requests{}
	r.DnsResolverRequests = generateDnsRequests(tt.url, alreadyConnected)

	r.BrowserControllerRequests = append(
		generateBccNewRequests(tt.url, alreadyConnected),
		generateBccCompleteRequest(tt.cachedCrawlLog(targetURI, ipAddress), true),
	)

	requestHeader, _ := generateCwProtocolHeaderRequest(u, tt.keepAlive)
	r.ContentWriterRequests = []*contentwriterV1.WriteRequest{
		requestHeader,
		{
			Value: &contentwriterV1.WriteRequest_Cancel{
				Cancel: "OK: Loaded from cache",
			},
		},
	}

	tt.wantGrpcRequests = r
}

func (tt *test) generateConnectionRefusedRequests() {
	u, _ := tt.parseUrlAndPort()

	r := &testutil.Requests{}

	r.DnsResolverRequests = generateDnsRequests(tt.url, false)

	r.BrowserControllerRequests = append(
		generateBccNewRequests(tt.url, false),
		generateBccCompleteRequest(
			tt.errorCrawlLog(
				tt.url,
				"127.0.0.1",
				errors.ConnectFailed,
				"CONNECT_FAILED",
				"connection refused",
			),
			false,
		),
	)

	requestHeader, _ := generateCwProtocolHeaderRequest(u, tt.keepAlive)
	r.ContentWriterRequests = []*contentwriterV1.WriteRequest{
		requestHeader,
		{
			Value: &contentwriterV1.WriteRequest_Cancel{
				Cancel: tt.recordErrorDetail("connection refused"),
			},
		},
	}

	tt.wantGrpcRequests = r
}

func (tt *test) generateConnectionRefusedThroughProxyRequests() {
	tt.generateConnectionRefusedRequests()
}

func (tt *test) generateHandshakeFailureRequests() {
	u, _ := tt.parseUrlAndPort()

	r := &testutil.Requests{}

	r.DnsResolverRequests = generateDnsRequests(tt.url, false)
	r.BrowserControllerRequests = append(
		generateBccNewRequests(tt.url, false),
		generateBccCompleteRequest(
			tt.errorCrawlLog(
				tt.url,
				"127.0.0.1",
				errors.ConnectFailed,
				"CONNECT_FAILED",
				"tls: handshake failure",
			),
			false,
		),
	)

	requestHeader, _ := generateCwProtocolHeaderRequest(u, tt.keepAlive)
	r.ContentWriterRequests = []*contentwriterV1.WriteRequest{
		requestHeader,
		{
			Value: &contentwriterV1.WriteRequest_Cancel{
				Cancel: tt.recordErrorDetail("tls: handshake failure"),
			},
		},
	}

	tt.wantGrpcRequests = r
}

func compareCW(t testing.TB, serviceName string, want []*contentwriterV1.WriteRequest, got []*contentwriterV1.WriteRequest) {
	t.Helper()

	if len(want) == 0 && len(got) == 0 {
		return
	}

	// If last request is a cancel request, then we don't care about the others
	if len(want) > 0 {
		lastWant := want[len(want)-1].Value
		if _, ok := lastWant.(*contentwriterV1.WriteRequest_Cancel); ok {
			if len(got) == 0 {
				// No requests at all is treated similar to cancel
				return
			}
			lastGot := got[len(got)-1].Value
			if reflect.DeepEqual(lastGot, lastWant) {
				return
			} else {
				t.Errorf("%s service got wrong cwcCancelFunc request.  %s request #%d\nWas:\n%v\nWant:\n%v", serviceName, serviceName,
					len(got), printRequest(lastGot), printRequest(lastWant))
			}
		}
	}

	for i, r := range want {
		if i >= len(got) {
			t.Errorf("%s service received too few requests. Got %d, want %d.\nFirst missing request is:\n%v", serviceName,
				len(got), len(want), printRequest(want[len(got)]))
		} else {
			if !compareCwWriteRequest(t, r, got[i]) {
				t.Errorf("Got wrong %s request. %s request #%d\nWas:\n%v\nWant:\n%v", serviceName, serviceName,
					i+1, printRequest(got[i]), printRequest(want[i]))
			}
		}
	}
	if len(got) > len(want) {
		t.Errorf("%s service received too many requests. Got %d, want %d.\nFirst unwanted request is:\n%v", serviceName,
			len(got), len(want), printRequest(got[len(want)]))
	}
}

func compareBC(t testing.TB, serviceName string, tt test, want []*testutil.BrowserControllerRequest, got []*testutil.BrowserControllerRequest) {
	t.Helper()

	for i, r := range want {
		if i >= len(got) {
			t.Errorf("%s service received too few requests. Got %d, want %d.\nFirst missing request is:\n%v", serviceName,
				len(got), len(want), printRequest(want[len(got)]))
			listGotWant(t, got, want)
		} else {
			if !compareBcRequest(t, tt, r, got[i]) {
				t.Errorf("Got wrong %s request. %s request #%d\nWas:\n%v\nWant:\n%v", serviceName, serviceName,
					i+1, printRequest(got[i]), printRequest(want[i]))
				if diff := deep.Equal(got[i], want[i]); diff != nil {
					t.Error(diff)
				}
			}
		}
	}
	if len(got) > len(want) {
		t.Errorf("%s service received too many requests. Got %d, want %d.\nFirst unwanted request is:\n%v", serviceName,
			len(got), len(want), printRequest(got[len(want)]))
		listGotWant(t, got, want)
	}
}
func listGotWant(t testing.TB, got, want interface{}) {
	t.Helper()

	g := reflect.ValueOf(got)
	for i := 0; i < g.Len(); i++ {
		t.Errorf(" GOT: %v", g.Index(i))
	}
	w := reflect.ValueOf(want)
	for i := 0; i < w.Len(); i++ {
		t.Errorf("WANT: %v", w.Index(i))
	}
}

func compareDNS(t testing.TB, serviceName string, want []*dnsresolverV1.ResolveRequest, got []*dnsresolverV1.ResolveRequest) {
	t.Helper()

	for i, r := range want {
		if i >= len(got) {
			t.Errorf("%s service received too few requests. Got %d, want %d.\nFirst missing request is:\n%v", serviceName,
				len(got), len(want), printRequest(want[len(got)]))
		} else {
			if !proto.Equal(got[i], r) {
				t.Errorf("Got wrong %s request. %s request #%d\nWas:\n%v\nWant:\n%v", serviceName, serviceName,
					i+1, printRequest(got[i]), printRequest(want[i]))
			}
		}
	}
	if len(got) > len(want) {
		t.Errorf("%s service received too many requests. Got %d, want %d.\nFirst unwanted request is:\n%v", serviceName,
			len(got), len(want), printRequest(got[len(want)]))
	}
}

var dateRe = regexp.MustCompile(`(?sm)^(.*Date: )([^\r\n]+)(.*)$`)

func compareCwWriteRequest(t testing.TB, want *contentwriterV1.WriteRequest, got *contentwriterV1.WriteRequest) (ok bool) {
	t.Helper()

	switch wt := want.Value.(type) {
	case *contentwriterV1.WriteRequest_ProtocolHeader:
		wantBytes := wt.ProtocolHeader.Data
		g, o := got.Value.(*contentwriterV1.WriteRequest_ProtocolHeader)
		if !o {
			ok = false
			return
		}
		gotBytes := g.ProtocolHeader.Data
		if reflect.DeepEqual(wantBytes, gotBytes) {
			ok = true
		} else {
			// Compare date in string to time.Now()
			gotDateM := dateRe.FindSubmatch(gotBytes)
			if gotDateM == nil {
				ok = false
			} else {
				wantT := time.Now()
				gotT, err := time.Parse(time.RFC1123, string(gotDateM[2]))
				if err != nil || wantT.Sub(gotT) > 10*time.Second {
					t.Errorf("Date differs to much: Got '%v' which is %v ago\n", gotT, wantT.Sub(gotT))
					ok = false
				} else {
					wantDateM := dateRe.FindSubmatch(wantBytes)
					gotBytes = []byte(fmt.Sprintf("%s%s%s", gotDateM[1], wantDateM[2], gotDateM[3]))
					got.Value.(*contentwriterV1.WriteRequest_ProtocolHeader).ProtocolHeader.Data = gotBytes
					if proto.Equal(want, got) {
						ok = true
					} else {
						ok = false
					}
				}
			}
		}
	case *contentwriterV1.WriteRequest_Meta:
		gotT := got.GetMeta().FetchTimeStamp.AsTime()
		wantT := time.Now()
		if wantT.Sub(gotT) > 10*time.Second {
			t.Errorf("Date differs to much: Got '%v' which is %v ago\n", gotT, wantT.Sub(gotT))
			ok = false
		} else {
			got.GetMeta().FetchTimeStamp = nil

			// Remove block digest since we cannot calculate the right value without access to content
			for _, r := range got.GetMeta().RecordMeta {
				if r.BlockDigest == "" {
					t.Errorf("Missing BlockDigest")
					return false
				}
				r.BlockDigest = ""
			}

			if proto.Equal(want, got) {
				ok = true
			} else {
				ok = false
			}
		}
	default:
		ok = true
		if !proto.Equal(got, want) {
			ok = false
		}
	}
	return
}

func compareBcRequest(t testing.TB, tt test, want *testutil.BrowserControllerRequest, got *testutil.BrowserControllerRequest) bool {
	t.Helper()

	switch {
	case want.RegisterResource != nil || got.RegisterResource != nil:
		if want.RegisterResource == nil || got.RegisterResource == nil {
			return false
		}
		return proto.Equal(want.RegisterResource, got.RegisterResource)
	case want.CompleteResource != nil || got.CompleteResource != nil:
		if want.CompleteResource == nil || got.CompleteResource == nil {
			return false
		}
		return compareBcCompleteRequest(t, tt, want.CompleteResource, got.CompleteResource)
	default:
		return want == nil && got == nil
	}
}

func compareBcCompleteRequest(
	t testing.TB,
	tt test,
	want *browsercontrollerV2.CompleteResourceRequest,
	got *browsercontrollerV2.CompleteResourceRequest,
) bool {
	t.Helper()

	if want.GetCrawlLog() == nil || got.GetCrawlLog() == nil {
		return proto.Equal(want, got)
	}

	g := proto.Clone(got).(*browsercontrollerV2.CompleteResourceRequest)

	allowedTimeDiff := time.Duration(g.GetCrawlLog().FetchTimeMs+1500) * time.Millisecond
	if !checkTime(t, g.GetCrawlLog().FetchTimeStamp.AsTime(), allowedTimeDiff) {
		return false
	}
	g.GetCrawlLog().FetchTimeStamp = nil

	if tt.wantRecord.responseBlockDigest != nil {
		wantDigest := *tt.wantRecord.responseBlockDigest
		gotDigest := g.GetCrawlLog().BlockDigest != ""

		if wantDigest && !gotDigest {
			t.Errorf("Missing BlockDigest")
			return false
		}

		if !wantDigest && gotDigest {
			t.Errorf("BlockDigest was not expected")
			return false
		}
	}

	g.GetCrawlLog().BlockDigest = ""
	g.GetCrawlLog().FetchTimeMs = 0

	return proto.Equal(want, g)
}

func checkTime(t testing.TB, ts time.Time, allowedDiff time.Duration) bool {
	t.Helper()

	wantT := time.Now()
	if wantT.Sub(ts) > allowedDiff {
		t.Errorf("Date differs to much: Got '%v' which is %v ago\n", ts, wantT.Sub(ts))
		return false
	} else {
		return true
	}
}

func get(uri string, client *http.Client, timeout time.Duration) (int, []byte, error) {
	return getWithHeaders(uri, client, nil, timeout)
}

func getWithHeaders(uri string, client *http.Client, headers http.Header, timeout time.Duration) (int, []byte, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return 0, nil, err
	}

	for name, values := range headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	if timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

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

func printRequest(req interface{}) string {
	return fmt.Sprintf("%30T: %v\n", req, req)
}

// localRecorderProxy creates a new recorderproxy which uses internal transport
func localRecorderProxy(t testing.TB, conn *serviceconnections.Connections, nextProxyAddr string) (*http.Client, *recorderproxy.RecorderProxy, error) {
	t.Helper()

	host := "localhost"
	port := 0

	proxy := recorderproxy.NewRecorderProxy(0, conn, nextProxyAddr)

	ln, err := proxy.Listen(host, port)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create recorder proxy: %v", err)
	}

	proxyAddr := ln.Addr().String()

	t.Logf("Local recorder proxy listening on %s", proxyAddr)

	if nextProxyAddr != "" {
		t.Logf("Local recorder proxy forwarding to next proxy at %s", nextProxyAddr)
	}

	go func() {
		if err := proxy.Serve(ln); err != nil && !stderrors.Is(err, net.ErrClosed) {
			panic(err)
		}
	}()

	proxyUrl, _ := url.Parse("http://" + proxyAddr)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:   acceptAllCerts,
			Proxy:             http.ProxyURL(proxyUrl),
			DisableKeepAlives: false,
		},
	}

	return client, proxy, nil
}
