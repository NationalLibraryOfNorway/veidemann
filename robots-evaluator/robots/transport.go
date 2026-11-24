package robots

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	dnsV1 "github.com/NationalLibraryOfNorway/veidemann/api/dnsresolver/v1"
)

const maxExchangeBytes = 4 << 20

type fetchKey struct{}
type fetchMetadata struct {
	collection, execution, job string
	capture                    *refreshCapture
}

type exchange struct {
	uri, ip, mime           string
	status                  int
	started                 time.Time
	duration                time.Duration
	request, response, body []byte
}

type refreshCapture struct {
	started   time.Time
	exchanges []exchange
	skipped   int
	causes    []string
}

// Transport resolves every target through the archival DNS API. Cache headers
// exist only on the outbound Squid request, never on the captured request.
type Transport struct {
	Base  http.RoundTripper
	DNS   dnsV1.DnsResolverClient
	Cache bool
}

func NewTransport(dns dnsV1.DnsResolverClient, cacheAddress string) (*Transport, error) {
	if cacheAddress == "" {
		return nil, fmt.Errorf("cache address is required")
	}
	proxyURL, err := url.Parse("http://" + cacheAddress)
	if err != nil || proxyURL.Hostname() == "" {
		return nil, fmt.Errorf("invalid cache address %q", cacheAddress)
	}
	base := http.DefaultTransport.(*http.Transport).Clone()
	base.Proxy = http.ProxyURL(proxyURL)
	base.DisableCompression = true
	base.ForceAttemptHTTP2 = false
	base.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return &Transport{Base: base, DNS: dns, Cache: true}, nil
}

func (t *Transport) RoundTrip(req *http.Request) (response *http.Response, resultErr error) {
	meta, _ := req.Context().Value(fetchKey{}).(fetchMetadata)
	defer func() {
		if resultErr != nil && meta.capture != nil {
			meta.capture.skip("fetch: " + resultErr.Error())
		}
	}()
	if req.Method != http.MethodGet || (req.Body != nil && req.Body != http.NoBody) {
		return nil, fmt.Errorf("robots transport requires a bodyless GET")
	}
	started := time.Now().UTC()
	dnsReq := &dnsV1.ResolveRequest{Host: req.URL.Hostname(), ExecutionId: meta.execution}
	if meta.collection != "" {
		dnsReq.CollectionRef = &configV1.ConfigRef{Kind: configV1.Kind_collection, Id: meta.collection}
	}
	reply, err := t.DNS.Resolve(req.Context(), dnsReq)
	if err != nil {
		return nil, fmt.Errorf("resolve robots target: %w", err)
	}
	if reply == nil || reply.GetError() != nil || net.ParseIP(reply.GetTextualIp()) == nil {
		return nil, fmt.Errorf("negative or invalid DNS reply for %s: %v", dnsReq.Host, reply.GetError())
	}
	clean := req.Clone(req.Context())
	clean.Header.Set("Accept-Encoding", "identity")
	if clean.Header.Get("User-Agent") == "" {
		clean.Header.Set("User-Agent", "Go-http-client/1.1")
	}
	for _, h := range []string{collectionID, executionID, jobExecutionID} {
		clean.Header.Del(h)
	}

	outbound := clean.Clone(clean.Context())
	if t.Cache {
		outbound.Header.Set(executionID, meta.execution)
		outbound.Header.Set(jobExecutionID, meta.job)
	}
	resp, err := t.Base.RoundTrip(outbound)
	if err != nil {
		return nil, err
	}
	if meta.capture != nil {
		if err := meta.capture.captureResponse(clean, resp, reply.TextualIp, started); err != nil {
			meta.capture.skip("capture: " + err.Error())
		}
	}
	return resp, nil
}

func (c *refreshCapture) skip(cause string) {
	c.skipped++
	c.causes = append(c.causes, cause)
}

func (c *refreshCapture) captureResponse(req *http.Request, resp *http.Response, ip string, started time.Time) error {
	for _, cacheStatus := range resp.Header.Values("X-Cache") {
		if strings.Contains(cacheStatus, "HIT from veidemann_cache") {
			c.skip("Squid cache hit")
			return nil
		}
	}

	headers := boundedBuffer{remaining: maxExchangeBytes}
	if err := req.Write(&headers); err != nil {
		return err
	}
	requestSize := headers.Len()
	if _, err := fmt.Fprintf(&headers, "%s %s\r\n", resp.Proto, resp.Status); err != nil {
		return err
	}
	if err := resp.Header.Write(&headers); err != nil {
		return err
	}
	if _, err := headers.WriteString("\r\n"); err != nil {
		return err
	}

	resp.Body = &captureBody{
		ReadCloser: resp.Body,
		capture:    c,
		exchange: exchange{
			uri:      req.URL.String(),
			ip:       ip,
			mime:     resp.Header.Get("Content-Type"),
			status:   resp.StatusCode,
			started:  started,
			request:  headers.Bytes()[:requestSize],
			response: headers.Bytes()[requestSize:],
		},
	}
	return nil
}

// captureBody observes the HTTP client's reads without draining unread redirects.
// Only an exchange that reaches EOF is eligible for recording.
type captureBody struct {
	io.ReadCloser
	capture  *refreshCapture
	exchange exchange
	finished bool
}

func (b *captureBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if b.finished {
		return n, err
	}
	size := len(b.exchange.request) + len(b.exchange.response) + len(b.exchange.body) + n
	if size > maxExchangeBytes {
		b.finish(errCaptureLimit)
		return n, err
	}
	b.exchange.body = append(b.exchange.body, p[:n]...)
	if err != nil {
		b.finish(err)
	}
	return n, err
}

func (b *captureBody) Close() error {
	if !b.finished {
		b.finish(errors.New("body closed before EOF"))
	}
	return b.ReadCloser.Close()
}

func (b *captureBody) finish(err error) {
	if b.finished {
		return
	}
	b.finished = true
	switch err {
	case io.EOF:
		b.exchange.duration = time.Since(b.exchange.started)
		b.capture.exchanges = append(b.capture.exchanges, b.exchange)
	case errCaptureLimit:
		b.capture.skip(err.Error())
	default:
		b.capture.skip("incomplete response: " + err.Error())
	}
	b.exchange.body = nil
}

var errCaptureLimit = errors.New("exchange exceeds 4 MiB")

// boundedBuffer also bounds protocol headers; oversized origins cannot reserve
// more recording memory simply by moving material from the body to headers.
type boundedBuffer struct {
	bytes.Buffer
	remaining int
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.remaining {
		return 0, errCaptureLimit
	}
	n, err := b.Buffer.Write(p)
	b.remaining -= n
	return n, err
}

func (b *boundedBuffer) WriteString(s string) (int, error) { return b.Write([]byte(s)) }
