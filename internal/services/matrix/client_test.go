package matrix

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

// TestRoomSummaryFallback_rejectsIPLiteralVia checks the early reject catches the canonical IP-literal forms
// of via before any resolve or dial, and never caches them. Disguised forms (decimal/octal/zone) pass this
// check by design and are refused later by the shared dial guard, which is the real authority.
func TestRoomSummaryFallback_rejectsIPLiteralVia(t *testing.T) {
	s := &Server{curlsCache: expirable.NewLRU[string, string](100, nil, time.Hour)}
	for _, via := range []string{"127.0.0.1", "10.0.0.1:8448", "[::1]", "[::1]:8448", "[2001:db8::1]", "2001:db8::1"} {
		if room := s.roomSummaryFallback(context.Background(), "!r:example.org", via); room != nil {
			t.Errorf("via=%q: an IP literal must be refused, got a room", via)
		}
		if _, ok := s.curlsCache.Get(via); ok {
			t.Errorf("via=%q: a refused IP literal must never be cached", via)
		}
	}
}

// TestRoomSummaryFallback_hostnameViaDoesNotPoisonCache checks the via path resolves through the uncached
// resolveCSURL, so an attacker-supplied hostname never writes the shared curlsCache. .invalid never resolves,
// so the dial fails; the assertion is only that resolution left the cache empty.
func TestRoomSummaryFallback_hostnameViaDoesNotPoisonCache(t *testing.T) {
	s := &Server{curlsCache: expirable.NewLRU[string, string](100, nil, time.Hour)}
	via := "attacker.invalid"

	_ = s.roomSummaryFallback(context.Background(), "!r:example.org", via)

	if _, ok := s.curlsCache.Get(via); ok {
		t.Errorf("a user-supplied via must never populate curlsCache, but %q was cached", via)
	}
}

// TestSummaryEndpoint_neutralizesHostileBase checks a hostile or oddly-shaped base_url can never move the
// request off its own host or swallow the appended summary path: the host stays the base host, the MSC3266
// path always lands in the path, and via is set, whether the base carries a #fragment, a query, a trailing
// slash, or a legitimate path prefix.
func TestSummaryEndpoint_neutralizesHostileBase(t *testing.T) {
	const alias = "!r:example.org"
	var s Server
	cases := []struct {
		name string
		base string
	}{
		{"plain", "https://victim.example"},
		{"trailing_slash", "https://victim.example/"},
		{"path_prefix", "https://victim.example/matrix"},
		{"fragment_smuggle", "https://victim.example/exact/target#"},
		{"query_smuggle", "https://victim.example/exact/target?x=y"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := s.summaryEndpoint(tc.base, alias)
			if err != nil {
				t.Fatalf("summaryEndpoint(%q): %v", tc.base, err)
			}
			u, err := url.Parse(got)
			if err != nil {
				t.Fatalf("result is not a URL: %v", err)
			}
			if u.Host != "victim.example" {
				t.Errorf("host must stay the base host, got %q from base %q", u.Host, tc.base)
			}
			if !strings.Contains(u.Path, "/im.nheko.summary/summary/"+alias) {
				t.Errorf("summary path must be appended into the path, got %q from base %q", u.Path, tc.base)
			}
			if u.Query().Get("via") != "example.org" {
				t.Errorf("via must be set, got query %q from base %q", u.RawQuery, tc.base)
			}
		})
	}
}
