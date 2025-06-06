//go:build integration
// +build integration

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

package server

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier"
	robotsevaluatorV1 "github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/database"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/logger"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/logwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/screenshotwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/serviceconnections"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/testutil"
	logServiceTestUtil "github.com/NationalLibraryOfNorway/veidemann/log-service/pkg/testutil"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/recorderproxy"
	proxyServiceConnections "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/serviceconnections"
	proxyTestUtil "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

// sessions is a registry of sessions
var sessions *session.Registry

// localhost is the ip address of the host machine
var localhost = GetOutboundIP().String()

// provider is a flag to select container provider
var provider = flag.String("provider", "docker", "container provider, \"docker\" or \"podman\".")

func TestMain(m *testing.M) {
	// Parse flags
	flag.Parse()

	// Set recorderproxy log level to warn to avoid too much output
	logrus.SetLevel(logrus.WarnLevel)

	// Set log level
	logger.InitLog("debug", "logfmt", false)

	// setup browser
	ctx, cancelBrowser := context.WithCancel(context.Background())
	defer cancelBrowser()
	browserHost, browserPort, err := setupBrowser(ctx)
	if err != nil {
		panic(err)
	}

	// setup database mock
	dbMock := setupDbMock()
	dbAdapter := database.NewConfigCache(dbMock.RethinkDbConnection, time.Minute)

	// setup screenshot writer mock
	screenShotWriter := &testutil.ScreenshotWriterMock{
		WriteFunc: func(data []byte, metadata screenshotwriter.Metadata) error {
			b := bytes.NewBuffer(data)
			f, err := os.Create("screenshot.png")
			if err != nil {
				return fmt.Errorf("error opening file: %w", err)
			}
			defer func() {
				_ = f.Close()
			}()
			_, err = io.Copy(f, b)
			if err != nil {
				return fmt.Errorf("failed to copy screenshot data to file: %w", err)
			}
			return nil
		},
		CloseFunc: func() error {
			return os.Remove("screenshot.png")
		},
	}

	// setup log service mock
	logServiceMock := logServiceTestUtil.NewLogServiceMock(5002)

	// setup writer client
	logWriter := logwriter.New(
		serviceconnections.WithPort(5002),
	)
	if err := logWriter.Connect(); err != nil {
		panic(err)
	}

	// setup sessions
	sessions = session.NewRegistry(
		2,
		session.WithBrowserHost(browserHost),
		session.WithBrowserPort(browserPort),
		session.WithProxyHost(localhost),
		session.WithProxyPort(6666),
		session.WithConfigCache(dbAdapter),
		session.WithScreenshotWriter(screenShotWriter),
		session.WithLogWriter(logWriter),
	)

	// setup robots evaluator mock
	robotsEvaluator := &testutil.RobotsEvaluatorMock{IsAllowedFunc: func(_ *robotsevaluatorV1.IsAllowedRequest) bool {
		return true
	}}

	// setup api server
	apiServer := NewApiServer("", 7777, sessions, robotsEvaluator, logWriter)
	go func() {
		_ = apiServer.Start()
	}()

	// setup recorder proxy
	opt := proxyTestUtil.WithExternalBrowserController(
		proxyServiceConnections.NewConnectionOptions("BrowserController",
			proxyServiceConnections.WithHost("localhost"),
			proxyServiceConnections.WithPort("7777"),
			proxyServiceConnections.WithConnectTimeout(10*time.Second),
		),
	)
	grpcServices := proxyTestUtil.NewGrpcServiceMock(opt)
	recorderProxy0 := localRecorderProxy(0, grpcServices.ClientConn, "")
	recorderProxy1 := localRecorderProxy(1, grpcServices.ClientConn, "")
	recorderProxy2 := localRecorderProxy(2, grpcServices.ClientConn, "")

	// Run the tests
	code := m.Run()

	// Clean up
	sessions.CloseWait(1 * time.Minute)
	apiServer.Close()
	grpcServices.Close()
	recorderProxy0.Close()
	recorderProxy1.Close()
	recorderProxy2.Close()
	_ = screenShotWriter.Close()
	_ = dbMock.Close()
	_ = logWriter.Close()
	logServiceMock.Close()
	cancelBrowser()

	os.Exit(code)
}

func TestSession_Fetch(t *testing.T) {
	conf := &configV1.ConfigObject{
		Id:         "conf1",
		ApiVersion: "",
		Kind:       configV1.Kind_crawlConfig,
		Meta:       nil,
		Spec: &configV1.ConfigObject_CrawlConfig{CrawlConfig: &configV1.CrawlConfig{
			BrowserConfigRef: &configV1.ConfigRef{Id: "browserConfig1"},
			PolitenessRef:    &configV1.ConfigRef{Id: "politenessConfig1"},
			CollectionRef:    &configV1.ConfigRef{Id: "collectionConfig1"},
			Extra:            &configV1.ExtraConfig{CreateScreenshot: true},
		}},
	}

	tests := []struct {
		name string
		url  *frontierV1.QueuedUri
	}{
		{"elg", &frontierV1.QueuedUri{Uri: "http://elg.no", DiscoveryPath: "L", JobExecutionId: "jid", ExecutionId: "eid"}},
		{"vg", &frontierV1.QueuedUri{Uri: "http://vg.no", DiscoveryPath: "L", JobExecutionId: "jid", ExecutionId: "eid"}},
		{"nb", &frontierV1.QueuedUri{Uri: "http://nb.no", DiscoveryPath: "L", JobExecutionId: "jid", ExecutionId: "eid"}},
		{"fhi", &frontierV1.QueuedUri{Uri: "http://fhi.no", DiscoveryPath: "L", JobExecutionId: "jid", ExecutionId: "eid"}},
		{"db", &frontierV1.QueuedUri{Uri: "http://db.no", DiscoveryPath: "L", JobExecutionId: "jid", ExecutionId: "eid"}},
		{"maps", &frontierV1.QueuedUri{Uri: "https://goo.gl/maps/EmpIH", DiscoveryPath: "L", JobExecutionId: "jid", ExecutionId: "eid"}},
		{"ranano", &frontierV1.QueuedUri{Uri: "https://ranano.no/", DiscoveryPath: "L", JobExecutionId: "jid", ExecutionId: "eid"}},
		{"cynergi", &frontierV1.QueuedUri{Uri: "https://www.cynergi.no/", DiscoveryPath: "L", JobExecutionId: "jid", ExecutionId: "eid"}},
		{"pdf1", &frontierV1.QueuedUri{Uri: "https://www.nb.no/content/uploads/2019/04/tildelingsbrev_nasjonalbiblioteket_2019.pdf", DiscoveryPath: "L", JobExecutionId: "jid", ExecutionId: "eid"}},
	}
	for _, tt := range tests {
		ctx := context.Background()
		t.Run(tt.name, func(t *testing.T) {
			// Get next available session
			s, err := sessions.GetNextAvailable(ctx)
			if err != nil {
				t.Error(err)
			}
			defer sessions.Release(s)

			// Fetch page
			result, err := s.Fetch(context.Background(), &frontierV1.PageHarvestSpec{
				QueuedUri:    tt.url,
				CrawlConfig:  conf,
				SessionToken: "test",
			})
			if err != nil {
				t.Error(err)
			} else {
				t.Logf("Resource count: %v, Time: %v\n", result.UriCount, result.PageFetchTimeMs)
			}
			time.Sleep(time.Second * 4)
		})
	}
}

func setupDbMock() *database.MockConnection {
	dbConn := database.NewMockConnection()
	dbConn.GetMock().On(r.Table("config").Get("browserConfig1")).Return(
		map[string]interface{}{
			"id":   "browserConfig1",
			"kind": "browserConfig",
			"meta": map[string]interface{}{
				"name":    "browser config 1",
				"label":   []map[string]interface{}{{"key": "foo", "value": "bar"}},
				"created": "2020-04-06T18:17:50.343827619Z",
			},
			"browserConfig": map[string]interface{}{
				"windowWidth":         1400,
				"windowHeight":        1280,
				"maxInactivityTimeMs": 5000,
				"pageLoadTimeoutMs":   60000,
				"scriptRef":           []map[string]interface{}{{"kind": "browserScript", "id": "script1"}},
			},
		},
		nil,
	)
	dbConn.GetMock().On(r.Table("config").Get("script1")).Return(
		map[string]interface{}{
			"id":   "script1",
			"kind": "browserScript",
			"meta": map[string]interface{}{
				"name":        "script1",
				"description": "script1",
				"label":       []map[string]interface{}{{"key": "type", "value": "extract_outlinks"}},
			},
			"browserScript": map[string]interface{}{
				"browserScriptType": "EXTRACT_OUTLINKS",
				"script": `
(function extractOutlinks(frame) {
   const framesDone = new Set();
   function isValid(link) {
   return (link != null
		 && link.attributes.href.value != ""
		 && link.attributes.href.value != "#"
		 && link.protocol != "tel:"
		 && link.protocol != "mailto:"
		);
   }
   function compileOutlinks(frame) {
	 framesDone.add(frame);
	 if (frame && frame.document) {
	   let outlinks = Array.from(frame.document.links);
	   for (var i = 0; i < frame.frames.length; i++) {
		 if (frame.frames[i] && !framesDone.has(frame.frames[i])) {
		   try {
			 outlinks = outlinks.concat(compileOutlinks(frame.frames[i]));
		   } catch {}
		 }
	   }
	   return outlinks;
	 }
	 return [];
   }
   return Array.from(new Set(compileOutlinks(frame).filter(isValid).map(_ => _.href)));
 })(window);
`,
			},
		},
		nil,
	)
	dbConn.GetMock().On(r.Table("config").Get("politenessConfig1")).Return(
		map[string]interface{}{
			"id":   "politenessConfig1",
			"kind": "politenessConfig",
			"meta": map[string]interface{}{
				"name":    "politeness config 1",
				"label":   []map[string]interface{}{{"key": "foo", "value": "bar"}},
				"created": "2020-04-06T18:17:50.343827619Z",
			},
			"politenessConfig": map[string]interface{}{},
		}, nil)
	dbConn.GetMock().On(r.Table("page_log").Insert(r.MockAnything())).Return(map[string]interface{}{}, nil)
	dbConn.GetMock().On(r.Table("crawl_log").Insert(r.MockAnything())).Return(map[string]interface{}{}, nil)

	return dbConn
}

// localRecorderProxy creates a new recorderproxy which uses internal transport
func localRecorderProxy(id int, conn *proxyServiceConnections.Connections, nextProxyAddr string) (proxy *recorderproxy.RecorderProxy) {
	proxy = recorderproxy.NewRecorderProxy(id, "0.0.0.0", 6666, conn, 5*time.Second, nextProxyAddr)
	proxy.Start()
	return
}

// GetOutboundIP returrns the preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = conn.Close()
	}()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}

func setupBrowser(ctx context.Context) (host string, port int, err error) {
	// Determine container provider
	skipReaper := false
	var providerType testcontainers.ProviderType
	if *provider == "podman" {
		providerType = testcontainers.ProviderPodman
		skipReaper = true
	} else {
		providerType = testcontainers.ProviderDocker
	}
	// Start browserless container
	browserless, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ProviderType: providerType,
		ContainerRequest: testcontainers.ContainerRequest{
			SkipReaper: skipReaper,
			Env: map[string]string{
				"DEBUG": "*",
			},
			Image:        "browserless/chrome:1.36.0-puppeteer-3.3.0",
			ExposedPorts: []string{"3000/tcp"},
			WaitingFor:   wait.ForListeningPort("3000/tcp"),
		},
		Started: true,
	})
	if err != nil {
		return
	}
	host, err = browserless.Host(ctx)
	if err != nil {
		return
	}
	var browserPort nat.Port
	browserPort, err = browserless.MappedPort(ctx, "3000/tcp")
	if err != nil {
		return
	}
	port = browserPort.Int()
	return
}
