package data

import (
	"context"
	"path/filepath"
	"testing"
	"time"
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
