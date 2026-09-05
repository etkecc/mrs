package matrix

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

func newCacheTestServer() *Server {
	return &Server{
		surlsCache:  expirable.NewLRU[string, string](100, nil, time.Hour),
		discoverSem: make(chan struct{}, 1),
	}
}

func TestDcrURL_cachesThreePartValue(t *testing.T) {
	s := newCacheTestServer()
	gotURL, gotHost, gotIP := s.dcrURL(context.Background(), "example.org", "https://delegated.example:8448", "delegated.example", "1.2.3.4", false)
	if gotURL != "https://delegated.example:8448" {
		t.Errorf("url: got %q", gotURL)
	}
	if gotHost != "delegated.example" {
		t.Errorf("host: got %q", gotHost)
	}
	if gotIP != "1.2.3.4" {
		t.Errorf("dialIP: got %q", gotIP)
	}

	cached, ok := s.surlsCache.Get("example.org")
	if !ok {
		t.Fatal("cache miss after dcrURL")
	}
	if want := "https://delegated.example:8448||delegated.example||1.2.3.4"; cached != want {
		t.Errorf("cache value must be url||host||ip\n want %q\n  got %q", want, cached)
	}
}

// TestGetURL_cacheHit_delegatedHostInURL_ipNotLeaked: delegated host stays in URL/Host/SNI, dial IP never leaks.
func TestGetURL_cacheHit_delegatedHostInURL_ipNotLeaked(t *testing.T) {
	s := newCacheTestServer()
	// seed exactly as the SRV-divergent branch does: delegated host in the URL, resolved IP as the pin.
	s.surlsCache.Add("example.org", "https://delegated.example:8448||delegated.example||1.2.3.4")

	_, gotURL, gotHost := s.getURL(context.Background(), "example.org", false)
	if gotURL != "https://delegated.example:8448" {
		t.Errorf("URL must carry the delegated host, got %q", gotURL)
	}
	if gotHost != "delegated.example" {
		t.Errorf("host: got %q", gotHost)
	}
	if strings.Contains(gotURL, "1.2.3.4") {
		t.Errorf("dial IP leaked into the URL: %q", gotURL)
	}
	if strings.Contains(gotHost, "1.2.3.4") {
		t.Errorf("dial IP leaked into the Host: %q", gotHost)
	}
}

// pre-3-part entry drops and re-resolves, never misparses; explicit-port serverName keeps it network-free.
func TestGetURL_staleTwoPartEntry_treatedAsMiss(t *testing.T) {
	s := newCacheTestServer()
	s.surlsCache.Add("example.org:8448", "https://example.org:8448||example.org:8448")

	_, gotURL, gotHost := s.getURL(context.Background(), "example.org:8448", false)
	if gotURL != "https://example.org:8448" {
		t.Errorf("url: got %q", gotURL)
	}
	if gotHost != "example.org" {
		t.Errorf("dcrURL port-strips the delegated host, got %q", gotHost)
	}

	cached, ok := s.surlsCache.Get("example.org:8448")
	if !ok {
		t.Fatal("re-resolution did not re-cache")
	}
	if parts := strings.Split(cached, "||"); len(parts) != 3 {
		t.Errorf("stale entry must be re-cached in 3-part format, got %d parts: %q", len(parts), cached)
	}
}
