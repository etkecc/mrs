package utils

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/etkecc/go-apm"
	"github.com/etkecc/go-kit/httpclient"
	"github.com/etkecc/go-msc1929"

	"github.com/etkecc/mrs/internal/version"
)

const (
	// DefaultTimeout bounds the whole retry sequence; 5m waits out slow TLS on 8448 across three patient tries.
	DefaultTimeout = 300 * time.Second
)

var httpClient = newHTTPClient()

func init() {
	// use dial-guarded client for MSC1929, say no to SSRF.
	msc1929.Client = httpClient
}

// cancelOnClose cancels when the body closes; media.go streams resp.Body long after Do already returned.
type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

// newHTTPClient is a patient crawler: 60s/attempt, 30s TLS handshake, 3 retries; we ingest even the broken hosts.
func newHTTPClient() *http.Client {
	client := httpclient.NewMultiHost(
		httpclient.WithDialGuard(),
		httpclient.WithPerAttemptTimeout(60*time.Second),
		httpclient.WithTLSHandshakeTimeout(30*time.Second),
		httpclient.WithResponseHeaderTimeout(60*time.Second),
	)
	client.CheckRedirect = checkRedirect
	return client
}

// SlowHTTPClient for heavy publicRooms queries; 120s per attempt for overloaded Synapse.
var SlowHTTPClient = newSlowHTTPClient()

func newSlowHTTPClient() *http.Client {
	client := httpclient.NewMultiHost(
		httpclient.WithDialGuard(),
		httpclient.WithPerAttemptTimeout(120*time.Second),
		httpclient.WithTLSHandshakeTimeout(30*time.Second),
		httpclient.WithResponseHeaderTimeout(120*time.Second),
	)
	client.CheckRedirect = checkRedirect
	return client
}

// checkRedirect follows only a well-known fetch redirect, the one 30x the spec allows; dial guard blocks bad IPs.
func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) > 0 && strings.HasPrefix(via[0].URL.Path, "/.well-known/matrix/") {
		return nil
	}
	apm.Log(req.Context()).Debug().Str("url", req.URL.String()).Msg("blocked redirect off a non-well-known request")
	return http.ErrUseLastResponse
}

// Get performs an HTTP GET request with timeout, User-Agent, and retrier
func Get(ctx context.Context, uri string, host ...string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, http.NoBody)
	if err != nil {
		return nil, err
	}
	if len(host) > 0 && host[0] != "" {
		req.Host = host[0]
	}
	return Do(req)
}

// Do performs an HTTP request with timeout, User-Agent, and retrier
func Do(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(req.Context(), DefaultTimeout)
	req = req.WithContext(ctx)
	// don't clobber a caller-set UA (the analytics path forwards the client's real one); default only when blank.
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", version.UserAgent)
	}
	resp, err := httpClient.Do(req) //nolint:gosec // via is user input; the dial guard, not the URL string, is what refuses a private/metadata target
	if err != nil {
		cancel()
		return nil, err
	}
	resp.Body = &cancelOnClose{ReadCloser: resp.Body, cancel: cancel}
	return resp, nil
}

func (c *cancelOnClose) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()
	return err
}
