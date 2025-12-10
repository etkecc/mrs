package utils

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/etkecc/go-apm"

	"github.com/etkecc/mrs/internal/version"
)

const (
	// DefaultTimeout for http requests
	DefaultTimeout = 120 * time.Second
	// MaxRetries for http requests
	MaxRetries = 5
	// RetryDelay for http requests
	RetryDelay = 5 * time.Second
)

// httpClient with timeout
var httpClient = apm.WrapClient(&http.Client{Timeout: DefaultTimeout}, apm.WithHealthchecks(false))

// Get performs HTTP GET request with timeout, User-Agent, and retrier
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

// Do performs HTTP request with timeout, User-Agent, and retrier
func Do(req *http.Request) (*http.Response, error) {
	// edge case: when function ends it execution and automatically calls cancel(),
	// it causes error "context canceled" when the function caller tries to read the body of the response
	// so we defer the cancel() function to be called only when there is an error
	var err error
	var resp *http.Response
	ctx, cancel := context.WithTimeout(req.Context(), DefaultTimeout)
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", version.UserAgent)
	client := httpClient
	// edge case: custom Host header is set, need to create custom Transport with ServerName
	if req.URL.Hostname() != req.Host && req.Host != "" {
		client = apm.WrapClient(&http.Client{
			Timeout: DefaultTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{ //nolint:gosec // there are _very_ different servers in the federation, so stick to defaults
					ServerName: req.Host,
				},
			},
		}, apm.WithHealthchecks(false))
		defer client.CloseIdleConnections()
	}

	resp, err = client.Do(req)
	return resp, err
}
