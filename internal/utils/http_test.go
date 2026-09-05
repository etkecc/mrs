package utils

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/etkecc/go-kit/httpclient"
)

// checkRedirect gates on the origin request (via[0]); same destination, different origin flips the decision.
func TestCheckRedirect_followsWellKnownOriginBlocksElse(t *testing.T) {
	dest, err := http.NewRequest(http.MethodGet, "https://b.example/anything", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	wkOrigin, err := http.NewRequest(http.MethodGet, "https://a.example/.well-known/matrix/server", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	if rerr := checkRedirect(dest, []*http.Request{wkOrigin}); rerr != nil {
		t.Errorf("a redirect on a well-known fetch must be followed, got %v", rerr)
	}

	fedOrigin, err := http.NewRequest(http.MethodGet, "https://a.example/_matrix/federation/v1/version", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	if rerr := checkRedirect(dest, []*http.Request{fedOrigin}); !errors.Is(rerr, http.ErrUseLastResponse) {
		t.Errorf("a redirect off a federation request must be blocked, got %v", rerr)
	}
}

// pins a dial straight at a live loopback listener; asserts the guard-refused outcome, never the exact error text.
func TestSharedClient_pinnedPrivateDialRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	_, port, err := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}

	ctx := httpclient.WithDialIP(context.Background(), "127.0.0.1")
	resp, err := Get(ctx, "http://delegated.invalid:"+port+"/")
	if err == nil {
		resp.Body.Close()
		t.Fatal("a dial pinned to a live loopback listener succeeded through the shared client; the guard is not on it")
	}
}
