//go:build integration

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
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	browsercontrollerV2 "github.com/NationalLibraryOfNorway/veidemann/api/browsercontroller/v2"
	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	robotsevaluatorV1 "github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/database"
	testcontainersupport "github.com/NationalLibraryOfNorway/veidemann/browser-controller/internal/testcontainers"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/logwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/screenshotwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/serviceconnections"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/testutil"
	logServiceTestUtil "github.com/NationalLibraryOfNorway/veidemann/log-service/pkg/testutil"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/recorderproxy"
	proxyServiceConnections "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/serviceconnections"
	proxyTestUtil "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

var (
	// sessions is a registry of sessions
	sessions *session.Registry

	// localhost is the ip address of the host machine
	localhost = GetOutboundIP().String()

	fixtureSiteBaseURL string

	// provider is a flag to select container provider
	provider = flag.String("provider", "docker", "container provider, \"docker\" or \"podman\".")
)

const (
	logServicePort        = 5002
	browserControllerPort = 7777
	proxyPort             = 6666
	maxSessions           = 2
)

func TestMain(m *testing.M) {
	// Parse flags
	flag.Parse()

	// setup browser
	ctx, cancelBrowser := context.WithCancel(context.Background())
	defer cancelBrowser()
	browserHost, browserPort, err := setupBrowser(ctx)
	if err != nil {
		panic(err)
	}

	// setup local fixture site
	fixtureSite, err := setupFixtureSite(ctx)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = fixtureSite.Close()
	}()
	fixtureSiteBaseURL = fixtureSite.baseURL

	// setup database mock
	dbMock := setupDbMock()
	dbAdapter := database.NewConfigAdapter(dbMock.RethinkDbConnection)

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
	logServiceMock := logServiceTestUtil.NewLogServiceMock(logServicePort)

	// setup writer client
	logWriter := logwriter.New(
		serviceconnections.WithPort(logServicePort),
	)
	if err := logWriter.Connect(); err != nil {
		panic(err)
	}

	// setup sessions
	sessions = session.NewRegistry(
		maxSessions,
		session.WithBrowserHost(browserHost),
		session.WithBrowserPort(browserPort),
		session.WithProxyHost(localhost),
		session.WithProxyPort(proxyPort),
		session.WithConfigAdapter(dbAdapter),
		session.WithScreenshotWriter(screenShotWriter),
		session.WithLogWriter(logWriter),
	)

	// setup robots evaluator mock
	robotsEvaluator := &testutil.RobotsEvaluatorMock{IsAllowedFunc: func(_ *robotsevaluatorV1.IsAllowedRequest) bool {
		return true
	}}

	// setup browsercontroller server
	browsercontrollerAdress := fmt.Sprintf(":%d", browserControllerPort)
	listener, err := net.Listen("tcp", browsercontrollerAdress)
	if err != nil {
		panic(err)
	}
	grpcServer := grpc.NewServer()
	apiServer := NewApiServer(sessions, robotsEvaluator, logWriter)
	browsercontrollerV2.RegisterBrowserControllerServer(grpcServer, apiServer)

	go func() {
		err := grpcServer.Serve(listener)
		if err != nil {
			if !errors.Is(err, grpc.ErrServerStopped) {
				panic(err)
			}
		}
	}()

	// setup recorder proxy
	opt := proxyTestUtil.WithExternalBrowserController(
		proxyServiceConnections.NewConnectionOptions("BrowserController",
			proxyServiceConnections.WithPort(fmt.Sprintf("%d", browserControllerPort)),
		),
	)
	grpcServices := proxyTestUtil.NewGrpcServiceMock(opt)
	recorderProxy0 := localRecorderProxy(0, grpcServices.ClientConn, "")
	recorderProxy1 := localRecorderProxy(1, grpcServices.ClientConn, "")
	recorderProxy2 := localRecorderProxy(2, grpcServices.ClientConn, "")

	// Run the tests
	code := m.Run()

	// Clean up
	sessions.Close()
	grpcServer.GracefulStop()
	grpcServices.Close()
	recorderProxy0.Shutdown(context.TODO())
	recorderProxy1.Shutdown(context.TODO())
	recorderProxy2.Shutdown(context.TODO())
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
			Extra:            &configV1.ExtraConfig{CreateScreenshot: false},
		}},
	}

	tests := []struct {
		name             string
		baseURL          string
		path             string
		expectedOutlinks []string
		skipReason       string
	}{
		{"static-outlink", fixtureSiteBaseURL, "/index.html", []string{fixtureSiteBaseURL + "/linked.html"}, ""},
		{
			"worker-outlink",
			fixtureSiteBaseURL,
			"/worker.html",
			[]string{fixtureSiteBaseURL + "/worker-hit.html"},
			"",
		},
	}
	for _, tt := range tests {
		ctx := context.Background()
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}
			if tt.baseURL == "" {
				t.Fatal("fixture site base URL was not initialized")
			}

			qUri := &frontierV1.QueuedUri{Uri: tt.baseURL + tt.path, DiscoveryPath: "L", JobExecutionId: "jid", ExecutionId: "eid"}

			s, err := sessions.GetNextAvailable(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer sessions.Release(s)

			t.Logf("Acquired session: %v", s.Id)

			// Fetch page
			phs := &frontierV1.PageHarvestSpec{
				QueuedUri:    qUri,
				CrawlConfig:  conf,
				SessionToken: "test",
			}

			t.Logf("Starting fetch test for %v", phs)

			result, err := s.Fetch(context.Background(), phs)
			t.Log("Session.Fetch returned")
			if err != nil {
				t.Fatal(err)
			}
			if result.UriCount == 0 {
				t.Fatalf("expected at least one resource for %s", qUri.Uri)
			}
			for _, want := range tt.expectedOutlinks {
				if !hasOutlink(result.Outlinks, want) {
					t.Fatalf("missing outlink %q, got %v", want, outlinkStrings(result.Outlinks))
				}
			}
			t.Logf("Resource count: %v, outlinks: %v, Time: %v\n", result.UriCount, outlinkStrings(result.Outlinks), result.PageFetchTimeMs)
		})
	}
}

type fixtureSite struct {
	baseURL string
	ctx     context.Context
	ctr     testcontainers.Container
}

func (s *fixtureSite) Close() error {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()
	return s.ctr.Terminate(ctx)
}

func setupFixtureSite(ctx context.Context) (*fixtureSite, error) {
	root, err := fixtureSiteRoot()
	if err != nil {
		return nil, err
	}

	providerType, skipReaper := containerProvider()
	fixtureContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ProviderType: providerType,
		ContainerRequest: testcontainers.ContainerRequest{
			SkipReaper:   skipReaper,
			Image:        "nginx:1.27-alpine",
			ExposedPorts: []string{"80/tcp"},
			WaitingFor:   wait.ForListeningPort("80/tcp"),
			Files:        fixtureSiteFiles(root),
		},
		Started: true,
	})
	if err != nil {
		return nil, err
	}

	host, err := fixtureContainer.Host(ctx)
	if err != nil {
		_ = fixtureContainer.Terminate(ctx)
		return nil, err
	}
	if host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}

	port, err := fixtureContainer.MappedPort(ctx, "80/tcp")
	if err != nil {
		_ = fixtureContainer.Terminate(ctx)
		return nil, err
	}

	return &fixtureSite{
		baseURL: fmt.Sprintf("http://%s:%d", host, port.Num()),
		ctx:     ctx,
		ctr:     fixtureContainer,
	}, nil
}

func fixtureSiteRoot() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed resolving server test path")
	}
	return filepath.Join(filepath.Dir(currentFile), "testdata", "site"), nil
}

func fixtureSiteFiles(root string) []testcontainers.ContainerFile {
	files := []string{
		"index.html",
		"linked.html",
		"worker.html",
		"worker.js",
		"worker-data.json",
		"worker-hit.html",
	}

	result := make([]testcontainers.ContainerFile, 0, len(files))
	for _, fileName := range files {
		result = append(result, testcontainers.ContainerFile{
			HostFilePath:      filepath.Join(root, fileName),
			ContainerFilePath: filepath.Join("/usr/share/nginx/html", fileName),
			FileMode:          0o644,
		})
	}
	return result
}

func containerProvider() (testcontainers.ProviderType, bool) {
	if *provider == "podman" {
		return testcontainers.ProviderPodman, true
	}
	return testcontainers.ProviderDocker, false
}

func hasOutlink(outlinks []*frontierV1.QueuedUri, want string) bool {
	for _, outlink := range outlinks {
		if outlink.GetUri() == want {
			return true
		}
	}
	return false
}

func outlinkStrings(outlinks []*frontierV1.QueuedUri) []string {
	result := make([]string, 0, len(outlinks))
	for _, outlink := range outlinks {
		result = append(result, outlink.GetUri())
	}
	return result
}

func setupDbMock() *database.MockConnection {
	dbConn := database.NewMockConnection()
	dbConn.GetMock().On(r.Table("config").Get("browserConfig1")).Return(
		map[string]any{
			"id":   "browserConfig1",
			"kind": "browserConfig",
			"meta": map[string]any{
				"name":    "browser config 1",
				"label":   []map[string]any{{"key": "foo", "value": "bar"}},
				"created": "2020-04-06T18:17:50.343827619Z",
			},
			"browserConfig": map[string]any{
				"windowWidth":         1400,
				"windowHeight":        1280,
				"maxInactivityTimeMs": 5000,
				"pageLoadTimeoutMs":   60000,
				"scriptRef":           []map[string]any{{"kind": "browserScript", "id": "script1"}},
			},
		},
		nil,
	)
	dbConn.GetMock().On(r.Table("config").Get("script1")).Return(
		map[string]any{
			"id":   "script1",
			"kind": "browserScript",
			"meta": map[string]any{
				"name":        "script1",
				"description": "script1",
				"label":       []map[string]any{{"key": "type", "value": "extract_outlinks"}},
			},
			"browserScript": map[string]any{
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
		map[string]any{
			"id":   "politenessConfig1",
			"kind": "politenessConfig",
			"meta": map[string]any{
				"name":    "politeness config 1",
				"label":   []map[string]any{{"key": "foo", "value": "bar"}},
				"created": "2020-04-06T18:17:50.343827619Z",
			},
			"politenessConfig": map[string]any{},
		}, nil)
	dbConn.GetMock().On(r.Table("page_log").Insert(r.MockAnything())).Return(map[string]any{}, nil)
	dbConn.GetMock().On(r.Table("crawl_log").Insert(r.MockAnything())).Return(map[string]any{}, nil)

	return dbConn
}

// localRecorderProxy creates a new recorderproxy which uses internal transport
func localRecorderProxy(id int, conn *proxyServiceConnections.Connections, nextProxyAddr string) *recorderproxy.RecorderProxy {
	proxy := recorderproxy.NewRecorderProxy(id, conn, nextProxyAddr)

	ln, err := proxy.Listen("", proxyPort)
	if err != nil {
		panic(err)
	}
	go proxy.Serve(ln)

	return proxy
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
	providerType, skipReaper := containerProvider()

	// Start browserless container
	browserless, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ProviderType: providerType,
		ContainerRequest: testcontainers.ContainerRequest{
			SkipReaper: skipReaper,
			Env: map[string]string{
				"DEBUG": "*",
			},
			Image:        testcontainersupport.BrowserlessChromium,
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
	if host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	browserPort, err := browserless.MappedPort(ctx, "3000/tcp")
	if err != nil {
		return
	}
	port = int(browserPort.Num())
	return
}
