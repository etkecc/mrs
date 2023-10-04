package postmark

import (
	"net/http"
)

// AuthTransport holds a authentication information for Postmark API.
type AuthTransport struct {
	Transport http.RoundTripper
	Token     string
}

// RoundTrip implements the RoundTripper interface.
func (t *AuthTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Postmark-Server-Token", t.Token)

	return transport.RoundTrip(req)
}
