package importutil

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// UriChecker checks if a uri is reachable
type UriChecker struct {
	*http.Client
}

// Check checks if a uri is reachable and returns the uri if it is reachable
// If the uri is not reachable, it returns an error
// If the uri is redirected with 301, it returns the redirected uri
func (uc *UriChecker) Check(uri string) (string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodHead, uri, nil)
	if err != nil {
		return "", err
	}
	resp, err := uc.Client.Do(req)
	if err != nil {
		var uerr *url.Error
		if errors.As(err, &uerr) && uerr.Timeout() {
			return "", fmt.Errorf("timeout")
		}
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) {
			return "", fmt.Errorf("no such host: %s", dnsErr.Name)
		}
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusMovedPermanently {
		u, err := resp.Location()
		if err != nil {
			return "", err
		}
		return u.String(), nil
	}

	if resp.StatusCode < 400 {
		return uri, nil
	}

	return uri, fmt.Errorf("%s", resp.Status)
}

// GetTitle returns the title of the uri
func (uc *UriChecker) GetTitle(uri string) string {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, uri, nil)
	if err != nil {
		return ""
	}
	resp, err := uc.Client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return ""
	}
	var title string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
			title = strings.TrimSpace(n.FirstChild.Data)
			return
		}
		if n.Type == html.ElementNode && n.Data == "body" {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return title
}
