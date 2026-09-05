package matrix

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/etkecc/mrs/internal/model"
)

// fakeVisibilityData is a minimal dataRepository driving GetClientRoomVisibility's found/not-found/banned branches.
type fakeVisibilityData struct {
	room   *model.MatrixRoom
	banned bool
}

func (f *fakeVisibilityData) GetRoom(context.Context, string) (*model.MatrixRoom, error) {
	return f.room, nil
}
func (f *fakeVisibilityData) GetRoomMapping(context.Context, string) string { return "" }
func (f *fakeVisibilityData) IsBanned(context.Context, string) bool         { return f.banned }

// TestGetClientRoomVisibility pins the spec fix: a held room is 200 public; uncrawled or banned is 404.
func TestGetClientRoomVisibility(t *testing.T) {
	const roomID = "!room:example.org"
	cases := []struct {
		name       string
		id         string
		data       *fakeVisibilityData
		wantStatus int
	}{
		{"empty id", "", &fakeVisibilityData{}, http.StatusBadRequest},
		{"unknown room", roomID, &fakeVisibilityData{room: nil}, http.StatusNotFound},
		{"banned room", roomID, &fakeVisibilityData{room: &model.MatrixRoom{ID: roomID}, banned: true}, http.StatusNotFound},
		{"indexed public room", roomID, &fakeVisibilityData{room: &model.MatrixRoom{ID: roomID}}, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Server{data: tc.data}
			status, _ := s.GetClientRoomVisibility(context.Background(), tc.id)
			if status != tc.wantStatus {
				t.Errorf("status = %d, want %d", status, tc.wantStatus)
			}
		})
	}
}

// catches IP-literal via forms pre-dial, uncached; disguised (decimal/octal/zone) forms pass, dial guard catches them.
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

// via resolves via uncached resolveCSURL; an attacker hostname never poisons curlsCache (.invalid never resolves).
func TestRoomSummaryFallback_hostnameViaDoesNotPoisonCache(t *testing.T) {
	s := &Server{curlsCache: expirable.NewLRU[string, string](100, nil, time.Hour)}
	via := "attacker.invalid"

	_ = s.roomSummaryFallback(context.Background(), "!r:example.org", via)

	if _, ok := s.curlsCache.Get(via); ok {
		t.Errorf("a user-supplied via must never populate curlsCache, but %q was cached", via)
	}
}

// a hostile/oddly-shaped base_url never moves the host or swallows the summary path: fragment, query, slash, prefix.
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
