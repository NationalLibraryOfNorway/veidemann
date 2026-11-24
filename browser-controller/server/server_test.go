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
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/database"
	bcerrors "github.com/NationalLibraryOfNorway/veidemann/browser-controller/errors"
	testcontainersupport "github.com/NationalLibraryOfNorway/veidemann/browser-controller/internal/testcontainers"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/screenshotwriter"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/testutil"
	"github.com/NationalLibraryOfNorway/veidemann/log-service/pkg/logservice"
	logServiceTestUtil "github.com/NationalLibraryOfNorway/veidemann/log-service/pkg/testutil"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/mitmcert"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/recorderproxy"
	proxyTestUtil "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

var (
	sessionOptions []session.Option

	// localhost is the ip address of the host machine
	localhost = func() string {
		localIP, _ := getOutboundIP("8.8.8.8:80")
		return localIP.String()
	}()

	fixtureSiteHTTPBaseURL  string
	fixtureSiteHTTPSBaseURL string
	recordedPages           *fixtureLogWriter

	// provider is a flag to select container provider
	provider                = flag.String("provider", "docker", "container provider, \"docker\" or \"podman\".")
	supplementalHarvestURLs = flag.String("harvest-urls", "", "Comma-separated URLs for supplementary live browser/recorder harvests")
)

const (
	logServicePort = 5002
	proxyPort      = 6666
	maxSessions    = 2
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
	fixtureSiteHTTPBaseURL = fixtureSite.httpBaseURL
	fixtureSiteHTTPSBaseURL = fixtureSite.httpsBaseURL

	// setup database mock
	dbMock := setupDbMock()
	dbAdapter := database.NewConfigAdapter(dbMock.RethinkDbConnection)

	// setup screenshot writer mock
	screenShotWriter := &testutil.ScreenshotWriterMock{
		WriteFunc: func(data []byte, metadata screenshotwriter.Metadata) error {
			if strings.Contains(metadata.CrawlLog.GetRequestedUri(), "fail-screenshot") {
				return fmt.Errorf("injected screenshot storage failure")
			}
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
	logServiceConn, err := grpc.NewClient(fmt.Sprintf("localhost:%d", logServicePort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	logWriter := &logservice.LogWriter{LogClient: logV1.NewLogClient(logServiceConn)}

	// Share one recorder identity with all proxy instances and scope Chromium's
	// certificate exception to the public key they actually serve.
	certPEM, keyPEM, err := mitmcert.Generate(time.Now())
	if err != nil {
		panic(err)
	}
	recorderIdentity, err := mitmcert.ParseIdentity(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	recorderSPKI := sha256.Sum256(recorderIdentity.Certificate().RawSubjectPublicKeyInfo)

	controlConn, err := grpc.NewClient("localhost:7780", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer controlConn.Close()
	// setup sessions
	recordedPages = &fixtureLogWriter{Writer: logWriter}
	sessionOptions = []session.Option{
		session.WithBrowserHost(browserHost),
		session.WithBrowserPort(browserPort),
		session.WithProxyHost(localhost),
		session.WithProxyPort(proxyPort),
		session.WithRecorderCertificateSPKI(base64.StdEncoding.EncodeToString(recorderSPKI[:])),
		session.WithConfigAdapter(dbAdapter),
		session.WithScreenshotWriter(screenShotWriter),
		session.WithRecorder(recorderV1.NewRecorderControlClient(controlConn)),
	}

	// Browser harvests use only recorder-owned control. No callback server.
	grpcServices := proxyTestUtil.NewGrpcServiceMock()
	recorderProxy1 := localRecorderProxy(1, grpcServices, "", recorderIdentity)
	recorderProxy2 := localRecorderProxy(2, grpcServices, "", recorderIdentity)

	controlListener, err := net.Listen("tcp", ":7780")
	if err != nil {
		panic(err)
	}
	controlServer := grpc.NewServer()
	recorderV1.RegisterRecorderControlServer(controlServer, recorderproxy.NewControl(recordedPages, recorderProxy1, recorderProxy2))
	go controlServer.Serve(controlListener)
	defer controlServer.Stop()
	// Run the tests
	code := m.Run()

	// Clean up
	grpcServices.Close()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	var shutdownWg sync.WaitGroup
	for _, proxy := range []*recorderproxy.RecorderProxy{recorderProxy1, recorderProxy2} {
		shutdownWg.Add(1)
		go func() {
			defer shutdownWg.Done()
			_ = proxy.Shutdown(shutdownCtx)
		}()
	}
	shutdownWg.Wait()
	cancelShutdown()
	_ = screenShotWriter.Close()
	_ = dbMock.Close()
	_ = logServiceConn.Close()
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

	t.Run("large-capture-replay", func(t *testing.T) { testLargeCaptureWorker(t, conf) })

	type harvestScenario struct {
		name             string
		baseURL          string
		path             string
		expectedOutlinks []string
		createScreenshot bool
		expectedResource string
		expectedStatus   int32
		maxDuration      time.Duration
		skipReason       string
		browserConfig    string
		failureCode      int32
	}
	tests := []harvestScenario{
		{name: "cutoff-success", baseURL: fixtureSiteHTTPBaseURL, path: "/slow-image.html", browserConfig: "browserConfigCutoff", createScreenshot: true, expectedResource: "/slow-image.png", expectedStatus: -5011, maxDuration: 10 * time.Second},
		{name: "cutoff-no-main", baseURL: fixtureSiteHTTPBaseURL, path: "/never-main", browserConfig: "browserConfigCutoff", failureCode: -5004, maxDuration: 10 * time.Second},
		{name: "required-screenshot-failure", baseURL: fixtureSiteHTTPBaseURL, path: "/index.html?fail-screenshot", createScreenshot: true, failureCode: -5, maxDuration: 12 * time.Second},
		{name: "required-outlinks-failure", baseURL: fixtureSiteHTTPBaseURL, path: "/index.html", browserConfig: "browserConfigBadScript", failureCode: -5, maxDuration: 12 * time.Second},
		{name: "crawl-log-persistence-failure", baseURL: fixtureSiteHTTPBaseURL, path: "/index.html?fail-crawl-log", failureCode: -5, maxDuration: 12 * time.Second},
		{name: "page-log-persistence-failure", baseURL: fixtureSiteHTTPBaseURL, path: "/index.html?fail-page-log", failureCode: -5, maxDuration: 12 * time.Second},
		{name: "http-static-outlink", baseURL: fixtureSiteHTTPBaseURL, path: "/index.html", expectedOutlinks: []string{fixtureSiteHTTPBaseURL + "/linked.html"}},
		{name: "client-redirect", baseURL: fixtureSiteHTTPBaseURL, path: "/redirect.html", expectedOutlinks: []string{fixtureSiteHTTPBaseURL + "/linked.html"}, maxDuration: 12 * time.Second},
		{name: "http-fragment", baseURL: fixtureSiteHTTPBaseURL, path: "/index.html#section", expectedOutlinks: []string{fixtureSiteHTTPBaseURL + "/linked.html"}},
		{name: "https-static-outlink", baseURL: fixtureSiteHTTPSBaseURL, path: "/index.html", expectedOutlinks: []string{fixtureSiteHTTPSBaseURL + "/linked.html"}},
		{name: "http-screenshot", baseURL: fixtureSiteHTTPBaseURL, path: "/index.html", expectedOutlinks: []string{fixtureSiteHTTPBaseURL + "/linked.html"}, createScreenshot: true, maxDuration: 20 * time.Second},
		{name: "http-eventsource-cutoff", baseURL: fixtureSiteHTTPBaseURL, path: "/eventsource.html", expectedResource: "/events", expectedStatus: -5011, maxDuration: 20 * time.Second},
		{name: "finite-media", baseURL: fixtureSiteHTTPBaseURL, path: "/finite-media.html", expectedResource: "/finite.wav", maxDuration: 20 * time.Second},
		{name: "range-media", baseURL: fixtureSiteHTTPBaseURL, path: "/range-media.html", expectedResource: "/finite.wav?range", expectedStatus: 206, maxDuration: 20 * time.Second},
		{name: "media-cancellation-reload", baseURL: fixtureSiteHTTPBaseURL, path: "/reloading-media.html", expectedResource: "/slow-media.wav", expectedStatus: -5011, createScreenshot: true, maxDuration: 20 * time.Second},
		{name: "iframe-slow-resource", baseURL: fixtureSiteHTTPBaseURL, path: "/iframe.html", expectedResource: "/slow-image.png", expectedStatus: 200, maxDuration: 30 * time.Second},
		{name: "slow-image-progress", baseURL: fixtureSiteHTTPBaseURL, path: "/slow-image.html", expectedResource: "/slow-image.png", expectedStatus: 200, maxDuration: 30 * time.Second},
		{
			name:             "http-worker-outlink",
			baseURL:          fixtureSiteHTTPBaseURL,
			path:             "/worker.html",
			expectedOutlinks: []string{fixtureSiteHTTPBaseURL + "/worker-hit.html"},
		},
	}
	for i, rawURL := range strings.Split(*supplementalHarvestURLs, ",") {
		if rawURL = strings.TrimSpace(rawURL); rawURL == "" {
			continue
		}
		tests = append(tests, harvestScenario{
			name: fmt.Sprintf("live-%d", i+1), baseURL: rawURL,
			browserConfig: "browserConfig1", createScreenshot: true, maxDuration: 120 * time.Second,
		})
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}
			if tt.baseURL == "" {
				t.Fatal("fixture site base URL was not initialized")
			}

			qUri := &frontierV1.QueuedUri{Uri: tt.baseURL + tt.path, DiscoveryPath: "L", JobExecutionId: "jid", ExecutionId: "eid"}

			s := session.New(1+i%maxSessions, sessionOptions...)

			t.Logf("Acquired session: %v", s.Id)

			// Fetch page
			conf.GetCrawlConfig().BrowserConfigRef.Id = "browserConfig1"
			if tt.browserConfig != "" {
				conf.GetCrawlConfig().BrowserConfigRef.Id = tt.browserConfig
			}
			conf.GetCrawlConfig().GetExtra().CreateScreenshot = tt.createScreenshot
			phs := &frontierV1.PageHarvestSpec{
				QueuedUri:    qUri,
				CrawlConfig:  conf,
				SessionToken: "test",
			}

			t.Logf("Starting fetch test for %v", phs)

			fetchStart := time.Now()
			result, err := s.Fetch(context.Background(), phs)
			fetchDuration := time.Since(fetchStart)
			t.Log("Session.Fetch returned")
			if tt.failureCode != 0 {
				if tt.name == "required-screenshot-failure" && recordedPages.latest(qUri.Uri) == nil {
					t.Fatal("screenshot failure lost page log")
				}
				if err == nil || bcerrors.CommonsError(err).Code != tt.failureCode {
					t.Fatalf("error=%v want code=%d", err, tt.failureCode)
				}
				if tt.maxDuration > 0 && fetchDuration > tt.maxDuration {
					t.Fatalf("unbounded failed fetch: %v", fetchDuration)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if page := recordedPages.latest(qUri.Uri); page != nil && !strings.HasPrefix(tt.name, "live-") {
				for _, resource := range page.Resource {
					if !strings.HasPrefix(resource.Uri, fixtureSiteHTTPBaseURL) && !strings.HasPrefix(resource.Uri, fixtureSiteHTTPSBaseURL) {
						t.Fatalf("browser background traffic contaminated attempt: %s", resource.Uri)
					}
				}
			}
			if strings.HasPrefix(tt.name, "live-") {
				pageLog := recordedPages.latest(qUri.Uri)
				if pageLog == nil || pageLog.GetWarcId() == "" {
					t.Fatal("missing live main recording/page log")
				}
				if len(result.Outlinks) == 0 {
					t.Fatal("live harvest extracted no links")
				}
				image, e := os.ReadFile("screenshot.png")
				if e != nil || len(image) == 0 {
					t.Fatalf("missing live screenshot: %v", e)
				}
			}
			if result.UriCount == 0 {
				t.Fatalf("expected at least one resource for %s", qUri.Uri)
			}
			if tt.maxDuration > 0 && fetchDuration > tt.maxDuration {
				t.Fatalf("fetch duration = %v, want at most %v", fetchDuration, tt.maxDuration)
			}
			for _, want := range tt.expectedOutlinks {
				if !hasOutlink(result.Outlinks, want) {
					t.Fatalf("missing outlink %q, got %v", want, outlinkStrings(result.Outlinks))
				}
			}
			if tt.expectedResource != "" {
				pageLog := recordedPages.latest(qUri.Uri)
				if pageLog == nil {
					t.Fatal("missing recorded page log")
				}
				found := false
				for _, resource := range pageLog.GetResource() {
					if !strings.HasPrefix(resource.GetUri(), tt.baseURL+tt.expectedResource) {
						continue
					}
					status := resource.GetStatusCode()
					if tt.expectedStatus != 0 && status != tt.expectedStatus {
						continue
					}
					if tt.expectedStatus == 0 && status != 200 && status != 206 {
						continue
					}
					if status > 0 && resource.GetWarcId() == "" {
						t.Fatal("successful media/image has no archival result")
					}
					if status == -5011 && resource.GetWarcId() != "" {
						t.Fatal("cancellation claims an archival result")
					}
					found = true
				}
				if !found {
					t.Fatalf("no recorded outcome for %s with status %d: %v", tt.expectedResource, tt.expectedStatus, pageLog.GetResource())
				}
			}
			t.Logf("Resource count: %v, outlinks: %v, Time: %v\n", result.UriCount, outlinkStrings(result.Outlinks), result.PageFetchTimeMs)
		})
	}
}

type fixtureSite struct {
	httpBaseURL  string
	httpsBaseURL string
	ctx          context.Context
	ctr          testcontainers.Container
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
	certDir, err := os.MkdirTemp("", "veidemann-fixture-certs-")
	if err != nil {
		return nil, fmt.Errorf("create fixture certificate directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(certDir)
	}()

	certPath, keyPath, err := writeFixtureCertFiles(certDir)
	if err != nil {
		return nil, fmt.Errorf("create fixture certificate: %w", err)
	}
	if err := writeMediaFixtures(certDir); err != nil {
		return nil, err
	}

	providerType, skipReaper := containerProvider()
	fixtureContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ProviderType: providerType,
		ContainerRequest: testcontainers.ContainerRequest{
			SkipReaper:     skipReaper,
			FromDockerfile: testcontainers.FromDockerfile{Context: root},
			ExposedPorts:   []string{"8080/tcp", "8443/tcp"},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("8080/tcp"),
				wait.ForListeningPort("8443/tcp"),
			),
			Files: fixtureSiteFiles(certPath, keyPath),
		},
		Started: true,
	})
	if err != nil {
		return nil, err
	}

	httpPort, err := fixtureContainer.MappedPort(ctx, "8080/tcp")
	if err != nil {
		_ = fixtureContainer.Terminate(ctx)
		return nil, err
	}
	httpsPort, err := fixtureContainer.MappedPort(ctx, "8443/tcp")
	if err != nil {
		_ = fixtureContainer.Terminate(ctx)
		return nil, err
	}

	return &fixtureSite{
		httpBaseURL:  fmt.Sprintf("http://%s:%d", localhost, httpPort.Num()),
		httpsBaseURL: fmt.Sprintf("https://localhost:%d", httpsPort.Num()),
		ctx:          ctx,
		ctr:          fixtureContainer,
	}, nil
}

func fixtureSiteRoot() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed resolving server test path")
	}
	return filepath.Join(filepath.Dir(currentFile), "testdata"), nil
}

func fixtureSiteFiles(certPath, keyPath string) []testcontainers.ContainerFile {
	result := []testcontainers.ContainerFile{
		{
			HostFilePath:      certPath,
			ContainerFilePath: "/etc/nginx/certs/localhost.crt",
			FileMode:          0o644,
		},
		{
			HostFilePath:      keyPath,
			ContainerFilePath: "/etc/nginx/certs/localhost.key",
			FileMode:          0o444, // The image user reads this root-owned, per-test key.
		},
	}
	for _, fileName := range []string{"events.txt", "finite.wav", "slow-image.png"} {
		result = append(result, testcontainers.ContainerFile{
			HostFilePath:      filepath.Join(filepath.Dir(certPath), fileName),
			ContainerFilePath: filepath.Join("/usr/share/nginx/html", fileName),
			FileMode:          0o644,
		})
	}
	return result
}

// Generate small, valid PCM media and an incompressible image locally. These
// exercise CDP Media/Image classification without external downloads or codecs.
func writeMediaFixtures(dir string) error {
	// Enough regular SSE data to keep the recorder's downstream writes active
	// until browser closure, without letting EOF finish the request first.
	events := append([]byte("data: connected\n\n"), bytes.Repeat([]byte(": streaming padding keeps the response open\n"), 4096)...)
	if err := os.WriteFile(filepath.Join(dir, "events.txt"), events, 0o644); err != nil {
		return err
	}
	const dataSize = 32000 // two seconds of mono 16-bit PCM at 8 kHz
	wav := make([]byte, 44+dataSize)
	copy(wav, "RIFF")
	binary.LittleEndian.PutUint32(wav[4:], uint32(len(wav)-8))
	copy(wav[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(wav[16:], 16)
	binary.LittleEndian.PutUint16(wav[20:], 1)
	binary.LittleEndian.PutUint16(wav[22:], 1)
	binary.LittleEndian.PutUint32(wav[24:], 8000)
	binary.LittleEndian.PutUint32(wav[28:], 16000)
	binary.LittleEndian.PutUint16(wav[32:], 2)
	binary.LittleEndian.PutUint16(wav[34:], 16)
	copy(wav[36:], "data")
	binary.LittleEndian.PutUint32(wav[40:], dataSize)
	if err := os.WriteFile(filepath.Join(dir, "finite.wav"), wav, 0o644); err != nil {
		return err
	}
	img := image.NewRGBA(image.Rect(0, 0, 80, 80))
	seed := uint32(1)
	for y := 0; y < 80; y++ {
		for x := 0; x < 80; x++ {
			seed = seed*1664525 + 1013904223
			img.SetRGBA(x, y, color.RGBA{uint8(seed >> 24), uint8(seed >> 16), uint8(seed >> 8), 255})
		}
	}
	f, err := os.Create(filepath.Join(dir, "slow-image.png"))
	if err != nil {
		return err
	}
	encodeErr := png.Encode(f, img)
	closeErr := f.Close()
	return errors.Join(encodeErr, closeErr)
}

type fixtureLogWriter struct {
	Writer *logservice.LogWriter
	mu     sync.Mutex
	pages  []*logV1.PageLog
	crawls []*logV1.CrawlLog
}

func (w *fixtureLogWriter) WritePageLog(ctx context.Context, page *logV1.PageLog) error {
	if strings.Contains(page.Uri, "fail-page-log") {
		return fmt.Errorf("injected page-log persistence failure")
	}
	if err := w.Writer.WritePageLog(ctx, page); err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pages = append(w.pages, proto.Clone(page).(*logV1.PageLog))
	return nil
}

func (w *fixtureLogWriter) WriteCrawlLog(ctx context.Context, cl *logV1.CrawlLog) error {
	if strings.Contains(cl.RequestedUri, "fail-crawl-log") {
		return fmt.Errorf("injected crawl-log persistence failure")
	}
	if err := w.Writer.WriteCrawlLog(ctx, cl); err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.crawls = append(w.crawls, proto.Clone(cl).(*logV1.CrawlLog))
	return nil
}

func (w *fixtureLogWriter) latest(uri string) *logV1.PageLog {
	w.mu.Lock()
	defer w.mu.Unlock()
	for i := len(w.pages) - 1; i >= 0; i-- {
		if w.pages[i].GetUri() == uri {
			return proto.Clone(w.pages[i]).(*logV1.PageLog)
		}
	}
	return nil
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
	for _, id := range []string{"browserConfigCutoff", "browserConfigBadScript"} {
		timeout := 60000
		script := "script1"
		if id == "browserConfigCutoff" {
			timeout = 1500
		} else {
			script = "bad-script"
		}
		dbConn.GetMock().On(r.Table("config").Get(id)).Return(map[string]any{"id": id, "kind": "browserConfig", "browserConfig": map[string]any{"windowWidth": 1400, "windowHeight": 1280, "maxInactivityTimeMs": 1000, "pageLoadTimeoutMs": timeout, "scriptRef": []map[string]any{{"kind": "browserScript", "id": script}}}}, nil)
	}
	dbConn.GetMock().On(r.Table("config").Get("bad-script")).Return(map[string]any{"id": "bad-script", "kind": "browserScript", "browserScript": map[string]any{"browserScriptType": "EXTRACT_OUTLINKS", "script": "throw new Error('injected required artifact failure')"}}, nil)
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
func localRecorderProxy(id int, services *proxyTestUtil.GrpcServiceMock, nextProxyAddr string, identity *mitmcert.Identity) *recorderproxy.RecorderProxy {
	proxy := recorderproxy.NewRecorderProxy(id, services.ContentWriterClient, services.DnsResolverClient, nextProxyAddr, recorderproxy.WithMITMIdentity(identity))

	ln, err := proxy.Listen("", proxyPort)
	if err != nil {
		panic(err)
	}
	go proxy.Serve(ln)

	return proxy
}

// getOutboundIP returrns the preferred outbound ip of this machine
func getOutboundIP(dst string) (net.IP, error) {
	conn, err := net.Dial("udp4", dst)
	if err != nil {
		return nil, fmt.Errorf("dial udp %q: %w", dst, err)
	}
	defer func() {
		_ = conn.Close()
	}()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil, fmt.Errorf("unexpected local addr type: %T", conn.LocalAddr())
	}

	return localAddr.IP, nil
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

func writeFixtureCertFiles(dir string) (certPath, keyPath string, err error) {
	cert, err := generateLocalhostCert()
	if err != nil {
		return "", "", err
	}

	certPath = filepath.Join(dir, "localhost.crt")
	keyPath = filepath.Join(dir, "localhost.key")

	if err := os.WriteFile(certPath, cert.CertPEM, 0644); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(keyPath, cert.KeyPEM, 0600); err != nil {
		return "", "", err
	}

	return
}

type tlsCertPEM struct {
	CertPEM []byte
	KeyPEM  []byte
}

func generateLocalhostCert() (*tlsCertPEM, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}

	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "localhost",
		},

		NotBefore: time.Now().Add(-time.Minute),
		NotAfter:  time.Now().Add(24 * time.Hour),

		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature |
			x509.KeyUsageCertSign,

		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},

		BasicConstraintsValid: true,
		IsCA:                  true,

		DNSNames: []string{
			"localhost",
		},
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		},
	}

	der, err := x509.CreateCertificate(
		rand.Reader,
		&tmpl,
		&tmpl,
		&priv.PublicKey,
		priv,
	)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	})

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})

	return &tlsCertPEM{
		CertPEM: certPEM,
		KeyPEM:  keyPEM,
	}, nil
}
