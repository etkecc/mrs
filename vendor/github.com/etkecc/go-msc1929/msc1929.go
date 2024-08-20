package msc1929

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"time"
)

var (
	// Client will be used to request MSC1929 support file
	Client        *http.Client
	defaultDialer = &net.Dialer{
		Timeout: 5 * time.Second,
	}
	defaultClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        1,
			MaxConnsPerHost:     1,
			MaxIdleConnsPerHost: 1,
			TLSHandshakeTimeout: 10 * time.Second,
			DialContext:         defaultDialer.DialContext,
			Dial:                defaultDialer.Dial,
		},
	}
)

// getClient returns either custom client (if set) or default client (if custom is not provided)
func getClient() *http.Client {
	if Client == nil {
		return defaultClient
	}
	return Client
}

// Get MSC1929 support file from serverName
func Get(serverName string) (*Response, error) {
	return GetWithContext(context.Background(), serverName)
}

// GetWithContext MSC1929 support file from serverName
func GetWithContext(ctx context.Context, serverName string) (*Response, error) {
	endpoint := "https://" + serverName + "/.well-known/matrix/support"
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Go-MSC1929-client/1.0 (+https://gitlab.com/etke.cc/go/msc1929)")

	resp, err := getClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	datab, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return ParseMSC1929(datab)
}

// ParseMSC1929 parses MSC1929 support file
func ParseMSC1929(content []byte) (*Response, error) {
	var data *Response
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}
	if data.IsEmpty() {
		return nil, nil
	}

	return data, nil
}
