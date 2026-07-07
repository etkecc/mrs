package matrix

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

// knock on a dead server twice; the second knock hits memory, not the corpse. the tell it short-circuited is the
// error going nil: a real re-lookup of nonexistent.invalid (guaranteed NXDOMAIN, so no real egress) would hand back one.
func TestQueryServerName_negativeCacheStopsRefetch(t *testing.T) {
	ctx := context.Background()
	s := &Server{
		namesCache:         expirable.NewLRU[string, string](100, nil, time.Hour),
		namesNegativeCache: expirable.NewLRU[string, struct{}](100, nil, time.Hour),
		surlsCache:         expirable.NewLRU[string, string](100, nil, time.Hour),
		discoverSem:        make(chan struct{}, 1),
	}
	const dead = "nonexistent.invalid"

	name, err := s.QueryServerName(ctx, dead)
	if name != "" || err == nil {
		t.Fatalf("a fresh failing lookup must return (\"\", <error>), got (%q, %v)", name, err)
	}
	if _, ok := s.namesNegativeCache.Get(dead); !ok {
		t.Fatal("a failing lookup must be recorded in the negative cache")
	}

	name, err = s.QueryServerName(ctx, dead)
	if name != "" || err != nil {
		t.Fatalf("a negative-cache hit must short-circuit to (\"\", nil), got (%q, %v)", name, err)
	}
}

// a server that came back from the dead shouldn't stay buried: a live positive-cache entry outranks a stale "it's dead" note.
func TestQueryServerName_positiveCacheBeatsNegative(t *testing.T) {
	ctx := context.Background()
	s := &Server{
		namesCache:         expirable.NewLRU[string, string](100, nil, time.Hour),
		namesNegativeCache: expirable.NewLRU[string, struct{}](100, nil, time.Hour),
	}
	const name = "good.example"
	s.namesCache.Add(name, name)
	s.namesNegativeCache.Add(name, struct{}{})

	got, err := s.QueryServerName(ctx, name)
	if got != name || err != nil {
		t.Fatalf("the positive cache must win over a stale negative entry, got (%q, %v)", got, err)
	}
}
