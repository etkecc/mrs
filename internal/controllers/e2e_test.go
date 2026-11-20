package controllers

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	echobasicauth "github.com/etkecc/go-echo-basic-auth/v2"
	"github.com/labstack/echo/v5"

	"github.com/etkecc/mrs/internal/model"
)

// e2e fixture: a login that passes, a peer IP the allowlist admits, a peer it refuses and a blocked address
const (
	e2eLogin    = "admin"
	e2ePassword = "secret"

	e2ePeerIP    = "10.1.2.3" // allowlisted, unblocked
	e2eBadIP     = "9.9.9.9"  // neither allowlisted nor blocked
	e2eBlockedIP = "8.8.8.8"  // blocklist entry

	e2eHeaderForwardedFor = "X-Forwarded-For"
	e2eHeaderAuthorizaton = "Authorization"

	e2eStartTimeout = 5 * time.Second
	e2eStopTimeout  = 5 * time.Second
)

// e2eClient carries a deadline so a wedged handler fails the test and keeps redirects observable
var e2eClient = &http.Client{
	Timeout:       10 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

// logSink collects the server log lines under test; slog handlers write concurrently from request goroutines
type logSink struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write implements io.Writer
func (s *logSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

// String returns everything logged so far
func (s *logSink) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// e2eConfigService serves a fixture config with real credentials so auth can be exercised end to end
type e2eConfigService struct {
	cfg *model.Config
}

// Get returns the fixture config
func (c *e2eConfigService) Get() *model.Config {
	return c.cfg
}

// newE2EConfig builds a config with credentials, an IP allowlist and one blocked address
func newE2EConfig() *model.Config {
	cfg := &model.Config{
		Auth:      &model.ConfigAuth{},
		Matrix:    &model.ConfigMatrix{ServerName: "example.org"},
		Blocklist: &model.ConfigBlocklist{IPs: []string{e2eBlockedIP}},
	}
	auths := []*echobasicauth.Auth{
		&cfg.Auth.Admin,
		&cfg.Auth.Metrics,
		&cfg.Auth.Catalog,
		&cfg.Auth.Discovery,
		&cfg.Auth.Moderation,
	}
	for _, auth := range auths {
		auth.Login, auth.Password, auth.IPs = e2eLogin, e2ePassword, []string{e2ePeerIP, "127.0.0.1"}
	}
	return cfg
}

// startE2E boots the real router behind echo.StartConfig and returns its base URL plus the log sink
func startE2E(t *testing.T) (string, *echo.Echo, *logSink) {
	t.Helper()
	sink := &logSink{}
	e := echo.New()
	e.Logger = slog.New(slog.NewTextHandler(sink, &slog.HandlerOptions{Level: slog.LevelWarn}))
	ConfigureRouter(e, &e2eConfigService{cfg: newE2EConfig()}, nil, nil, stubCache{}, nil, nil, nil, nil, nil)

	addrs := make(chan net.Addr, 1)
	sc := echo.StartConfig{
		Address:          "127.0.0.1:0",
		HideBanner:       true,
		HidePort:         true,
		ListenerAddrFunc: func(addr net.Addr) { addrs <- addr },
	}
	ctx, cancel := context.WithCancel(t.Context())
	errCh := make(chan error, 1)
	go func() { errCh <- sc.Start(ctx, e) }()

	var addr net.Addr
	select {
	case addr = <-addrs:
	case err := <-errCh:
		cancel()
		t.Fatalf("server exited before listening: %v", err)
	case <-time.After(e2eStartTimeout):
		cancel()
		t.Fatal("timed out waiting for the listener")
	}

	t.Cleanup(func() {
		cancel()
		select {
		case err := <-errCh:
			if err != nil {
				t.Errorf("server stop: %v", err)
			}
		case <-time.After(e2eStopTimeout):
			t.Error("timed out waiting for server shutdown")
		}
	})

	return "http://" + addr.String(), e, sink
}

// do sends a request and returns the status, headers and body, closing the body itself
func do(t *testing.T, method, url string, headers map[string]string) (status int, header http.Header, body string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, url, http.NoBody)
	if err != nil {
		t.Fatalf("build request %s %s: %v", method, url, err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := e2eClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	bodyb, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("%s %s: read body: %v", method, url, err)
	}
	return resp.StatusCode, resp.Header, string(bodyb)
}

// e2eHeaders builds the header set of one request
func e2eHeaders(peer string, withCredentials bool) map[string]string {
	headers := make(map[string]string, 2)
	if peer != "" {
		headers[e2eHeaderForwardedFor] = peer
	}
	if withCredentials {
		headers[e2eHeaderAuthorizaton] = "Basic " + base64.StdEncoding.EncodeToString([]byte(e2eLogin+":"+e2ePassword))
	}
	return headers
}

// TestE2EAuth covers the basic auth middleware, the blocklist and the IP extractor over a real socket
func TestE2EAuth(t *testing.T) {
	base, _, _ := startE2E(t)

	t.Run("health answers without credentials", func(t *testing.T) {
		status, _, body := do(t, http.MethodGet, base+"/_health", nil)
		if status != http.StatusOK {
			t.Fatalf("status = %d, want %d", status, http.StatusOK)
		}
		if !strings.Contains(body, `"status":"ok"`) {
			t.Errorf("body = %q, want the health payload", body)
		}
	})

	t.Run("missing credentials get 401 and a challenge", func(t *testing.T) {
		status, header, _ := do(t, http.MethodGet, base+"/metrics", nil)
		if status != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
		}
		if challenge := header.Get(echo.HeaderWWWAuthenticate); !strings.HasPrefix(strings.ToLower(challenge), "basic realm=") {
			t.Errorf("WWW-Authenticate = %q, want a Basic realm challenge", challenge)
		}
	})

	t.Run("allowlisted peer with credentials passes", func(t *testing.T) {
		status, _, _ := do(t, http.MethodGet, base+"/metrics", e2eHeaders(e2ePeerIP, true))
		if status != http.StatusOK {
			t.Fatalf("status = %d, want %d", status, http.StatusOK)
		}
	})

	t.Run("credentials from a peer outside the allowlist get 401", func(t *testing.T) {
		status, _, _ := do(t, http.MethodGet, base+"/metrics", e2eHeaders(e2eBadIP, true))
		if status != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", status, http.StatusUnauthorized)
		}
	})

	t.Run("blocklisted peer gets 403 before auth runs", func(t *testing.T) {
		status, _, _ := do(t, http.MethodGet, base+"/metrics", e2eHeaders(e2eBlockedIP, true))
		if status != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", status, http.StatusForbidden)
		}
	})
}

// TestE2EErrors covers the status codes the custom error handler must keep from the v4 behavior
func TestE2EErrors(t *testing.T) {
	base, _, sink := startE2E(t)

	t.Run("unknown path stays 404", func(t *testing.T) {
		status, _, body := do(t, http.MethodGet, base+"/nope", nil)
		if status != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", status, http.StatusNotFound)
		}
		if !strings.Contains(body, `"message":"Not Found"`) {
			t.Errorf("body = %q, want the default not-found message", body)
		}
	})

	t.Run("wrong method stays 405", func(t *testing.T) {
		status, _, _ := do(t, http.MethodPost, base+"/robots.txt", nil)
		if status != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", status, http.StatusMethodNotAllowed)
		}
	})

	t.Run("panic answers 500 and logs at ERROR", func(t *testing.T) {
		status, _, _ := do(t, http.MethodGet, base+"/stats", nil)
		if status != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", status, http.StatusInternalServerError)
		}
		if line := sink.String(); !strings.Contains(line, "request failed") || !strings.Contains(line, "level=ERROR") {
			t.Errorf("log = %q, want an ERROR line for the failed request", line)
		}
	})

	t.Run("swagger ui serves", func(t *testing.T) {
		status, header, _ := do(t, http.MethodGet, base+"/_docs", nil)
		if status != http.StatusMovedPermanently {
			t.Fatalf("status = %d, want %d", status, http.StatusMovedPermanently)
		}
		if location := header.Get("Location"); location != "/_docs/index.html" {
			t.Fatalf("location = %q, want /_docs/index.html", location)
		}
		status, _, body := do(t, http.MethodGet, base+"/_docs/index.html", nil)
		if status != http.StatusOK {
			t.Fatalf("status = %d, want %d", status, http.StatusOK)
		}
		if !strings.Contains(body, "swagger") {
			t.Errorf("body = %q, want the swagger ui", body)
		}
	})

	t.Run("rate limiter denies the second anonymous discovery", func(t *testing.T) {
		path := base + "/discover/msc1929/example.org"
		do(t, http.MethodPost, path, nil)
		status, _, _ := do(t, http.MethodPost, path, nil)
		if status != http.StatusTooManyRequests {
			t.Fatalf("status = %d, want %d", status, http.StatusTooManyRequests)
		}
	})
}

// TestE2EMiddleware covers the global middleware v5 reshaped: CORS options handling and the gzip skipper
func TestE2EMiddleware(t *testing.T) {
	base, _, _ := startE2E(t)

	t.Run("cors preflight is answered", func(t *testing.T) {
		headers := map[string]string{
			"Origin":                        "https://example.com",
			"Access-Control-Request-Method": http.MethodGet,
		}
		status, header, _ := do(t, http.MethodOptions, base+"/search", headers)
		if status != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", status, http.StatusNoContent)
		}
		if origin := header.Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("Access-Control-Allow-Origin = %q, want *", origin)
		}
		if maxAge := header.Get("Access-Control-Max-Age"); maxAge != "86400" {
			t.Errorf("Access-Control-Max-Age = %q, want 86400", maxAge)
		}
	})

	t.Run("gzip compresses a compressible route", func(t *testing.T) {
		status, header, _ := do(t, http.MethodGet, base+"/_docs/index.html", map[string]string{"Accept-Encoding": "gzip"})
		if status != http.StatusOK {
			t.Fatalf("status = %d, want %d", status, http.StatusOK)
		}
		if encoding := header.Get("Content-Encoding"); encoding != "gzip" {
			t.Errorf("Content-Encoding = %q, want gzip", encoding)
		}
	})
}

// TestE2ERoutesResolve drives every registered route so a mis-registered path cannot hide behind the unit tests
func TestE2ERoutesResolve(t *testing.T) {
	base, e, _ := startE2E(t)

	for _, route := range e.Router().Routes() {
		if strings.HasPrefix(route.Name, "echo_route_not_found") || strings.Contains(route.Path, "*") {
			continue
		}
		method := route.Method
		if method == echo.RouteAny {
			method = http.MethodGet
		}
		path := route.Path
		for _, param := range route.Parameters {
			path = strings.ReplaceAll(path, ":"+param, "example.org")
		}
		status, _, _ := do(t, method, base+path, nil)
		if status == http.StatusNotFound || status == http.StatusMethodNotAllowed {
			t.Errorf("%s %s (registered as %s) resolves to %d", method, path, route.Path, status)
		}
	}
}
