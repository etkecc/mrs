package data

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/etkecc/mrs/internal/model"
)

// a deferred stub must carry OnlineAt, or the same-run prune eats it.
func TestBatchServers_StubCarriesOnlineAt(t *testing.T) {
	d, err := New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer d.Close()

	ctx := context.Background()
	if err := d.BatchServers(ctx, []string{"harvested.example"}); err != nil {
		t.Fatalf("BatchServers: %v", err)
	}

	srv, err := d.GetServerInfo(ctx, "harvested.example")
	if err != nil {
		t.Fatalf("GetServerInfo: %v", err)
	}
	if srv == nil {
		t.Fatal("stub was not persisted")
	}
	if srv.OnlineAt.IsZero() {
		t.Fatal("stub OnlineAt is zero: it would be deleted the same run by removeOldOfflineServers")
	}

	// mirror the removeOldOfflineServers predicate: a fresh stub must NOT be selected for prune.
	threshold := time.Now().UTC().AddDate(0, -1, 0)
	if !srv.Online && srv.OnlineAt.Before(threshold) {
		t.Fatal("fresh stub is selected by the 30d offline prune filter")
	}
}

// the two-clock invariant guard: marking a server offline stamps CheckedAt (backoff) but must leave OnlineAt (prune) put.
// if OnlineAt ever moves here, the 30d prune clock resets on every offline dial and dead servers become immortal.
func TestMarkServersOffline_StampsCheckedAtLeavesOnlineAt(t *testing.T) {
	d, err := New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer d.Close()

	ctx := context.Background()
	onlineAt := time.Now().UTC().Add(-48 * time.Hour)
	if err := d.AddServer(ctx, &model.MatrixServer{Name: "dead.example", Online: true, OnlineAt: onlineAt}); err != nil {
		t.Fatalf("AddServer: %v", err)
	}

	d.MarkServersOffline(ctx, []string{"dead.example"})

	srv, err := d.GetServerInfo(ctx, "dead.example")
	if err != nil {
		t.Fatalf("GetServerInfo: %v", err)
	}
	if srv == nil {
		t.Fatal("server vanished after MarkServersOffline")
	}
	if !srv.OnlineAt.Equal(onlineAt) {
		t.Errorf("OnlineAt moved: prune clock reset, dead servers go immortal. was %v, now %v", onlineAt, srv.OnlineAt)
	}
	if srv.CheckedAt.IsZero() {
		t.Error("CheckedAt left zero on the offline dial: backoff never engages")
	}
	if srv.Online {
		t.Error("Online still true after MarkServersOffline")
	}
}

// first-contact-offline (no prior record) must still get an OnlineAt, or the fresh stub lands in year-1 and the
// same-run prune buries it on sight. discoverServer no longer persists the offline case, so markServerOffline is
// the only writer that can save a resolves-but-down server on first sighting.
func TestMarkServersOffline_FreshStubGetsOnlineAt(t *testing.T) {
	d, err := New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer d.Close()

	ctx := context.Background()
	d.MarkServersOffline(ctx, []string{"firstcontact.example"}) // no prior AddServer

	srv, err := d.GetServerInfo(ctx, "firstcontact.example")
	if err != nil {
		t.Fatalf("GetServerInfo: %v", err)
	}
	if srv == nil {
		t.Fatal("fresh offline stub was not persisted")
	}
	if srv.OnlineAt.IsZero() {
		t.Fatal("fresh offline stub has zero OnlineAt: the same-run prune deletes it on arrival")
	}

	// mirror the removeOldOfflineServers predicate: the fresh stub must NOT be selected for prune.
	threshold := time.Now().UTC().AddDate(0, -1, 0)
	if !srv.Online && srv.OnlineAt.Before(threshold) {
		t.Fatal("fresh offline stub is selected by the 30d prune filter")
	}
	if srv.CheckedAt.IsZero() {
		t.Error("fresh offline stub has zero CheckedAt")
	}
}
