package utils

import (
	"context"
	"net/http"
	"time"

	"gitlab.com/etke.cc/mrs/api/version"
)

// DefaultTimeout for http requests
const DefaultTimeout = 120 * time.Second

// HTTPClient with timeout
var HTTPClient = &http.Client{Timeout: DefaultTimeout}

// Get performs HTTP GET request with timeout and User-Agent set
func Get(uri string, optionalTimeout ...time.Duration) (*http.Response, error) {
	timeout := DefaultTimeout
	if len(optionalTimeout) > 0 {
		timeout = optionalTimeout[0]
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", version.UserAgent)
	return HTTPClient.Do(req)
}
